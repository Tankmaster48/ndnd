//go:build js && wasm

package storage_test

import (
	"syscall/js"
	"testing"

	"github.com/named-data/ndnd/std/object/storage"
	tu "github.com/named-data/ndnd/std/utils/testutils"
)

// Constructs a JavaScript-backed storage instance using the global `_ndnd_store_js` object and runs tests to verify basic storage operations, range removal, and (commented out) transaction support.
func TestJsStore(t *testing.T) {
	tu.SetT(t)
	store := storage.NewJsStore(js.Global().Get("_ndnd_store_js"))
	testStoreBasic(t, store)
	testStoreRemoveRange(t, store)
	// testStoreTxn(t, store)
}
