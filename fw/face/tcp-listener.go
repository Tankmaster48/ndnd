/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/named-data/ndnd/fw/core"
	defn "github.com/named-data/ndnd/fw/defn"
	"github.com/named-data/ndnd/fw/face/impl"
	"github.com/named-data/ndnd/std/ndn/mgmt_2022"
)

// TCPListener listens for incoming TCP unicast connections.
type TCPListener struct {
	conn     net.Listener
	localURI *defn.URI
	stopped  chan bool
}

// MakeTCPListener constructs a TCPListener.
func MakeTCPListener(localURI *defn.URI) (*TCPListener, error) {
	localURI.Canonize()
	if !localURI.IsCanonical() || (localURI.Scheme() != "tcp4" && localURI.Scheme() != "tcp6") {
		return nil, defn.ErrNotCanonical
	}

	l := new(TCPListener)
	l.localURI = localURI
	l.stopped = make(chan bool, 1)
	return l, nil
}

// Returns a string representation of the TCPListener in the format "tcp-listener (localURI)", where localURI is the listener's bound endpoint.
func (l *TCPListener) String() string {
	return fmt.Sprintf("tcp-listener (%s)", l.localURI)
}

// This function runs a TCP listener that accepts incoming connections, establishes unicast TCP transports, and initializes NDN link services for each connection to handle Named Data Networking communication.
func (l *TCPListener) Run() {
	defer func() { l.stopped <- true }()

	// Create dialer and set reuse address option
	listenConfig := &net.ListenConfig{Control: impl.SyscallReuseAddr}

	// Create listener
	var remote string
	if l.localURI.Scheme() == "tcp4" {
		remote = fmt.Sprintf("%s:%d", l.localURI.PathHost(), l.localURI.Port())
	} else {
		remote = fmt.Sprintf("[%s]:%d", l.localURI.Path(), l.localURI.Port())
	}

	// Start listening for incoming connections
	var err error
	l.conn, err = listenConfig.Listen(context.Background(), l.localURI.Scheme(), remote)
	if err != nil {
		core.Log.Error(l, "Unable to start TCP listener", "err", err)
		return
	}

	// Run accept loop
	for !core.ShouldQuit {
		remoteConn, err := l.conn.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			core.Log.Warn(l, "Unable to accept connection", "err", err)
			continue
		}

		newTransport, err := AcceptUnicastTCPTransport(remoteConn, l.localURI, mgmt_2022.PersistencyPersistent)
		if err != nil {
			core.Log.Error(l, "Failed to create new unicast TCP transport", "err", err)
			continue
		}

		core.Log.Info(l, "Accepting new TCP face", "uri", newTransport.RemoteURI())
		options := MakeNDNLPLinkServiceOptions()
		options.IsFragmentationEnabled = false // reliable stream
		MakeNDNLPLinkService(newTransport, options).Run(nil)
	}
}

// Closes the underlying TCP connection and waits for the listener to fully terminate by blocking until the `stopped` channel receives a signal.
func (l *TCPListener) Close() {
	if l.conn != nil {
		l.conn.Close()
		<-l.stopped
	}
}
