package engine

import (
	"fmt"
	"net/url"
	"os"

	"github.com/named-data/ndnd/std/engine/basic"
	"github.com/named-data/ndnd/std/engine/face"
	"github.com/named-data/ndnd/std/ndn"
)

// Constructs a basic Engine using the provided Face and a new Timer for managing time-based operations.
func NewBasicEngine(face ndn.Face) ndn.Engine {
	return basic.NewEngine(face, basic.NewTimer())
}

// Constructs an NDN face using a Unix domain socket at the specified address for stream-based communication.
func NewUnixFace(addr string) ndn.Face {
	return face.NewStreamFace("unix", addr, true)
}

// Constructs a default Face using the transport URI specified in the client configuration, creating Unix domain socket or TCP-based connections depending on the URI scheme.
func NewDefaultFace() ndn.Face {
	config := GetClientConfig()

	// Parse transport URI
	uri, err := url.Parse(config.TransportUri)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse transport URI %s: %v (invalid client config)\n", uri, err)
		os.Exit(1)
	}

	if uri.Scheme == "unix" {
		return NewUnixFace(uri.Path)
	}

	if uri.Scheme == "tcp" || uri.Scheme == "tcp4" || uri.Scheme == "tcp6" {
		return face.NewStreamFace(uri.Scheme, uri.Host, false)
	}

	fmt.Fprintf(os.Stderr, "Unsupported transport URI: %s (invalid client config)\n", uri)
	os.Exit(1)

	return nil
}
