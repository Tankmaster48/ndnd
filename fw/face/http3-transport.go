//go:build !tinygo

package face

import (
	"fmt"
	"net/netip"

	"github.com/named-data/ndnd/fw/core"
	defn "github.com/named-data/ndnd/fw/defn"
	spec_mgmt "github.com/named-data/ndnd/std/ndn/mgmt_2022"
	"github.com/quic-go/webtransport-go"
)

type HTTP3Transport struct {
	transportBase
	c *webtransport.Session
}

// Constructs an HTTP3Transport using the provided WebTransport session, initializing it with remote and local address-port pairs along with default transport parameters for persistency, scope, and capacity.
func NewHTTP3Transport(remote, local netip.AddrPort, c *webtransport.Session) (t *HTTP3Transport) {
	t = &HTTP3Transport{c: c}
	t.makeTransportBase(defn.MakeQuicFaceURI(remote), defn.MakeQuicFaceURI(local), spec_mgmt.PersistencyOnDemand, defn.NonLocal, defn.PointToPoint, 1000)
	t.running.Store(true)
	return
}

// Returns a string representation of the HTTP3Transport containing face ID, remote URI, and local URI for identification.
func (t *HTTP3Transport) String() string {
	return fmt.Sprintf("http3-transport (faceid=%d remote=%s local=%s)", t.faceID, t.remoteURI, t.localURI)
}

// Returns true if the persistency is set to OnDemand.
func (t *HTTP3Transport) SetPersistency(persistency spec_mgmt.Persistency) bool {
	return persistency == spec_mgmt.PersistencyOnDemand
}

// Returns the current number of bytes queued for transmission in the HTTP/3 transport send buffer.
func (t *HTTP3Transport) GetSendQueueSize() uint64 {
	return 0
}

// Sends a frame over the HTTP3Transport if the transport is active and the frame size is within the MTU, handling errors by closing the connection and updating transmission statistics.
func (t *HTTP3Transport) sendFrame(frame []byte) {
	if !t.running.Load() {
		return
	}

	if len(frame) > t.MTU() {
		core.Log.Warn(t, "Attempted to send frame larger than MTU")
		return
	}

	e := t.c.SendDatagram(frame)
	if e != nil {
		core.Log.Warn(t, "Unable to send on socket - Face DOWN", "err", e)
		t.Close()
		return
	}

	t.nOutBytes += uint64(len(frame))
}

// Handles incoming WebTransport datagrams by receiving and validating NDN packets, updating byte counters, and forwarding valid packets to the link service for processing, while terminating on errors or oversized packets.
func (t *HTTP3Transport) runReceive() {
	defer t.Close()

	for {
		message, err := t.c.ReceiveDatagram(t.c.Context())
		if err != nil {
			core.Log.Warn(t, "Unable to read from WebTransport - DROP and Face DOWN", "err", err)
			return
		}

		if len(message) > defn.MaxNDNPacketSize {
			core.Log.Warn(t, "Received too much data without valid TLV block")
			continue
		}

		t.nInBytes += uint64(len(message))
		t.linkService.handleIncomingFrame(message)
	}
}

// Shuts down the HTTP/3 transport by stopping its operation and closing the underlying connection without reporting an error.
func (t *HTTP3Transport) Close() {
	t.running.Store(false)
	t.c.CloseWithError(0, "")
}
