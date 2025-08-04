package svs

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
	spec_svs "github.com/named-data/ndnd/std/ndn/svs/v2"
	"github.com/named-data/ndnd/std/schema"
	"github.com/named-data/ndnd/std/utils"
)

type SyncState int

type MissingData struct {
	Name     enc.Name
	StartSeq uint64
	EndSeq   uint64
}

const (
	SyncSteady SyncState = iota
	SyncSuppression
)

// SvsNode implements the StateVectorSync but works for only one instance.
// Similar is RegisterPolicy. A better implementation is needed if there is
// a need that multiple producers under the same name pattern that runs on the same application instance.
// It would also be more natural if we make 1-1 mapping between MatchedNodes and SVS instances,
// instead of the Node and the SVS instance, which is against the philosophy of matching.
// Also, this sample always starts from sequence number 0.
type SvsNode struct {
	schema.BaseNodeImpl

	OnMissingData *schema.EventTarget

	SyncInterval        time.Duration
	SuppressionInterval time.Duration
	BaseMatching        enc.Matching
	ChannelSize         uint64
	SelfName            enc.Name

	dataLock        sync.Mutex
	timer           ndn.Timer
	cancelSyncTimer func() error
	missChan        chan MissingData
	stopChan        chan struct{}

	localSv   spec_svs.StateVector
	aggSv     spec_svs.StateVector
	state     SyncState
	selfSeq   uint64
	ownPrefix enc.Name
	notifNode *schema.Node
}

// Returns a string representation of the SvsNode instance as "svs-node".
func (n *SvsNode) String() string {
	return "svs-node"
}

// Returns the receiver as a `schema.NodeImpl` interface, allowing the `SvsNode` to be used as a generic node implementation.
func (n *SvsNode) NodeImplTrait() schema.NodeImpl {
	return n
}

// Constructs and configures an SVS node with data sequence tracking under `/nodeId/seqNo`, notification handling under `/notif`, and event handlers for synchronization, attachment/detachment, and data suppression.
func CreateSvsNode(node *schema.Node) schema.NodeImpl {
	ret := &SvsNode{
		BaseNodeImpl: schema.BaseNodeImpl{
			Node:        node,
			OnAttachEvt: &schema.EventTarget{},
			OnDetachEvt: &schema.EventTarget{},
		},
		OnMissingData:       &schema.EventTarget{},
		BaseMatching:        enc.Matching{},
		SyncInterval:        30 * time.Second,
		SuppressionInterval: 200 * time.Millisecond,
	}

	path, _ := enc.NamePatternFromStr("/<8=nodeId>/<seq=seqNo>")
	leafNode := node.PutNode(path, schema.LeafNodeDesc)
	leafNode.Set(schema.PropCanBePrefix, false)
	leafNode.Set(schema.PropMustBeFresh, false)
	leafNode.Set(schema.PropLifetime, 4*time.Second)
	leafNode.Set(schema.PropFreshness, 60*time.Second)
	leafNode.Set("ValidDuration", 876000*time.Hour)

	path, _ = enc.NamePatternFromStr("/32=notif")
	ret.notifNode = node.PutNode(path, schema.ExpressPointDesc)
	ret.notifNode.Set(schema.PropCanBePrefix, true)
	ret.notifNode.Set(schema.PropMustBeFresh, true)
	ret.notifNode.Set(schema.PropLifetime, 1*time.Second)
	ret.notifNode.AddEventListener(schema.PropOnInterest, utils.IdPtr(ret.onSyncInt))

	ret.BaseMatching = enc.Matching{}
	ret.OnAttachEvt.Add(utils.IdPtr(ret.onAttach))
	ret.OnDetachEvt.Add(utils.IdPtr(ret.onDetach))

	return ret
}

// Returns the index of the entry in the StateVector with the given name, or -1 if no such entry exists.  

**Explanation**:  
This function iterates through the `Entries` of the provided `StateVector`, checking each entry's `Name` against the given `name` using the `Equal` method. If a match is found, the corresponding index is returned; otherwise, `-1` is returned after the loop completes. The implementation prioritizes simplicity over efficiency, as noted by the comment.
func findSvsEntry(v *spec_svs.StateVector, name enc.Name) int {
	// This is less efficient but enough for a demo.
	for i, n := range v.Entries {
		if name.Equal(n.Name) {
			return i
		}
	}
	return -1
}

// Processes an incoming Sync Interest by comparing remote and local state vectors, updating missing data ranges, handling synchronization state transitions, and managing aggregation timers to ensure consistency in the State Vector Sync protocol.
func (n *SvsNode) onSyncInt(event *schema.Event) any {
	remoteSv, err := spec_svs.ParseStateVector(enc.NewWireView(event.Content), true)
	if err != nil {
		log.Error(n, "Unable to parse state vector - DROP", "err", err)
	}

	// If append() is called on localSv slice, a lock is necessary
	n.dataLock.Lock()
	defer n.dataLock.Unlock()

	// Compare state vectors
	// needFetch := false
	needNotif := false
	for _, cur := range remoteSv.Entries {
		li := findSvsEntry(&n.localSv, cur.Name)
		if li == -1 {
			n.localSv.Entries = append(n.localSv.Entries, &spec_svs.StateVectorEntry{
				Name:  cur.Name,
				SeqNo: cur.SeqNo,
			})
			// needFetch = true
			n.missChan <- MissingData{
				Name:     cur.Name,
				StartSeq: 1,
				EndSeq:   cur.SeqNo + 1,
			}
		} else if n.localSv.Entries[li].SeqNo < cur.SeqNo {
			log.Debug(n, "Missing data found", "name", cur.Name, "local", n.localSv.Entries[li].SeqNo, "cur", cur.SeqNo)
			n.missChan <- MissingData{
				Name:     cur.Name,
				StartSeq: n.localSv.Entries[li].SeqNo + 1,
				EndSeq:   cur.SeqNo + 1,
			}
			n.localSv.Entries[li].SeqNo = cur.SeqNo
			// needFetch = true
		} else if n.localSv.Entries[li].SeqNo > cur.SeqNo {
			log.Debug(n, "Outdated remote on", "name", cur.Name, "local", n.localSv.Entries[li].SeqNo, "cur", cur.SeqNo)
			needNotif = true
		}
	}
	for _, cur := range n.localSv.Entries {
		li := findSvsEntry(remoteSv, cur.Name)
		if li == -1 {
			needNotif = true
		}
	}
	// Notify the callback coroutine if applicable
	// if needFetch {
	// 	select {
	// 	case n.sigChan <- struct{}{}:
	// 	default:
	// 	}
	// }
	// Set sync state if applicable
	// if needNotif {
	// 	n.aggregate(remoteSv)
	// 	if n.state == SyncSteady {
	// 		n.transitToSuppress(remoteSv)
	// 	}
	// }
	// TODO: Have trouble understanding this mechanism from the Spec.
	// From StateVectorSync Spec 4.4,
	// "Incoming Sync Interest is outdated: Node moves to Suppression State."
	// implies the state becomes Suppression State when `remote any< local`
	// From StateVectorSync Spec 6, the box below
	// "local_state_vector any< x"
	// implies the state becomes Suppression State when `local any< remote`
	// Contradiction. The wrong one should be the figure.
	// Since suppression is an optimization that does not affect the demo, ignore for now.
	// Report this issue to the team when have time.

	if needNotif || n.state == SyncSuppression {
		// Set the aggregation timer
		if n.state == SyncSteady {
			n.state = SyncSuppression
			n.aggSv = spec_svs.StateVector{Entries: make([]*spec_svs.StateVectorEntry, len(remoteSv.Entries))}
			copy(n.aggSv.Entries, remoteSv.Entries)
			n.cancelSyncTimer()
			n.cancelSyncTimer = n.timer.Schedule(n.getAggIntv(), n.onSyncTimer)
		} else {
			// Should aggregate the incoming sv first, and only shoot after sync timer.
			n.aggregate(remoteSv)
		}
	} else {
		// Reset the sync timer (already in lock)
		n.cancelSyncTimer()
		n.cancelSyncTimer = n.timer.Schedule(n.getSyncIntv(), n.onSyncTimer)
	}

	return true
}

// Returns a channel for receiving notifications when data is missing, allowing direct access to missing data events without using the OnMissingData handler.
func (n *SvsNode) MissingDataChannel() chan MissingData {
	// Note: DO NOT use with OnMissingData
	return n.missChan
}

// Returns the current sequence number associated with this node, typically used to track or order data versions in the network.
func (n *SvsNode) MySequence() uint64 {
	return n.selfSeq
}

// Aggregates a remote state vector into the node's aggregated state vector by updating existing entries with the maximum sequence number or adding new entries if they do not already exist.
func (n *SvsNode) aggregate(remoteSv *spec_svs.StateVector) {
	for _, cur := range remoteSv.Entries {
		li := findSvsEntry(&n.aggSv, cur.Name)
		if li == -1 {
			n.aggSv.Entries = append(n.aggSv.Entries, &spec_svs.StateVectorEntry{
				Name:  cur.Name,
				SeqNo: cur.SeqNo,
			})
		} else {
			n.aggSv.Entries[li].SeqNo = max(n.aggSv.Entries[li].SeqNo, cur.SeqNo)
		}
	}
}

// This function handles the periodic synchronization timer for an SVS node, transitioning from suppression to steady state if necessary, checking local data for updates against the aggregate state vector, and triggering state vector expression while rescheduling the timer for the next interval.
func (n *SvsNode) onSyncTimer() {
	n.dataLock.Lock()
	defer n.dataLock.Unlock()
	// If in suppression state, first test necessity
	notNecessary := false
	if n.state == SyncSuppression {
		n.state = SyncSteady
		notNecessary = true
		for _, cur := range n.localSv.Entries {
			li := findSvsEntry(&n.aggSv, cur.Name)
			if li == -1 || n.aggSv.Entries[li].SeqNo < cur.SeqNo {
				notNecessary = false
				break
			}
		}
	}
	if !notNecessary {
		n.expressStateVec()
	}
	// In case a new one is just scheduled by the onInterest callback. No-op most of the case.
	n.cancelSyncTimer()
	n.cancelSyncTimer = n.timer.Schedule(n.getSyncIntv(), n.onSyncTimer)
}

// Encodes the local state vector and signals the notification node to request state updates via the "NeedChan" channel.
func (n *SvsNode) expressStateVec() {
	n.notifNode.Apply(n.BaseMatching).Call("NeedChan", n.localSv.Encode())
}

// Returns a randomized synchronization interval by adding a deviation between -1/8 and +1/4 of the base SyncInterval's duration, ensuring asynchronous behavior among nodes.
func (n *SvsNode) getSyncIntv() time.Duration {
	dev := rand.Int63n(n.SyncInterval.Nanoseconds()/4) - n.SyncInterval.Nanoseconds()/8
	return n.SyncInterval + time.Duration(dev)*time.Nanosecond
}

// Returns a suppression interval with randomized jitter, adding a deviation between -50% and +50% of the original SuppressionInterval's nanoseconds to avoid synchronized behavior among nodes.
func (n *SvsNode) getAggIntv() time.Duration {
	dev := rand.Int63n(n.SuppressionInterval.Nanoseconds()) - n.SuppressionInterval.Nanoseconds()/2
	return n.SuppressionInterval + time.Duration(dev)*time.Nanosecond
}

// Generates a new signed Data packet with a unique name derived from the node's own prefix and an auto-incremented sequence number, using the provided content and MatchedNode to construct the packet.
func (n *SvsNode) NewData(mNode schema.MatchedNode, content enc.Wire) enc.Wire {
	n.dataLock.Lock()
	defer n.dataLock.Unlock()

	n.selfSeq++
	newDataName := make(enc.Name, len(n.ownPrefix)+1)
	copy(newDataName, n.ownPrefix)
	newDataName[len(n.ownPrefix)] = enc.NewSequenceNumComponent(n.selfSeq)
	mLeafNode := mNode.Refine(newDataName)
	ret := mLeafNode.Call("Provide", content).(enc.Wire)
	if len(ret) > 0 {
		li := findSvsEntry(&n.localSv, n.SelfName)
		if li >= 0 {
			n.localSv.Entries[li].SeqNo = n.selfSeq
		}
		n.state = SyncSteady
		log.Debug(n, "NewData generated", "seq", n.selfSeq)
		n.expressStateVec()
	} else {
		log.Error(n, "Failed to provide", "seq", n.selfSeq)
		n.selfSeq--
	}
	return ret
}

// Initializes an SVS (State Vector Sync) node by configuring internal state, setting up timers and channels for synchronization, and starting the initial sync process to exchange state vectors with other nodes.
func (n *SvsNode) onAttach(event *schema.Event) any {
	if n.ChannelSize == 0 || len(n.SelfName) == 0 ||
		n.BaseMatching == nil || n.SyncInterval <= 0 || n.SuppressionInterval <= 0 {
		panic(errors.New("SvsNode: not configured before Init"))
	}

	n.timer = event.TargetNode.Engine().Timer()
	n.dataLock = sync.Mutex{}
	n.dataLock.Lock()
	defer n.dataLock.Unlock()

	n.ownPrefix = event.TargetNode.Apply(n.BaseMatching).Name
	n.ownPrefix = append(n.ownPrefix, n.SelfName...)

	// OnMissingData callback

	n.localSv = spec_svs.StateVector{Entries: make([]*spec_svs.StateVectorEntry, 0)}
	n.aggSv = spec_svs.StateVector{Entries: make([]*spec_svs.StateVectorEntry, 0)}
	// n.onMiss = schema.NewEvent[*SvsOnMissingEvent]()
	n.state = SyncSteady
	n.missChan = make(chan MissingData, n.ChannelSize)
	// The first sync Interest should be sent out ASAP
	n.cancelSyncTimer = n.timer.Schedule(min(n.getSyncIntv(), 100*time.Millisecond), n.onSyncTimer)

	n.stopChan = make(chan struct{}, 1)
	if len(n.OnMissingData.Val()) > 0 {
		go n.callbackRoutine()
	}

	// initialize localSv
	// TODO: this demo does not consider recovery from off-line. Should be done via ENV and storage policy.
	n.localSv.Entries = append(n.localSv.Entries, &spec_svs.StateVectorEntry{
		Name:  n.SelfName,
		SeqNo: 0,
	})
	n.selfSeq = 0
	return nil
}

// Handles node detachment by canceling synchronization timers, closing communication channels, and signaling termination in a thread-safe manner.
func (n *SvsNode) onDetach(event *schema.Event) any {
	n.dataLock.Lock()
	defer n.dataLock.Unlock()
	n.cancelSyncTimer()
	close(n.missChan)
	n.stopChan <- struct{}{}
	close(n.stopChan)
	return nil
}

// This function serves as a placeholder for handling callback operations associated with the SVS node, but its implementation is currently incomplete and raises a TODO panic.
func (n *SvsNode) callbackRoutine() {
	panic("TODO: TO BE DONE")
}

// Constructs a Data packet name by appending a generic name component and a sequence number component to a matched node's base name.
func (n *SvsNode) GetDataName(mNode schema.MatchedNode, name []byte, seq uint64) enc.Name {
	ret := make(enc.Name, len(mNode.Name)+2)
	copy(ret, mNode.Name)
	ret[len(mNode.Name)] = enc.Component{Typ: enc.TypeGenericNameComponent, Val: name}
	ret[len(mNode.Name)+1] = enc.NewSequenceNumComponent(seq)
	return ret
}

// Casts the SvsNode to the specified target type, returning the node itself if casting to *SvsNode, its base implementation if casting to *schema.BaseNodeImpl, or nil for unsupported types.
func (n *SvsNode) CastTo(ptr any) any {
	switch ptr.(type) {
	case (*SvsNode):
		return n
	case (*schema.BaseNodeImpl):
		return &(n.BaseNodeImpl)
	default:
		return nil
	}
}

var SvsNodeDesc *schema.NodeImplDesc

// Registers the SvsNode implementation with schema-defined properties (like SyncInterval and ContentType), event handlers (OnAttach/Detach/MissingData), and functions (NewData, GetDataName) for managing NDN data synchronization behavior.
func init() {
	SvsNodeDesc = &schema.NodeImplDesc{
		ClassName: "SvsNode",
		Properties: map[schema.PropKey]schema.PropertyDesc{
			"SyncInterval":        schema.TimePropertyDesc("SyncInterval"),
			"SuppressionInterval": schema.TimePropertyDesc("SuppressionInterval"),
			"BaseMatching":        schema.MatchingPropertyDesc("BaseMatching"),
			"ChannelSize":         schema.DefaultPropertyDesc("ChannelSize"),
			"SelfName":            schema.DefaultPropertyDesc("SelfName"),
			"ContentType":         schema.SubNodePropertyDesc("/<8=nodeId>/<seq=seqNo>", "ContentType"),
			"Lifetime":            schema.SubNodePropertyDesc("/<8=nodeId>/<seq=seqNo>", "Lifetime"),
			"Freshness":           schema.SubNodePropertyDesc("/<8=nodeId>/<seq=seqNo>", "Freshness"),
			"ValidDuration":       schema.SubNodePropertyDesc("/<8=nodeId>/<seq=seqNo>", "ValidDuration"),
			"MustBeFresh":         schema.SubNodePropertyDesc("/<8=nodeId>/<seq=seqNo>", "MustBeFresh"),
		},
		Events: map[schema.PropKey]schema.EventGetter{
			schema.PropOnAttach: schema.DefaultEventTarget(schema.PropOnAttach),
			schema.PropOnDetach: schema.DefaultEventTarget(schema.PropOnDetach),
			"OnMissingData":     schema.DefaultEventTarget("OnMissingData"),
		},
		Functions: map[string]schema.NodeFunc{
			"NewData": func(mNode schema.MatchedNode, args ...any) any {
				if len(args) != 1 {
					err := fmt.Errorf("SvsNode.NewData requires 1 arguments but got %d", len(args))
					log.Error(mNode.Node, err.Error())
					return err
				}
				content, ok := args[0].(enc.Wire)
				if !ok && args[0] != nil {
					err := ndn.ErrInvalidValue{Item: "content", Value: args[0]}
					log.Error(mNode.Node, err.Error())
					return err
				}
				return schema.QueryInterface[*SvsNode](mNode.Node).NewData(mNode, content)
			},
			"MissingDataChannel": func(mNode schema.MatchedNode, args ...any) any {
				if len(args) != 0 {
					err := fmt.Errorf("SvsNode.MissingDataChannel requires 0 arguments but got %d", len(args))
					log.Error(mNode.Node, err.Error())
					return err
				}
				return schema.QueryInterface[*SvsNode](mNode.Node).MissingDataChannel()
			},
			"MySequence": func(mNode schema.MatchedNode, args ...any) any {
				if len(args) != 0 {
					err := fmt.Errorf("SvsNode.MySequence requires 0 arguments but got %d", len(args))
					log.Error(mNode.Node, err.Error())
					return err
				}
				return schema.QueryInterface[*SvsNode](mNode.Node).MySequence()
			},
			"GetDataName": func(mNode schema.MatchedNode, args ...any) any {
				if len(args) != 2 {
					err := fmt.Errorf("SvsNode.GetDataName requires 2 arguments but got %d", len(args))
					log.Error(mNode.Node, err.Error())
					return err
				}
				nodeId, ok := args[0].([]byte)
				if !ok && args[0] != nil {
					err := ndn.ErrInvalidValue{Item: "nodeId", Value: args[0]}
					log.Error(mNode.Node, err.Error())
					return err
				}
				seq, ok := args[1].(uint64)
				if !ok && args[1] != nil {
					err := ndn.ErrInvalidValue{Item: "seq", Value: args[1]}
					log.Error(mNode.Node, err.Error())
					return err
				}
				return schema.QueryInterface[*SvsNode](mNode.Node).GetDataName(mNode, nodeId, seq)
			},
		},
		Create: CreateSvsNode,
	}
	schema.RegisterNodeImpl(SvsNodeDesc)
}
