//go:build !tinygo

package face

import (
	"fmt"
	"net"

	"github.com/gorilla/websocket"
	"github.com/named-data/ndnd/fw/core"
	defn "github.com/named-data/ndnd/fw/defn"
	spec_mgmt "github.com/named-data/ndnd/std/ndn/mgmt_2022"
)

// WebSocketTransport communicates with web applications via WebSocket.
type WebSocketTransport struct {
	transportBase
	c *websocket.Conn
}

// Constructs a WebSocket-based transport for Named Data Networking (NDN) communication, initializing remote and local URIs, determining network scope (local or non-local), and configuring transport parameters such as persistency, link kind, and maximum packet size.
func NewWebSocketTransport(localURI *defn.URI, c *websocket.Conn) (t *WebSocketTransport) {
	remoteURI := defn.MakeWebSocketClientFaceURI(c.RemoteAddr())

	scope := defn.NonLocal
	ip := net.ParseIP(remoteURI.PathHost())
	if ip != nil && ip.IsLoopback() {
		scope = defn.Local
	}

	t = &WebSocketTransport{c: c}
	t.makeTransportBase(remoteURI, localURI, spec_mgmt.PersistencyOnDemand, scope, defn.PointToPoint, defn.MaxNDNPacketSize)
	t.running.Store(true)

	return t
}

// Returns a string representation of the WebSocket transport including its face ID, remote URI, and local URI.
func (t *WebSocketTransport) String() string {
	return fmt.Sprintf("web-socket-transport (faceid=%d remote=%s local=%s)", t.faceID, t.remoteURI, t.localURI)
}

// Returns true if the persistency is set to PersistencyOnDemand, otherwise false.
func (t *WebSocketTransport) SetPersistency(persistency spec_mgmt.Persistency) bool {
	return persistency == spec_mgmt.PersistencyOnDemand
}

// Returns the number of packets currently in the send queue waiting to be transmitted over the WebSocket connection.
func (t *WebSocketTransport) GetSendQueueSize() uint64 {
	return 0
}

// Sends a binary frame over a WebSocket connection if the transport is active and the frame size is within the MTU limit, handling errors by closing the connection and tracking total output bytes.
func (t *WebSocketTransport) sendFrame(frame []byte) {
	if !t.running.Load() {
		return
	}

	if len(frame) > t.MTU() {
		core.Log.Warn(t, "Attempted to send frame larger than MTU")
		return
	}

	e := t.c.WriteMessage(websocket.BinaryMessage, frame)
	if e != nil {
		core.Log.Warn(t, "Unable to send on socket - Face DOWN")
		t.Close()
		return
	}

	t.nOutBytes += uint64(len(frame))
}

// Handles incoming WebSocket messages by validating their type and size, processes valid binary NDN packets through the link service, and terminates the connection on errors or closure.
func (t *WebSocketTransport) runReceive() {
	defer t.Close()

	for {
		mt, message, err := t.c.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err) {
				// gracefully closed
			} else if websocket.IsUnexpectedCloseError(err) {
				core.Log.Info(t, "WebSocket closed unexpectedly - DROP and Face DOWN", "err", err)
			} else {
				core.Log.Warn(t, "Unable to read from WebSocket - DROP and Face DOWN", "err", err)
			}
			return
		}

		if mt != websocket.BinaryMessage {
			core.Log.Warn(t, "Ignored non-binary message")
			continue
		}

		if len(message) > defn.MaxNDNPacketSize {
			core.Log.Warn(t, "Received too much data without valid TLV block")
			continue
		}

		t.nInBytes += uint64(len(message))
		t.linkService.handleIncomingFrame(message)
	}
}

// Closes the WebSocket transport by stopping its operation and terminating the underlying WebSocket connection.
func (t *WebSocketTransport) Close() {
	t.running.Store(false)
	t.c.Close()
}
