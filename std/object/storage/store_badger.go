//go:build !js

package storage

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v4"
	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
)

// Store implementation using badger
type BadgerStore struct {
	db *badger.DB
	tx *badger.Txn
}

// Constructs a new BadgerStore using the specified path for the BadgerDB database.
func NewBadgerStore(path string) (*BadgerStore, error) {
	db, err := badger.Open(badger.DefaultOptions(path))
	if err != nil {
		return nil, err
	}

	return &BadgerStore{db: db}, nil
}

// Closes the underlying Badger database connection and returns any error encountered during closure.
func (s *BadgerStore) Close() error {
	return s.db.Close()
}

// Retrieves the most recent data value associated with the exact name or the longest prefix match from the BadgerDB store, returning the encoded content as a byte slice.
func (s *BadgerStore) Get(name enc.Name, prefix bool) (wire []byte, err error) {
	if s.tx != nil {
		panic("Get() called within a write transaction")
	}

	key := s.nameKey(name)
	err = s.db.View(func(txn *badger.Txn) error {
		// Exact match
		if !prefix {
			item, err := txn.Get(key)
			if errors.Is(err, badger.ErrKeyNotFound) {
				return nil
			}
			wire, err = item.ValueCopy(nil)
			return err
		}

		// Prefix match
		opts := badger.DefaultIteratorOptions
		opts.Reverse = true // newest first
		it := txn.NewIterator(opts)
		defer it.Close()

		it.Seek(append(key, 0xFF))
		if !it.ValidForPrefix(key) {
			return nil
		}

		item := it.Item()
		wire, err = item.ValueCopy(nil)
		return err
	})

	return
}

// Stores the wire-encoded data associated with the given name in the Badger database using a transactional set operation.
func (s *BadgerStore) Put(name enc.Name, wire []byte) error {
	key := s.nameKey(name)
	return s.update(func(txn *badger.Txn) error {
		return txn.Set(key, wire)
	})
}

// Removes the entry identified by the given name from the BadgerStore by deleting the corresponding key-value pair.
func (s *BadgerStore) Remove(name enc.Name) error {
	key := s.nameKey(name)
	return s.update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
}

// Deletes all entries in the BadgerStore that have names starting with the specified prefix.
func (s *BadgerStore) RemovePrefix(prefix enc.Name) error {
	keyPfx := s.nameKey(prefix)

	return s.update(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false // keys only
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(keyPfx); it.ValidForPrefix(keyPfx); it.Next() {
			key := it.Item().KeyCopy(nil)
			if err := txn.Delete(key); err != nil {
				return err
			}
		}

		return nil
	})
}

// Removes all entries in the Badger store whose keys are in the lexicographic range [prefix+first, prefix+last], inclusive.
func (s *BadgerStore) RemoveFlatRange(prefix enc.Name, first enc.Component, last enc.Component) error {
	firstKey := s.nameKey(prefix.Append(first))
	lastKey := s.nameKey(prefix.Append(last))
	if bytes.Compare(firstKey, lastKey) > 0 {
		return fmt.Errorf("firstKey > lastKey")
	}

	return s.update(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false // keys only
		it := txn.NewIterator(opts)
		defer it.Close()

		it.Seek(firstKey)
		for {
			if !it.Valid() {
				return nil
			}

			key := it.Item().KeyCopy(nil)
			if bytes.Compare(key, lastKey) > 0 {
				return nil
			}

			if err := txn.Delete(key); err != nil {
				return err
			}
			it.Next()
		}
	})
}

// Starts a write transaction, returning a new BadgerStore instance bound to the transaction, and panics if called while already within a transaction.
func (s *BadgerStore) Begin() (ndn.Store, error) {
	if s.tx != nil {
		panic("Begin() called within a write transaction")
	}
	tx := s.db.NewTransaction(true)
	return &BadgerStore{db: s.db, tx: tx}, nil
}

// Commits the current write transaction, ensuring all pending changes are persisted to the BadgerStore.
func (s *BadgerStore) Commit() error {
	if s.tx == nil {
		panic("Commit() called without a write transaction")
	}
	return s.tx.Commit()
}

// Aborts the current write transaction, discarding all uncommitted changes.
func (s *BadgerStore) Rollback() error {
	if s.tx == nil {
		panic("Rollback() called without a write transaction")
	}
	s.tx.Discard()
	return nil
}

// Converts the provided `enc.Name` to its internal byte representation for use as a key in BadgerStore.
func (s *BadgerStore) nameKey(name enc.Name) []byte {
	return name.BytesInner()
}

// Executes a write operation within a BadgerDB transaction, using an existing active transaction if present or creating a new one otherwise.
func (s *BadgerStore) update(f func(tx *badger.Txn) error) error {
	if s.tx != nil {
		return f(s.tx)
	} else {
		return s.db.Update(f)
	}
}
