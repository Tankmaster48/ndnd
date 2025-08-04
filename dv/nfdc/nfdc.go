package nfdc

import (
	"fmt"
	"time"

	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
	mgmt "github.com/named-data/ndnd/std/ndn/mgmt_2022"
)

type NfdMgmtCmd struct {
	Module  string
	Cmd     string
	Args    *mgmt.ControlArgs
	Retries int
}

type NfdMgmtThread struct {
	// engine
	engine ndn.Engine
	// channel for management commands
	channel chan NfdMgmtCmd
	// stop the management thread
	stop chan bool
}

// Constructs a new NfdMgmtThread with the provided ndn.Engine, initializing a buffered channel for management commands (capacity 4096) and an unbuffered stop channel for termination signaling.
func NewNfdMgmtThread(engine ndn.Engine) *NfdMgmtThread {
	return &NfdMgmtThread{
		engine:  engine,
		channel: make(chan NfdMgmtCmd, 4096),
		stop:    make(chan bool),
	}
}

// Returns the string representation "dv-nfdc" for the NFD management thread when printed.
func (m *NfdMgmtThread) String() string {
	return "dv-nfdc"
}

// Starts a loop to process NFD management commands from a channel, retrying failed commands up to a specified limit (or indefinitely if retries are negative), and exits gracefully when a stop signal is received.
func (m *NfdMgmtThread) Start() {
	for {
		select {
		case cmd := <-m.channel:
			for i := 0; i < cmd.Retries || cmd.Retries < 0; i++ {
				_, err := m.engine.ExecMgmtCmd(cmd.Module, cmd.Cmd, cmd.Args)
				if err != nil {
					log.Error(m, "Forwarder command failed", "err", err, "attempt", i,
						"module", cmd.Module, "cmd", cmd.Cmd, "args", cmd.Args)
					time.Sleep(100 * time.Millisecond)
				} else {
					time.Sleep(1 * time.Millisecond)
					break
				}
			}
		case <-m.stop:
			return
		}
	}
}

// Signals the NFD management thread to terminate by sending a termination notification through its stop channel.
func (m *NfdMgmtThread) Stop() {
	m.stop <- true
}

// Sends the provided management command to the internal channel, allowing the NFD management thread to process it asynchronously.
func (m *NfdMgmtThread) Exec(mgmt_cmd NfdMgmtCmd) {
	m.channel <- mgmt_cmd
}

// CreatePermFace creates a new permanent face to the given neighbor.
// This is a blocking call.
// returns: face ID of the created link, whether the face was created, error
func (m *NfdMgmtThread) CreateFace(args *mgmt.ControlArgs) (uint64, bool, error) {
	// create a new face or get the existing one
	raw, err := m.engine.ExecMgmtCmd("faces", "create", args)
	// don't check error here, as the face may already exist (409)

	res, ok := raw.(*mgmt.ControlResponse)
	if !ok || res == nil || res.Val == nil || res.Val.Params == nil || !res.Val.Params.FaceId.IsSet() {
		return 0, false, fmt.Errorf("failed to create face: %+v", err)
	}

	faceId := res.Val.Params.FaceId.Unwrap()
	created := res.Val.StatusCode == 200

	return faceId, created, nil
}
