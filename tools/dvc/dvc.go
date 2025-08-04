package dvc

import (
	"fmt"
	"os"

	"github.com/named-data/ndnd/std/engine"
	"github.com/named-data/ndnd/std/ndn"
	"github.com/spf13/cobra"
)

// Constructs and returns a slice of Cobra commands for managing a router's status and neighbor links, including creating, destroying, and querying link status.
func Cmds() []*cobra.Command {
	t := Tool{}

	return []*cobra.Command{{
		Use:   "status",
		Short: "Print general status of the router",
		Args:  cobra.NoArgs,
		Run:   t.RunDvStatus,
	}, {
		Use:   "link-create NEIGHBOR-URI",
		Short: "Create a new active neighbor link",
		Args:  cobra.ExactArgs(1),
		Run:   t.RunDvLinkCreate,
	}, {
		Use:   "link-destroy NEIGHBOR-URI",
		Short: "Destroy an active neighbor link",
		Args:  cobra.ExactArgs(1),
		Run:   t.RunDvLinkDestroy,
	}}
}

type Tool struct {
	engine ndn.Engine
}

// Initializes and starts the NDN engine with a default face, terminating the tool if the engine fails to start.
func (t *Tool) Start() {
	t.engine = engine.NewBasicEngine(engine.NewDefaultFace())

	err := t.engine.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to start engine: %+v\n", err)
		os.Exit(1)
		return
	}
}

// Stops the tool's engine.
func (t *Tool) Stop() {
	t.engine.Stop()
}
