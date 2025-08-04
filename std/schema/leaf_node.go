package schema

import (
	"fmt"
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
	"github.com/named-data/ndnd/std/types/optional"
	"github.com/named-data/ndnd/std/utils"
)

// LeafNode is a leaf of the NTSchema tree, a point where Data packets can be named.
type LeafNode struct {
	ExpressPoint

	OnGetDataSigner *EventTarget

	ContentType ndn.ContentType
	Freshness   time.Duration
	ValidDur    time.Duration
}

// Returns the receiver as a `NodeImpl` interface, enabling `LeafNode` to satisfy the `NodeImpl` interface by providing its own implementation.
func (n *LeafNode) NodeImplTrait() NodeImpl {
	return n
}

// Provide a Data packet with given name and content.
// Name is constructed from matching if nil. If given, name must agree with matching.
func (n *LeafNode) Provide(
	mNode MatchedNode, content enc.Wire, dataCfg *ndn.DataConfig,
) enc.Wire {
	if mNode.Node != n.Node {
		panic("NTSchema tree compromised.")
	}

	// Construct the Data
	node := n.Node
	engine := n.Node.engine
	spec := engine.Spec()
	if dataCfg == nil {
		dataCfg = &ndn.DataConfig{
			ContentType: optional.Some(n.ContentType),
			Freshness:   optional.Some(n.Freshness),
		}
	}
	validDur := n.ValidDur

	event := &Event{
		TargetNode: node,
		Target:     &mNode,
		DataConfig: dataCfg,
		Content:    content,
	}

	// Get a signer for Data.
	evtRet := n.OnGetDataSigner.DispatchUntil(event, func(a any) bool {
		ret, ok := a.(ndn.Signer)
		return ok && ret != nil
	})
	signer, _ := evtRet.(ndn.Signer)

	data, err := spec.MakeData(mNode.Name, dataCfg, content, signer)
	if err != nil {
		log.Error(n, "Unable to encode Data in Provide()", "err", err)
		return nil
	}

	// Store data in the storage
	event.RawPacket = data.Wire
	event.SelfProduced = utils.IdPtr(true)
	event.ValidDuration = &validDur
	event.Deadline = utils.IdPtr(engine.Timer().Now().Add(validDur))
	n.OnSaveStorage.Dispatch(event)

	// Return encoded data
	return data.Wire
}

// Constructs a LeafNode with blob content type, 1-minute freshness, and a long validity duration using the provided node's ExpressPoint and default data-signing event handling.
func CreateLeafNode(node *Node) NodeImpl {
	return &LeafNode{
		ExpressPoint:    *CreateExpressPoint(node).(*ExpressPoint),
		ContentType:     ndn.ContentTypeBlob,
		Freshness:       1 * time.Minute,
		ValidDur:        876000 * time.Hour,
		OnGetDataSigner: &EventTarget{},
	}
}

var LeafNodeDesc *NodeImplDesc

// Initializes a LeafNode implementation descriptor that extends ExpressPoint with content provisioning capabilities, including type-specific properties, event handlers, and a Provide function for signing and publishing Data packets.
func initLeafNodeDesc() {
	LeafNodeDesc = &NodeImplDesc{
		ClassName:  "LeafNode",
		Properties: make(map[PropKey]PropertyDesc, len(ExpressPointDesc.Properties)+3),
		Events:     make(map[PropKey]EventGetter, len(ExpressPointDesc.Events)+1),
		Functions:  make(map[string]NodeFunc, len(ExpressPointDesc.Functions)+1),
		Create:     CreateLeafNode,
	}
	for k, v := range ExpressPointDesc.Properties {
		LeafNodeDesc.Properties[k] = v
	}
	LeafNodeDesc.Properties[PropContentType] = DefaultPropertyDesc(PropContentType)
	LeafNodeDesc.Properties[PropFreshness] = TimePropertyDesc(PropFreshness)
	LeafNodeDesc.Properties["ValidDuration"] = TimePropertyDesc(PropValidDuration)
	for k, v := range ExpressPointDesc.Events {
		LeafNodeDesc.Events[k] = v
	}
	LeafNodeDesc.Events[PropOnGetDataSigner] = DefaultEventTarget(PropOnGetDataSigner)
	for k, v := range ExpressPointDesc.Functions {
		LeafNodeDesc.Functions[k] = v
	}
	LeafNodeDesc.Functions["Provide"] = func(mNode MatchedNode, args ...any) any {
		if len(args) < 1 || len(args) > 2 {
			err := fmt.Errorf("LeafNode.Provide requires 1~2 arguments but got %d", len(args))
			log.Error(mNode.Node, err.Error())
			return err
		}
		// content enc.Wire, dataCfg *ndn.DataConfig,
		content, ok := args[0].(enc.Wire)
		if !ok && args[0] != nil {
			err := ndn.ErrInvalidValue{Item: "content", Value: args[0]}
			log.Error(mNode.Node, err.Error())
			return err
		}
		var dataCfg *ndn.DataConfig
		if len(args) >= 2 {
			dataCfg, ok = args[1].(*ndn.DataConfig)
			if !ok && args[1] != nil {
				err := ndn.ErrInvalidValue{Item: "dataCfg", Value: args[0]}
				log.Error(mNode.Node, err.Error())
				return err
			}
		}
		return QueryInterface[*LeafNode](mNode.Node).Provide(mNode, content, dataCfg)
	}
	RegisterNodeImpl(LeafNodeDesc)
}

// Casts the LeafNode to the specified type, returning a pointer to the corresponding embedded struct (ExpressPoint, BaseNodeImpl) or nil if the type is not supported.
func (n *LeafNode) CastTo(ptr any) any {
	switch ptr.(type) {
	case (*LeafNode):
		return n
	case (*ExpressPoint):
		return &(n.ExpressPoint)
	case (*BaseNodeImpl):
		return &(n.BaseNodeImpl)
	default:
		return nil
	}
}
