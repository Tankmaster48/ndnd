package lockfree

import (
	"sync/atomic"
)

// Queue is a lock-free queue with a single consumer and multiple producers.
type Queue[T any] struct {
	head *node[T]
	tail atomic.Pointer[node[T]]
}

type node[T any] struct {
	val  T
	next atomic.Pointer[node[T]]
}

// Constructs a new empty queue with a head node of type T and initializes the tail to point to the head, enabling thread-safe operations.
func NewQueue[T any]() *Queue[T] {
	q := &Queue[T]{
		head: &node[T]{},
		tail: atomic.Pointer[node[T]]{},
	}
	q.tail.Store(q.head)
	return q
}

// Adds the given value to the end of the queue using a thread-safe, lock-free algorithm with atomic compare-and-swap operations.
func (q *Queue[T]) Push(v T) {
	n := &node[T]{val: v}
	for {
		tail := q.tail.Load()
		if q.tail.CompareAndSwap(tail, n) {
			tail.next.Store(n)
			return
		}
	}
}

// Returns the front element of the queue by updating the head to the next node, returning the value and true if the queue was non-empty, or the zero value and false otherwise.
func (q *Queue[T]) Pop() (val T, ok bool) {
	next := q.head.next.Load()
	if next == nil {
		return val, false
	}
	q.head = next
	return next.val, true
}
