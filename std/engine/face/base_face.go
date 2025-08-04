package face

import (
	"sync"
	"sync/atomic"
)

// baseFace is the base struct for face implementations.
type baseFace struct {
	running atomic.Bool
	local   bool
	onPkt   func(frame []byte)
	onError func(err error)
	sendMut sync.Mutex

	onUp     sync.Map
	onDown   sync.Map
	onUpHndl int
	onDnHndl int
}

// Constructs a baseFace with the specified local flag and initializes empty sync.Map instances for onUp and onDown event handlers.
func newBaseFace(local bool) baseFace {
	return baseFace{
		local:  local,
		onUp:   sync.Map{},
		onDown: sync.Map{},
	}
}

// Returns true if the face is currently running.
func (f *baseFace) IsRunning() bool {
	return f.running.Load()
}

// Returns true if the face is local (e.g., connected to a local NDN daemon).
func (f *baseFace) IsLocal() bool {
	return f.local
}

// Sets the callback function to be invoked when a packet is received on this face, passing the raw packet data as a byte slice.
func (f *baseFace) OnPacket(onPkt func(frame []byte)) {
	f.onPkt = onPkt
}

// Sets the error handler function to be called when an error occurs on this face, passing the error as an argument.
func (f *baseFace) OnError(onError func(err error)) {
	f.onError = onError
}

// Registers a callback to be invoked when the face becomes active and returns a function to cancel the registration.
func (f *baseFace) OnUp(onUp func()) (cancel func()) {
	hndl := f.onUpHndl
	f.onUp.Store(hndl, onUp)
	f.onUpHndl++
	return func() { f.onUp.Delete(hndl) }
}

// Registers a callback to be invoked when the face becomes unreachable and returns a function to cancel the callback registration.
func (f *baseFace) OnDown(onDown func()) (cancel func()) {
	hndl := f.onDnHndl
	f.onDown.Store(hndl, onDown)
	f.onDnHndl++
	return func() { f.onDown.Delete(hndl) }
}

// setStateDown sets the face to down state, and makes the down
// callback if the face was previously up.
func (f *baseFace) setStateDown() {
	if f.running.Swap(false) {
		f.onDown.Range(func(_, cb any) bool {
			cb.(func())()
			return true
		})
	}
}

// setStateUp sets the face to up state, and makes the up
// callback if the face was previously down.
func (f *baseFace) setStateUp() {
	if !f.running.Swap(true) {
		f.onUp.Range(func(_, cb any) bool {
			cb.(func())()
			return true
		})
	}
}

// setStateClosed sets the face to closed state without
// making the onDown callback. Returns if the face was running.
func (f *baseFace) setStateClosed() bool {
	return f.running.Swap(false)
}
