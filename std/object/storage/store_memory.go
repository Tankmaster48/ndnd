package storage

import (
	"fmt"
	"sync"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
)

type MemoryStore struct {
	// root of the store
	root *memoryStoreNode
	// thread safety
	mutex sync.RWMutex

	// active transaction
	tx *memoryStoreNode
	// transaction mutex
	txMutex sync.Mutex
}

type memoryStoreNode struct {
	// name component
	comp enc.Component
	// children
	children map[string]*memoryStoreNode
	// data wire
	wire []byte
}

// Constructs a new in-memory data store with an empty root node for storing NDN data.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		root: &memoryStoreNode{},
	}
}

// Retrieves the data associated with the given name from the memory store, returning the newest matching entry if the direct entry is empty and prefix mode is enabled.
func (s *MemoryStore) Get(name enc.Name, prefix bool) ([]byte, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if node := s.root.find(name); node != nil {
		if node.wire == nil && prefix {
			node = node.findNewest()
		}
		return node.wire, nil
	}
	return nil, nil
}

// Stores the provided wire-encoded data under the specified name in the MemoryStore, using an active transaction context if one exists.  

*Example usage context:*  
This function is typically used to persist data entries in the in-memory storage, ensuring atomic updates when a transaction is in progress.
func (s *MemoryStore) Put(name enc.Name, wire []byte) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	root := s.root
	if s.tx != nil {
		root = s.tx
	}

	root.insert(name, wire)
	return nil
}

// Removes the data entry with the exact specified name from the in-memory store, ensuring thread safety with a mutex lock.
func (s *MemoryStore) Remove(name enc.Name) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.root.remove(name, false)
	return nil
}

// Removes all entries with the specified prefix from the memory store in a thread-safe manner.
func (s *MemoryStore) RemovePrefix(prefix enc.Name) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.root.remove(prefix, true)
	return nil
}

// Removes all entries from the MemoryStore under the given prefix where the component falls lexicographically between first and last (inclusive), using TLV-encoded string comparison.
func (s *MemoryStore) RemoveFlatRange(prefix enc.Name, first enc.Component, last enc.Component) error {
	firstKey, lastKey := first.TlvStr(), last.TlvStr()
	if firstKey > lastKey {
		return fmt.Errorf("firstKey > lastKey")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	pfx := s.root.find(prefix)
	for child := range pfx.children {
		if child >= firstKey && child <= lastKey {
			delete(pfx.children, child)
		}
	}

	return nil
}

// "Initiates a transaction on the memory store by acquiring a lock and resetting the transaction context, returning the store instance for subsequent transactional operations."
func (s *MemoryStore) Begin() (ndn.Store, error) {
	s.txMutex.Lock()
	s.tx = &memoryStoreNode{}
	return s, nil
}

// Commits the current transaction by merging it into the root data structure and resetting the transaction state.
func (s *MemoryStore) Commit() error {
	defer s.txMutex.Unlock()
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.root.merge(s.tx)
	s.tx = nil
	return nil
}

// Rolls back the current transaction, discarding any uncommitted changes and releasing the transaction lock.
func (s *MemoryStore) Rollback() error {
	defer s.txMutex.Unlock()
	s.tx = nil
	return nil
}

// Returns the total memory size (in bytes) of all stored Data packets in the MemoryStore by summing the lengths of their wire representations.
func (s *MemoryStore) MemSize() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	size := 0
	s.root.walk(func(n *memoryStoreNode) { size += len(n.wire) })
	return size
}

// Returns the deepest node matching the given name by recursively traversing the trie-like structure of name components.
func (n *memoryStoreNode) find(name enc.Name) *memoryStoreNode {
	if len(name) == 0 {
		return n
	}

	if n.children == nil {
		return nil
	}

	key := name[0].TlvStr()
	if child := n.children[key]; child != nil {
		return child.find(name[1:])
	} else {
		return nil
	}
}

// Returns the newest node in the subtree rooted at `n` by recursively selecting the lexicographically greatest named child until a leaf node is reached.
func (n *memoryStoreNode) findNewest() *memoryStoreNode {
	if len(n.children) == 0 {
		return n
	}

	var newest string = ""
	for key := range n.children {
		if key > newest {
			newest = key
		}
	}
	if newest == "" {
		return nil
	}

	known := n.children[newest]
	if sub := known.findNewest(); sub != nil {
		return sub
	}
	return known
}

// Inserts the provided wire-encoded data into a memory-based trie structure, creating or traversing nodes hierarchically according to the components of the given name.
func (n *memoryStoreNode) insert(name enc.Name, wire []byte) {
	if len(name) == 0 {
		n.wire = wire
		return
	}

	if n.children == nil {
		n.children = make(map[string]*memoryStoreNode)
	}

	key := name[0].TlvStr()
	if child := n.children[key]; child != nil {
		child.insert(name[1:], wire)
	} else {
		child = &memoryStoreNode{comp: name[0]}
		child.insert(name[1:], wire)
		n.children[key] = child
	}
}

// Removes the specified name (or subtree if prefix is true) from the trie-like memory store node and returns whether the node should be pruned by its parent.
func (n *memoryStoreNode) remove(name enc.Name, prefix bool) bool {
	// return value is if the parent should prune this child
	if len(name) == 0 {
		n.wire = nil
		if prefix {
			n.children = nil // prune subtree
		}
		return n.children == nil
	}

	if n.children == nil {
		return false
	}

	key := name[0].TlvStr()
	if child := n.children[key]; child != nil {
		prune := child.remove(name[1:], prefix)
		if prune {
			delete(n.children, key)
		}
	}

	return n.wire == nil && len(n.children) == 0
}

// Merges the transaction node's wire data and child nodes into the current node, recursively combining shared children and preserving existing structure where possible.
func (n *memoryStoreNode) merge(tx *memoryStoreNode) {
	if tx.wire != nil {
		n.wire = tx.wire
	}

	for key, child := range tx.children {
		if n.children == nil {
			n.children = make(map[string]*memoryStoreNode)
		}

		if nchild := n.children[key]; nchild != nil {
			nchild.merge(child)
		} else {
			n.children[key] = child
		}
	}
}

// Performs a depth-first traversal of the node's subtree, recursively applying the provided function to the node and all its descendants.
func (n *memoryStoreNode) walk(f func(*memoryStoreNode)) {
	f(n)
	for _, child := range n.children {
		child.walk(f)
	}
}
