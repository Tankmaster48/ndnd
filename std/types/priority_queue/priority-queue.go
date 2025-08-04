package priority_queue

import (
	"container/heap"

	"golang.org/x/exp/constraints"
)

type Item[V any, P constraints.Ordered] struct {
	object   V
	priority P
	index    int
}

type wrapper[V any, P constraints.Ordered] []*Item[V, P]

// Queue represents a priority queue with MINIMUM priority.
type Queue[V any, P constraints.Ordered] struct {
	pq wrapper[V, P]
}

// Returns the number of elements currently in the priority queue.
func (pq *wrapper[V, P]) Len() int {
	return len(*pq)
}

// Implements the Less method for a priority queue, comparing priorities of elements at indices i and j and returning true if the priority at i is less than at j, ensuring the heap maintains correct ordering.
func (pq *wrapper[V, P]) Less(i, j int) bool {
	return (*pq)[i].priority < (*pq)[j].priority
}

// Swaps the elements at positions i and j in the priority queue and updates their stored indices to maintain consistency with their new positions in the underlying slice.
func (pq *wrapper[V, P]) Swap(i, j int) {
	(*pq)[i], (*pq)[j] = (*pq)[j], (*pq)[i]
	(*pq)[i].index = i
	(*pq)[j].index = j
}

// Adds an item to the priority queue by asserting its type, setting its index to the current length of the queue, and appending it to the underlying slice.
func (pq *wrapper[V, P]) Push(x any) {
	item := x.(*Item[V, P])
	item.index = len(*pq)
	*pq = append(*pq, item)
}

// Removes and returns the last element from the priority queue, ensuring memory safety by nil-ing the element and invalidating its index.
func (pq *wrapper[V, P]) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// Len returns the length of the priroity queue.
func (pq *Queue[V, P]) Len() int {
	return pq.pq.Len()
}

// Push pushes the 'value' onto the priority queue.
func (pq *Queue[V, P]) Push(value V, priority P) *Item[V, P] {
	ret := &Item[V, P]{
		object:   value,
		priority: priority,
	}
	heap.Push(&pq.pq, ret)
	return ret
}

// Peek returns the minimum element of the priority queue without removing it.
func (pq *Queue[V, P]) Peek() V {
	return pq.pq[0].object
}

// Peek returns the minimum element's priority.
func (pq *Queue[V, P]) PeekPriority() P {
	return pq.pq[0].priority
}

// Pop removes and returns the minimum element of the priority queue.
func (pq *Queue[V, P]) Pop() V {
	return heap.Pop(&pq.pq).(*Item[V, P]).object
}

// Update modifies the priority and value of the item
func (pq *Queue[V, P]) Update(item *Item[V, P], value V, priority P) {
	item.object = value
	pq.UpdatePriority(item, priority)
}

// UpdatePriority modifies the priority of the item
func (pq *Queue[V, P]) UpdatePriority(item *Item[V, P], priority P) {
	item.priority = priority
	heap.Fix(&pq.pq, item.index)
}

// Value returns the value of the item
func (item *Item[V, P]) Value() V {
	return item.object
}

// New creates a new priority queue. Not required to call.
func New[V any, P constraints.Ordered]() Queue[V, P] {
	return Queue[V, P]{wrapper[V, P]{}}
}
