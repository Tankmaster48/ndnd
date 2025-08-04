// Package basic gives a default implementation of the Engine interface.
// It only connects to local forwarding node via Unix socket.
package basic

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
	mgmt "github.com/named-data/ndnd/std/ndn/mgmt_2022"
	spec "github.com/named-data/ndnd/std/ndn/spec_2022"
	sig "github.com/named-data/ndnd/std/security/signer"
	"github.com/named-data/ndnd/std/types/optional"
	"github.com/named-data/ndnd/std/utils"
)

const DefaultInterestLife = 4 * time.Second
const TimeoutMargin = 10 * time.Millisecond

type fibEntry = ndn.InterestHandler

type pendInt struct {
	callback    ndn.ExpressCallbackFunc
	deadline    time.Time
	canBePrefix bool
	// mustBeFresh is actually not useful, since Freshness is decided by the cache, not us.
	mustBeFresh   bool
	impSha256     []byte
	timeoutCancel func() error
}

type pitEntry = []*pendInt

type Engine struct {
	face  ndn.Face
	timer ndn.Timer

	// fib contains the registered Interest handlers.
	fib *NameTrie[fibEntry]
	// pit contains pending outgoing Interests.
	pit *NameTrie[pitEntry]

	// Since there is only one main coroutine, no need for RW locks.
	fibLock sync.Mutex
	pitLock sync.Mutex

	// mgmtConf is the configuration for the management protocol.
	mgmtConf *mgmt.MgmtConfig
	// cmdChecker is used to validate NFD management packets.
	cmdChecker ndn.SigChecker

	// inQueue is the incoming packet queue.
	// The face will be blocked when the queue is full.
	inQueue chan []byte
	// taskQueue is the task queue for the main goroutine.
	taskQueue chan func()
	// close is the channel to signal the main goroutine to stop.
	close chan struct{}
	// running is the flag to indicate if the engine is running.
	running atomic.Bool

	// (advanced usage) global hook on receiving data packets
	OnDataHook func(data ndn.Data, raw enc.Wire, sigCov enc.Wire) error
}

// Constructs and initializes a new NDN forwarding engine with the provided face and timer, setting up internal data structures (FIB, PIT), concurrency controls, and asynchronous processing channels for handling network traffic and tasks.
func NewEngine(face ndn.Face, timer ndn.Timer) *Engine {
	if face == nil || timer == nil {
		return nil
	}
	mgmtCfg := mgmt.NewConfig(face.IsLocal(), sig.NewSha256Signer(), spec.Spec{})
	return &Engine{
		face:  face,
		timer: timer,

		fib: NewNameTrie[fibEntry](),
		pit: NewNameTrie[pitEntry](),

		fibLock: sync.Mutex{},
		pitLock: sync.Mutex{},

		mgmtConf:   mgmtCfg,
		cmdChecker: func(enc.Name, enc.Wire, ndn.Signature) bool { return true },

		inQueue:   make(chan []byte, 256),
		taskQueue: make(chan func(), 512),
		close:     make(chan struct{}),
		running:   atomic.Bool{},
	}
}

// Returns a string representation of the Engine instance, which is "basic-engine", for identification or logging purposes.
func (e *Engine) String() string {
	return "basic-engine"
}

// Returns the Engine instance as an ndn.Engine interface, allowing the concrete type to satisfy the ndn.Engine interface contract.
func (e *Engine) EngineTrait() ndn.Engine {
	return e
}

// Returns a new ndn.Spec instance with default values.
func (*Engine) Spec() ndn.Spec {
	return spec.Spec{}
}

// Returns the Engine's internal timer for managing time-based operations.
func (e *Engine) Timer() ndn.Timer {
	return e.timer
}

// Returns the engine's network face for communication with the NDN network.
func (e *Engine) Face() ndn.Face {
	return e.face
}

// Attaches an Interest handler to a specified prefix in the Forwarding Information Base (FIB), returning an error if another handler is already registered for that prefix.
func (e *Engine) AttachHandler(prefix enc.Name, handler ndn.InterestHandler) error {
	e.fibLock.Lock()
	defer e.fibLock.Unlock()
	n := e.fib.MatchAlways(prefix)
	if n.Value() != nil {
		return fmt.Errorf("%w: %s", ndn.ErrMultipleHandlers, prefix)
	}
	n.SetValue(handler)
	return nil
}

// Detaches a handler associated with the specified prefix from the Forwarding Information Base (FIB), removing any associated routing entries and cleaning up the node if necessary.  

Example: Detaches a previously registered handler for the given NDN name prefix from the FIB.
func (e *Engine) DetachHandler(prefix enc.Name) error {
	e.fibLock.Lock()
	defer e.fibLock.Unlock()

	n := e.fib.ExactMatch(prefix)
	if n == nil {
		return ndn.ErrInvalidValue{Item: "prefix", Value: prefix}
	}
	n.SetValue(nil)
	n.Prune()
	return nil
}

// Processes incoming NDN packets by parsing L3/L2 formats, handling LpPacket fragmentation, extracting Nack/Interest/Data content, and routing to appropriate handler functions (onInterest, onData, or onNack) with contextual metadata like PIT tokens and signature coverage.
func (e *Engine) onPacket(frame []byte) error {
	reader := enc.NewBufferView(frame)

	var nackReason uint64 = spec.NackReasonNone
	var pitToken []byte = nil
	var incomingFaceId optional.Optional[uint64]
	var raw enc.Wire = nil

	if hasLogTrace() {
		wire := reader.Range(0, reader.Length())
		log.Trace(e, "Received packet bytes", "wire", hex.EncodeToString(wire.Join()))
	}

	// Parse the outer packet - could be either L2 or L3
	pkt, ctx, err := spec.ReadPacket(reader)
	if err != nil {
		// Recoverable error. Should continue.
		log.Error(e, "Failed to parse packet", "err", err)
		return nil
	}

	// Now, exactly one of Interest, Data, LpPacket is not nil
	// First check LpPacket, and do further parse.
	if pkt.LpPacket != nil {
		lpPkt := pkt.LpPacket
		if lpPkt.FragIndex.IsSet() || lpPkt.FragCount.IsSet() {
			log.Warn(e, "Fragmented LpPackets are not supported - DROP")
			return nil
		}

		// Parse the inner packet.
		raw = pkt.LpPacket.Fragment
		if len(raw) == 1 {
			pkt, ctx, err = spec.ReadPacket(enc.NewBufferView(raw[0]))
		} else {
			pkt, ctx, err = spec.ReadPacket(enc.NewWireView(raw))
		}

		// Make sure there is an inner packet.
		if err != nil || (pkt.Data == nil) == (pkt.Interest == nil) {
			if hasLogTrace() {
				wire := reader.Range(0, reader.Length())
				log.Trace(e, "Failed to parse packet bytes", "wire", hex.EncodeToString(wire.Join()))
			}

			// Recoverable error. Should continue.
			log.Error(e, "Failed to parse packet in LpPacket", "err", err)
			return nil
		}

		// Set parameters
		if lpPkt.Nack != nil {
			nackReason = lpPkt.Nack.Reason
		}
		pitToken = lpPkt.PitToken
		incomingFaceId = lpPkt.IncomingFaceId
	} else {
		raw = reader.Range(0, reader.Length())
	}

	// Now pkt is either Data or Interest (including Nack).
	if nackReason != spec.NackReasonNone {
		if pkt.Interest == nil {
			log.Error(e, "Nack received for non-Interest", "reason", nackReason)
			return nil
		}
		log.Trace(e, "Nack received", "reason", nackReason, "name", pkt.Interest.Name())
		e.onNack(pkt.Interest.NameV, nackReason)
	} else if pkt.Interest != nil {
		log.Trace(e, "Interest received", "name", pkt.Interest.Name())
		e.onInterest(ndn.InterestHandlerArgs{
			Interest:       pkt.Interest,
			RawInterest:    raw,
			SigCovered:     ctx.Interest_context.SigCovered(),
			PitToken:       pitToken,
			IncomingFaceId: incomingFaceId,
		})
	} else if pkt.Data != nil {
		log.Trace(e, "Data received", "name", pkt.Data.Name())
		// PitToken is not used for now
		e.onData(pkt.Data, ctx.Data_context.SigCovered(), raw, pitToken)
	} else {
		panic("[BUG] unexpected packet type") // checked above
	}

	return nil
}

// Processes an incoming Interest by determining the appropriate handler via FIB longest-prefix matching, configuring a reply callback with PIT token, and invoking the handler to generate a Data response.
func (e *Engine) onInterest(args ndn.InterestHandlerArgs) {
	name := args.Interest.Name()

	// Compute deadline
	args.Deadline = e.timer.Now().Add(
		args.Interest.Lifetime().GetOr(DefaultInterestLife))

	// Match node
	handler := func() ndn.InterestHandler {
		e.fibLock.Lock()
		defer e.fibLock.Unlock()
		n := e.fib.PrefixMatch(name)

		// If we have the prefix-free condition, we can return the value here
		// directly. But we need longest prefix match now.
		// return n.Value()

		for n != nil && n.Value() == nil {
			n = n.Parent()
		}
		if n != nil {
			return n.Value()
		}
		return nil
	}()
	if handler == nil {
		log.Warn(e, "No handler for interest", "name", name)
		return
	}

	// The reply callback function
	args.Reply = e.newDataReplyFunc(args.PitToken)

	// Call the handler. The handler should create goroutine to avoid blocking.
	// Do not `go` here because if Data is ready at hand, creating a goroutine is slower.
	handler(args)
}

// Constructs a WireReplyFunc that sends a Data packet (with optional LP PIT token wrapping) through the Engine's face, returning an error if the face is not running.
func (e *Engine) newDataReplyFunc(pitToken []byte) ndn.WireReplyFunc {
	return func(dataWire enc.Wire) error {
		if dataWire == nil {
			return nil
		}

		// Check if the face is running
		if !e.IsRunning() || !e.face.IsRunning() {
			return ndn.ErrFaceDown
		}

		// Outgoing packet
		var outWire enc.Wire = dataWire

		// Wrap the data in LP packet if needed
		if pitToken != nil {
			lpPkt := &spec.Packet{
				LpPacket: &spec.LpPacket{
					PitToken: pitToken,
					Fragment: dataWire,
				},
			}
			encoder := spec.PacketEncoder{}
			encoder.Init(lpPkt)
			wire := encoder.Encode(lpPkt)
			if wire == nil {
				log.Error(e, "[BUG] Failed to encode LP packet")
			} else {
				outWire = wire
			}
		}

		return e.face.Send(outWire)
	}
}

// Handles incoming Data packets by finding and removing matching PIT entries based on name prefix, CanBePrefix flag, and implicit digest validation, returning satisfied entries for further processing.
func (e *Engine) onDataMatch(pkt *spec.Data, raw enc.Wire) pitEntry {
	e.pitLock.Lock()
	defer e.pitLock.Unlock()

	n := e.pit.PrefixMatch(pkt.NameV)
	if n == nil {
		log.Warn(e, "Received data for an unknown interest - DROP", "name", pkt.Name())
		return nil
	}

	ret := make(pitEntry, 0, 4)
	for cur := n; cur != nil; cur = cur.Parent() {
		entries := cur.Value()
		for i := 0; i < len(entries); i++ {
			entry := entries[i]

			// we don't check MustBeFresh, as it is the job of the cache/forwarder.
			// check CanBePrefix
			if cur.Depth() < len(pkt.NameV) && !entry.canBePrefix {
				continue
			}

			// check ImplicitDigest256
			if entry.impSha256 != nil {
				h := sha256.New()
				for _, buf := range raw {
					h.Write(buf)
				}
				digest := h.Sum(nil)
				if !bytes.Equal(entry.impSha256, digest) {
					continue
				}
			}

			// pop entry
			entries[i] = entries[len(entries)-1]
			entries = entries[:len(entries)-1]
			i-- // recheck the current index
			ret = append(ret, entry)
		}
		cur.SetValue(entries)
	}

	n.PruneIf(func(lst []*pendInt) bool { return len(lst) == 0 })

	return ret
}

// Handles an incoming Data packet by invoking registered hooks, canceling corresponding PIT entries, and notifying their callbacks with the Data or any hook-generated errors.
func (e *Engine) onData(pkt *spec.Data, sigCovered enc.Wire, raw enc.Wire, pitToken []byte) {
	var hookErr error = nil
	if e.OnDataHook != nil {
		hookErr = e.OnDataHook(pkt, raw, sigCovered)
	}

	for _, entry := range e.onDataMatch(pkt, raw) {
		entry.timeoutCancel()
		if entry.callback == nil {
			panic("[BUG] PIT has empty entry")
		}

		if hookErr != nil {
			entry.callback(ndn.ExpressCallbackArgs{
				Result: ndn.InterestResultError,
				Error:  hookErr,
			})
			continue
		}

		entry.callback(ndn.ExpressCallbackArgs{
			Result:     ndn.InterestResultData,
			Data:       pkt,
			RawData:    raw,
			SigCovered: sigCovered,
			NackReason: spec.NackReasonNone,
		})
	}
}

// Handles a received Nack by removing the corresponding PIT entry and invoking the associated callback with the Nack reason and InterestResultNack status.
func (e *Engine) onNack(name enc.Name, reason uint64) {
	entries := func() []*pendInt {
		e.pitLock.Lock()
		defer e.pitLock.Unlock()

		n := e.pit.ExactMatch(name)
		if n == nil {
			log.Warn(e, "Received Nack for an unknown interest - DROP", "name", name)
			return nil
		}

		ret := n.Value()
		n.SetValue(nil)
		n.Prune()
		return ret
	}()

	for _, entry := range entries {
		entry.timeoutCancel()

		if entry.callback == nil {
			panic("[BUG] PIT has empty entry")
		}

		entry.callback(ndn.ExpressCallbackArgs{
			Result:     ndn.InterestResultNack,
			NackReason: reason,
		})
	}
}

// Starts the engine's processing loop, initializing the network face, handling incoming packets and tasks asynchronously, and returning an error if the face is already running or fails to open.
func (e *Engine) Start() error {
	if e.face.IsRunning() {
		return fmt.Errorf("face is already running")
	}

	e.face.OnPacket(func(frame []byte) {
		// Copy received buffer from face so face can reuse it
		frameCopy := make([]byte, len(frame))
		copy(frameCopy, frame)
		e.inQueue <- frameCopy
	})
	e.face.OnError(func(err error) {
		log.Error(e, "Error on face", "err", err, "face", e.face)
		e.Stop()
	})

	err := e.face.Open()
	if err != nil {
		return err
	}

	e.running.Store(true)
	go func() {
		defer e.face.Close()
		defer e.running.Store(false)

		for {
			select {
			case frame := <-e.inQueue:
				err := e.onPacket(frame)
				if err != nil {
					// This never really happens.
					log.Error(e, "[BUG] Engine::onPacket error", "err", err)
				}
			case <-e.close:
				return
			case task := <-e.taskQueue:
				task()
			}
		}
	}()

	return nil
}

// Stops the engine by sending a close signal to terminate its operation and close the associated face, returning an error if the engine is not running.
func (e *Engine) Stop() error {
	if !e.IsRunning() {
		return fmt.Errorf("engine is not running")
	}

	e.close <- struct{}{} // closes face too
	return nil
}

// Returns whether the engine is currently running.
func (e *Engine) IsRunning() bool {
	return e.running.Load()
}

// Handles timeout of pending interest entries by removing expired entries from the PIT, invoking their callbacks with a timeout result, and pruning the NameTrie node if empty.
func (e *Engine) onExpressTimeout(n *NameTrie[pitEntry]) {
	now := e.timer.Now()

	expired := func() []*pendInt {
		e.pitLock.Lock()
		defer e.pitLock.Unlock()

		ret := make([]*pendInt, 0, 4)
		entries := n.Value()
		for i := 0; i < len(entries); i++ {
			entry := entries[i]
			if entry.deadline.After(now) {
				continue
			}

			// pop entry
			entries[i] = entries[len(entries)-1]
			entries = entries[:len(entries)-1]
			i-- // recheck the current index
			ret = append(ret, entry)
		}

		n.SetValue(entries)
		n.PruneIf(func(lst []*pendInt) bool { return len(lst) == 0 })

		return ret
	}()

	for _, entry := range expired {
		if entry.callback == nil {
			panic("[BUG] PIT has empty entry")
		}

		entry.callback(ndn.ExpressCallbackArgs{
			Result:     ndn.InterestResultTimeout,
			NackReason: spec.NackReasonNone,
		})
	}
}

// Sends an Interest packet with specified parameters, processes implicit digest components, schedules timeout handling in the PIT (Pending Interest Table), and prepares to invoke a callback upon receiving a matching Data packet or timeout.
func (e *Engine) Express(interest *ndn.EncodedInterest, callback ndn.ExpressCallbackFunc) error {
	var impSha256 []byte = nil

	finalName := interest.FinalName
	nodeName := interest.FinalName

	if callback == nil {
		callback = func(ndn.ExpressCallbackArgs) {}
	}

	// Handle implicit digest
	if len(finalName) <= 0 {
		return ndn.ErrInvalidValue{Item: "finalName", Value: finalName}
	}
	lastComp := finalName[len(finalName)-1]
	if lastComp.Typ == enc.TypeImplicitSha256DigestComponent {
		impSha256 = lastComp.Val
		nodeName = finalName[:len(finalName)-1]
	}

	// Handle deadline
	lifetime := interest.Config.Lifetime.GetOr(DefaultInterestLife)
	deadline := e.timer.Now().Add(lifetime)

	// Inject interest into PIT
	func() {
		e.pitLock.Lock()
		defer e.pitLock.Unlock()

		n := e.pit.MatchAlways(nodeName)
		entry := &pendInt{
			callback:    callback,
			deadline:    deadline,
			canBePrefix: interest.Config.CanBePrefix,
			mustBeFresh: interest.Config.MustBeFresh,
			impSha256:   impSha256,
			timeoutCancel: e.timer.Schedule(lifetime+TimeoutMargin, func() {
				e.onExpressTimeout(n)
			}),
		}
		n.SetValue(append(n.Value(), entry))
	}()

	// Wrap the interest in link packet if needed
	wire := interest.Wire
	if interest.Config.NextHopId.IsSet() {
		lpPkt := &spec.Packet{
			LpPacket: &spec.LpPacket{
				Fragment:      wire,
				NextHopFaceId: interest.Config.NextHopId,
			},
		}
		encoder := spec.PacketEncoder{}
		encoder.Init(lpPkt)
		wire = encoder.Encode(lpPkt)
	}

	// Send interest to face
	err := e.face.Send(wire)
	if err != nil {
		log.Error(e, "Failed to send interest", "err", err)
	}

	log.Trace(e, "Interest sent", "name", finalName)
	return err
}

// ExecMgmtCmd executes a signed NDN management command by constructing and expressing a signed Interest, validating the received Data packet's signature, and returning the parsed ControlResponse or error.
func (e *Engine) ExecMgmtCmd(module string, cmd string, args any) (any, error) {
	cmdArgs, ok := args.(*mgmt.ControlArgs)
	if !ok {
		return nil, ndn.ErrInvalidValue{Item: "args", Value: args}
	}

	intCfg := &ndn.InterestConfig{
		Lifetime:    optional.Some(1 * time.Second),
		Nonce:       utils.ConvertNonce(e.timer.Nonce()),
		MustBeFresh: true,

		// Signed interest shenanigans (NFD wants this)
		SigNonce: e.timer.Nonce(),
		SigTime:  optional.Some(time.Duration(e.timer.Now().UnixMilli()) * time.Millisecond),
	}
	interest, err := e.mgmtConf.MakeCmd(module, cmd, cmdArgs, intCfg)
	if err != nil {
		return nil, err
	}

	type mgmtResp struct {
		err error
		val *mgmt.ControlResponse
	}
	respCh := make(chan *mgmtResp)

	err = e.Express(interest, func(args ndn.ExpressCallbackArgs) {
		resp := &mgmtResp{}
		defer func() {
			respCh <- resp
			close(respCh)
		}()

		if args.Result == ndn.InterestResultNack {
			resp.err = fmt.Errorf("nack received: %v", args.NackReason)
		} else if args.Result == ndn.InterestResultTimeout {
			resp.err = ndn.ErrDeadlineExceed
		} else if args.Result == ndn.InterestResultData {
			data := args.Data
			valid := e.cmdChecker(data.Name(), args.SigCovered, data.Signature())
			if !valid {
				resp.err = fmt.Errorf("command signature is not valid")
			} else {
				ret, err := mgmt.ParseControlResponse(enc.NewWireView(data.Content()), true)
				if err != nil {
					resp.err = err
				} else {
					resp.val = ret
					if ret.Val != nil {
						if ret.Val.StatusCode == 200 {
							return
						} else {
							resp.err = fmt.Errorf("command failed due to error %d: %s",
								ret.Val.StatusCode, ret.Val.StatusText)
						}
					} else {
						resp.err = fmt.Errorf("improper response")
					}
				}
			}
		} else {
			resp.err = fmt.Errorf("unknown result: %v", args.Result)
		}
	})
	if err != nil {
		return nil, err
	}

	resp := <-respCh
	return resp.val, resp.err
}

// Sets the signer for generating command signatures and the validation function for verifying signatures on incoming commands in the Engine.
func (e *Engine) SetCmdSec(signer ndn.Signer, validator func(enc.Name, enc.Wire, ndn.Signature) bool) {
	e.mgmtConf.SetSigner(signer)
	e.cmdChecker = validator
}

// Registers a prefix with the Routing Information Base (RIB) by executing a management command, returning an error if the registration fails.
func (e *Engine) RegisterRoute(prefix enc.Name) error {
	_, err := e.ExecMgmtCmd("rib", "register", &mgmt.ControlArgs{Name: prefix})
	if err != nil {
		log.Error(e, "Failed to register prefix", "err", err, "name", prefix)
		return err
	} else {
		log.Debug(e, "Prefix registered", "name", prefix)
	}
	return nil
}

// Unregisters a route with the specified prefix from the Routing Information Base (RIB) using a management command.
func (e *Engine) UnregisterRoute(prefix enc.Name) error {
	_, err := e.ExecMgmtCmd("rib", "unregister", &mgmt.ControlArgs{Name: prefix})
	if err != nil {
		log.Error(e, "Failed to unregister prefix", "err", err, "name", prefix)
		return err
	} else {
		log.Debug(e, "Prefix unregistered", "name", prefix)
	}
	return nil
}

// Schedules a task for execution by adding it to the engine's task queue, or spawns a goroutine to enqueue the task if the queue is full to prevent blocking the caller (typically the main goroutine).
func (e *Engine) Post(task func()) {
	select {
	case e.taskQueue <- task:
	default:
		// Do not block in case this is being called from the
		// main goroutine itself - ideally this never happens.
		go func() { e.taskQueue <- task }()
	}
}

// Returns true if the default logger's level is set to trace, indicating that trace-level logging is enabled.
func hasLogTrace() bool {
	return log.Default().Level() <= log.LevelTrace
}
