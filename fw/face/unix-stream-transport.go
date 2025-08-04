/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"fmt"
	"net"

	"github.com/named-data/ndnd/fw/core"
	defn "github.com/named-data/ndnd/fw/defn"
	"github.com/named-data/ndnd/fw/face/impl"
	spec_mgmt "github.com/named-data/ndnd/std/ndn/mgmt_2022"
	ndn_io "github.com/named-data/ndnd/std/utils/io"
)

// UnixStreamTransport is a Unix stream transport for communicating with local applications.
type UnixStreamTransport struct {
	conn *net.UnixConn
	transportBase
}

// MakeUnixStreamTransport creates a Unix stream transport.
func MakeUnixStreamTransport(remoteURI *defn.URI, localURI *defn.URI, conn net.Conn) (*UnixStreamTransport, error) {
	// Validate URIs
	if !remoteURI.IsCanonical() || remoteURI.Scheme() != "fd" || !localURI.IsCanonical() || localURI.Scheme() != "unix" {
		return nil, defn.ErrNotCanonical
	}

	t := new(UnixStreamTransport)
	t.makeTransportBase(remoteURI, localURI, spec_mgmt.PersistencyPersistent, defn.Local, defn.PointToPoint, defn.MaxNDNPacketSize)

	// Set connection
	t.conn = conn.(*net.UnixConn)
	t.running.Store(true)

	return t, nil
}

// Returns a string representation of the Unix stream transport including its face ID, remote URI, and local URI.
func (t *UnixStreamTransport) String() string {
	return fmt.Sprintf("unix-stream-transport (faceid=%d remote=%s local=%s)", t.faceID, t.remoteURI, t.localURI)
}

// SetPersistency changes the persistency of the face.
func (t *UnixStreamTransport) SetPersistency(persistency spec_mgmt.Persistency) bool {
	if persistency == t.persistency {
		return true
	}

	if persistency == spec_mgmt.PersistencyPersistent {
		t.persistency = persistency
		return true
	}

	return false
}

// GetSendQueueSize returns the current size of the send queue.
func (t *UnixStreamTransport) GetSendQueueSize() uint64 {
	rawConn, err := t.conn.SyscallConn()
	if err != nil {
		core.Log.Warn(t, "Unable to get raw connection to get socket length", "err", err)
	}
	return impl.SyscallGetSocketSendQueueSize(rawConn)
}

// Sends a data frame over the Unix stream transport, enforces MTU size limits, handles transmission errors by closing the connection, and updates the total transmitted byte count on success.
func (t *UnixStreamTransport) sendFrame(frame []byte) {
	if !t.running.Load() {
		return
	}

	if len(frame) > t.MTU() {
		core.Log.Warn(t, "Attempted to send frame larger than MTU")
		return
	}

	_, err := t.conn.Write(frame)
	if err != nil {
		core.Log.Warn(t, "Unable to send on socket - Face DOWN")
		t.Close()
		return
	}

	t.nOutBytes += uint64(len(frame))
}

// Receives and processes incoming NDN TLV-encoded frames from a Unix stream socket connection, updating byte counters and handling each frame via the link service, while logging errors if reading fails.
func (t *UnixStreamTransport) runReceive() {
	defer t.Close()

	err := ndn_io.ReadTlvStream(t.conn, func(b []byte) bool {
		t.nInBytes += uint64(len(b))
		t.linkService.handleIncomingFrame(b)
		return true
	}, nil)
	if err != nil && t.running.Load() {
		core.Log.Warn(t, "Unable to read from socket - Face DOWN", "err", err)
	}
}

// Closes the Unix stream transport, closing the underlying connection only if the transport was previously running, and ensuring it cannot be reused.
func (t *UnixStreamTransport) Close() {
	if t.running.Swap(false) {
		t.conn.Close()
	}
}
