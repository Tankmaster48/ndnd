//go:build js && wasm

package face

import (
	"fmt"
	"syscall/js"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	jsutil "github.com/named-data/ndnd/std/utils/js"
)

type WasmSimFace struct {
	baseFace
	gosim js.Value
}

// Constructs a new WasmSimFace object with a base simulation face (configured for Wasm) and initializes its JavaScript interop reference (gosim) to null.
func NewWasmSimFace() *WasmSimFace {
	return &WasmSimFace{
		baseFace: newBaseFace(true),
		gosim:    js.Null(),
	}
}

// Returns the string representation of the WasmSimFace type, which is "wasm-sim-face", when converted to a string.
func (f *WasmSimFace) String() string {
	return "wasm-sim-face"
}

// Initializes the WasmSimFace by validating required callbacks, establishing a connection to the JavaScript gondnsim module, setting up packet reception handling, and transitioning the face to an active state.
func (f *WasmSimFace) Open() error {
	if f.onError == nil || f.onPkt == nil {
		return fmt.Errorf("face callbacks are not set")
	}

	if !f.gosim.IsNull() {
		return fmt.Errorf("face is already running")
	}

	// It seems now Go cannot handle exceptions thrown by JS
	f.gosim = js.Global().Get("gondnsim")
	f.gosim.Call("setRecvPktCallback", js.FuncOf(f.receive))

	log.Info(f, "Face opened")
	f.setStateUp()

	return nil
}

// Closes the WasmSimFace by clearing its JavaScript packet callback and releasing the associated simulation object, ensuring no further packet reception occurs.
func (f *WasmSimFace) Close() error {
	if f.setStateClosed() {
		f.gosim.Call("setRecvPktCallback", js.FuncOf(func(this js.Value, args []js.Value) any {
			return nil
		}))
		f.gosim = js.Null()
	}

	return nil
}

// Sends a packet by converting it into a JavaScript Uint8Array and invoking the `sendPkt` method on the associated JS simulation object, returning an error if the face is not running.
func (f *WasmSimFace) Send(pkt enc.Wire) error {
	if !f.IsRunning() {
		return fmt.Errorf("face is not running")
	}

	l := pkt.Length()
	arr := js.Global().Get("Uint8Array").New(int(l))
	js.CopyBytesToJS(arr, pkt.Join())
	f.gosim.Call("sendPkt", arr)

	return nil
}

// Receives a packet from a JavaScript array, converts it to a Go slice, and passes it to the `onPkt` handler for processing.
func (f *WasmSimFace) receive(this js.Value, args []js.Value) any {
	f.onPkt(jsutil.JsArrayToSlice(args[0]))
	return nil
}
