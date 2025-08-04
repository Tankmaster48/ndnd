package sync

import (
	"iter"

	enc "github.com/named-data/ndnd/std/encoding"
)

// SimplePs is a simple Pub/Sub system.
type SimplePs[V any] struct {
	// subs is the list of subscribers.
	subs map[string]SimplePsSub[V]
}

type SimplePsSub[V any] struct {
	// Prefix is the name prefix to subscribe.
	Prefix enc.Name
	// Callback is the callback function.
	Callback func(V)
}

// Constructs a new SimplePs instance with an empty map for managing subscribers and producers of type V.
func NewSimplePs[V any]() SimplePs[V] {
	return SimplePs[V]{
		subs: make(map[string]SimplePsSub[V]),
	}
}

// Registers a callback function to be invoked when data matching the specified prefix is published, storing the subscription in the internal map and panicking if the callback is nil.
func (ps *SimplePs[V]) Subscribe(prefix enc.Name, callback func(V)) error {
	if callback == nil {
		panic("Callback is required for subscription")
	}

	ps.subs[prefix.TlvStr()] = SimplePsSub[V]{
		Prefix:   prefix,
		Callback: callback,
	}

	return nil
}

// Removes a subscription to the specified prefix by deleting the corresponding entry from the internal subscription map using its TLV-encoded string representation.
func (ps *SimplePs[V]) Unsubscribe(prefix enc.Name) {
	delete(ps.subs, prefix.TlvStr())
}

// Returns an iterator over subscriber callbacks whose registered prefix matches the given name prefix.
func (ps *SimplePs[V]) Subs(prefix enc.Name) iter.Seq[func(V)] {
	return func(yield func(func(V)) bool) {
		for _, sub := range ps.subs {
			if sub.Prefix.IsPrefix(prefix) {
				if !yield(sub.Callback) {
					return
				}
			}
		}
	}
}

// Returns true if there are existing subscriptions for the given prefix.
func (ps *SimplePs[V]) HasSub(prefix enc.Name) bool {
	for range ps.Subs(prefix) {
		return true
	}
	return false
}

// Publishes the given data to all subscribers associated with the specified name by invoking their callback functions.
func (ps *SimplePs[V]) Publish(name enc.Name, data V) {
	for sub := range ps.Subs(name) {
		sub(data)
	}
}
