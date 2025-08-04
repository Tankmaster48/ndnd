package toolutils

import (
	"fmt"
	"os"
	"strings"
)

type StatusPrinter struct {
	File    *os.File
	Padding int
}

// Prints a key-value pair to the specified file with the key right-padded to the configured padding length, followed by an equals sign and the value.
func (s StatusPrinter) Print(key string, value any) {
	fmt.Fprintf(s.File, "%s%s=%v\n", strings.Repeat(" ", s.Padding-len(key)), key, value)
}
