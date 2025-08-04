//go:build js && wasm

package face

import (
	"fmt"
	"sync/atomic"
	"syscall/js"
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	jsutil "github.com/named-data/ndnd/std/utils/js"
)

type WasmWsFace struct {
	baseFace
	url    string
	conn   js.Value
	closed atomic.Bool
}

// Constructs a new WasmWsFace instance with the given URL and local setting, initializing the base face, connection as null, and storing the provided URL for later WebSocket communication.
func NewWasmWsFace(url string, local bool) *WasmWsFace {
	return &WasmWsFace{
		baseFace: newBaseFace(local),
		url:      url,
		conn:     js.Null(),
	}
}

// Returns a string representation of the WasmWsFace, including its URL, for identification or debugging purposes.
func (f *WasmWsFace) String() string {
	return fmt.Sprintf("wasm-ws-face (%s)", f.url)
}

// Opens the WebSocket face, ensuring it is not already running and that error and packet callbacks are set, returning an error if callbacks are missing or nil if already open.
func (f *WasmWsFace) Open() error {
	if f.IsRunning() {
		return nil
	}

	if f.onError == nil || f.onPkt == nil {
		return fmt.Errorf("face callbacks are not set")
	}

	f.closed.Store(false)
	f.reopen()

	return nil
}

// Closes the WebSocket connection by transitioning the state to closed, invoking the JavaScript close method, and nullifying the connection reference.
func (f *WasmWsFace) Close() error {
	if f.setStateClosed() {
		f.closed.Store(true)
		f.conn.Call("close")
		f.conn = js.Null()
	}

	return nil
}

// Sends a packet over a WebSocket connection by converting it to a JavaScript array, if the face is running.
func (f *WasmWsFace) Send(pkt enc.Wire) error {
	if !f.IsRunning() {
		return nil
	}

	arr := jsutil.SliceToJsArray(pkt.Join())
	f.conn.Call("send", arr)

	return nil
}

// Reestablishes a WebSocket connection if not already closed or running, setting up event handlers for messages, opens, errors, and closes, with automatic retries every 4 seconds on disconnection.
func (f *WasmWsFace) reopen() {
	if f.closed.Load() || f.IsRunning() {
		return
	}

	// It seems now Go cannot handle exceptions thrown by JS
	conn := js.Global().Get("WebSocket").New(f.url)
	conn.Set("binaryType", "arraybuffer")

	conn.Call("addEventListener", "message", js.FuncOf(f.receive))
	conn.Call("addEventListener", "open", js.FuncOf(func(this js.Value, args []js.Value) any {
		f.conn = conn
		f.setStateUp()
		return nil
	}))
	conn.Call("addEventListener", "error", js.FuncOf(func(this js.Value, args []js.Value) any {
		f.setStateDown()
		f.conn = js.Null()
		if !f.closed.Load() {
			time.AfterFunc(4*time.Second, func() { f.reopen() })
		}
		return nil
	}))
	conn.Call("addEventListener", "close", js.FuncOf(func(this js.Value, args []js.Value) any {
		f.setStateDown()
		f.conn = js.Null()
		if !f.closed.Load() {
			time.AfterFunc(4*time.Second, func() { f.reopen() })
		}
		return nil
	}))
}

// Handles incoming WebSocket messages by extracting data from the event, converting it to a Go slice, and invoking the packet processing callback (onPkt) for further handling.
func (f *WasmWsFace) receive(this js.Value, args []js.Value) any {
	event := args[0]
	data := event.Get("data")
	f.onPkt(jsutil.JsArrayToSlice(data))
	return nil
}
