//go:build !tinygo

package face

import (
	"fmt"

	"github.com/gorilla/websocket"
	enc "github.com/named-data/ndnd/std/encoding"
)

type WebSocketFace struct {
	baseFace
	url  string
	conn *websocket.Conn
}

// Constructs a WebSocketFace with the given URL and local boolean indicating whether it is a local face.
func NewWebSocketFace(url string, local bool) *WebSocketFace {
	return &WebSocketFace{
		baseFace: newBaseFace(local),
		url:      url,
	}
}

// Returns a string representation of the WebSocketFace, including its associated URL, for formatting or logging purposes.
func (f *WebSocketFace) String() string {
	return fmt.Sprintf("websocket-face (%s)", f.url)
}

// Opens a WebSocket connection to the specified URL, verifies required callbacks are set, and initializes the face for communication by starting a goroutine to receive packets.
func (f *WebSocketFace) Open() error {
	if f.IsRunning() {
		return fmt.Errorf("face is already running")
	}

	if f.onError == nil || f.onPkt == nil {
		return fmt.Errorf("face callbacks are not set")
	}

	c, _, err := websocket.DefaultDialer.Dial(f.url, nil)
	if err != nil {
		return err
	}

	f.conn = c
	f.setStateUp()
	go f.receive()

	return nil
}

// Closes the WebSocket connection if the face state is successfully transitioned to closed.
func (f *WebSocketFace) Close() error {
	if f.setStateClosed() {
		return f.conn.Close()
	}

	return nil
}

// Sends a wire-encoded packet over the WebSocket connection if the face is running, returning an error if the face is not active.
func (f *WebSocketFace) Send(pkt enc.Wire) error {
	if !f.IsRunning() {
		return fmt.Errorf("face is not running")
	}

	return f.conn.WriteMessage(websocket.BinaryMessage, pkt.Join())
}

// Receives and processes incoming WebSocket binary messages as NDN packets until the face stops running or an error occurs, transitioning the face to a down state upon completion or error.
func (f *WebSocketFace) receive() {
	defer f.setStateDown()

	for f.IsRunning() {
		messageType, pkt, err := f.conn.ReadMessage()
		if err != nil {
			if f.IsRunning() {
				f.onError(err)
			}
			return
		}

		if messageType != websocket.BinaryMessage {
			continue
		}

		f.onPkt(pkt)
	}
}
