package cmd

import (
	"fmt"

	"github.com/named-data/ndnd/dv/config"
	"github.com/named-data/ndnd/dv/dv"
	"github.com/named-data/ndnd/std/engine"
	"github.com/named-data/ndnd/std/ndn"
)

type DvExecutor struct {
	engine ndn.Engine
	router *dv.Router
}

// Constructs a DvExecutor instance by validating configuration, initializing an NDN engine with a default face, and creating a distance-vector router.
func NewDvExecutor(config *config.Config) (*DvExecutor, error) {
	dve := new(DvExecutor)

	// Validate configuration sanity
	err := config.Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to validate dv config: %w", err)
	}

	// Start NDN engine
	dve.engine = engine.NewBasicEngine(engine.NewDefaultFace())

	// Create the DV router
	dve.router, err = dv.NewRouter(config, dve.engine)
	if err != nil {
		return nil, fmt.Errorf("failed to create dv router: %w", err)
	}

	return dve, nil
}

// Starts the DV engine and router, initializing the engine first and deferring its cleanup, then running the router indefinitely until an error occurs or the program exits.
func (dve *DvExecutor) Start() {
	err := dve.engine.Start()
	if err != nil {
		panic(fmt.Errorf("failed to start dv engine: %w", err))
	}
	defer dve.engine.Stop()

	err = dve.router.Start() // blocks forever
	if err != nil {
		panic(fmt.Errorf("failed to start dv router: %w", err))
	}
}

// Stops the router associated with the DvExecutor by invoking its Stop method.
func (dve *DvExecutor) Stop() {
	dve.router.Stop()
}

// Returns the router instance associated with this DvExecutor.
func (dve *DvExecutor) Router() *dv.Router {
	return dve.router
}
