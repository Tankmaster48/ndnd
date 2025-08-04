package utils

import (
	"fmt"
	"os"
	"runtime"
)

// Prints the stack traces of all currently running goroutines to standard error for debugging purposes.
func PrintStackTrace() {
	buf := make([]byte, 1<<20)
	stacklen := runtime.Stack(buf, true)
	fmt.Fprintf(os.Stderr, "*** goroutine dump...\n%s\n*** end\n", buf[:stacklen])
}
