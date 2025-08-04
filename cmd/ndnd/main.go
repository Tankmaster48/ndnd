package main

import (
	"github.com/named-data/ndnd/cmd"
)

// Initializes and runs the NDN daemon command-line interface to start the Named Data Networking service.
func main() {
	cmd.CmdNDNd.Execute()
}
