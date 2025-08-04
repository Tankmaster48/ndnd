//go:build js && wasm

package js

import "syscall/js"

// Releases all JavaScript functions stored in the provided map to prevent resource leaks.
func ReleaseMap(m map[string]any) {
	for _, val := range m {
		if val, ok := val.(js.Func); ok {
			val.Release()
		}
	}
}
