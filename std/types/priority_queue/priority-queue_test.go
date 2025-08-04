package priority_queue_test

import (
	"testing"

	"github.com/named-data/ndnd/std/types/priority_queue"
	"github.com/stretchr/testify/assert"
)

// This function tests the basic operations of a priority queue by adding elements with varying priorities, verifying the queue length, and ensuring elements are popped in ascending priority order (lowest numerical priority first). 

Example: Validates that a priority queue correctly adds, peeks, and removes elements based on their assigned priorities.
func TestBasics(t *testing.T) {
	q := priority_queue.New[int, int]()
	assert.Equal(t, 0, q.Len())
	q.Push(1, 1)
	q.Push(2, 3)
	q.Push(3, 2)
	assert.Equal(t, 3, q.Len())
	assert.Equal(t, 1, q.PeekPriority())
	assert.Equal(t, 1, q.Pop())
	assert.Equal(t, 2, q.PeekPriority())
	assert.Equal(t, 3, q.Pop())
	assert.Equal(t, 2, q.Pop())
	assert.Equal(t, 0, q.Len())
}
