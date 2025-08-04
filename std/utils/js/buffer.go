//go:build js && wasm

package js

import "syscall/js"

var arrayBuffer = js.Global().Get("ArrayBuffer")
var uint8Array = js.Global().Get("Uint8Array")

// Converts a Go byte slice into a JavaScript Uint8Array by copying its contents for interoperability with JavaScript code.
func SliceToJsArray(slice []byte) js.Value {
	jsSlice := uint8Array.New(len(slice))
	js.CopyBytesToJS(jsSlice, slice)
	return jsSlice
}

// Converts a JavaScript array (ArrayBuffer or Uint8Array) to a Go byte slice by copying its contents.
func JsArrayToSlice(jsArray js.Value) []byte {
	if jsArray.InstanceOf(arrayBuffer) {
		jsArray = uint8Array.New(jsArray)
	}

	slice := make([]byte, jsArray.Get("byteLength").Int())
	js.CopyBytesToGo(slice, jsArray)
	return slice
}
