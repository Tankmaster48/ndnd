package main

import (
	"github.com/named-data/ndnd/dv/cmd"
)

// Executes the `CmdDv` command, which initializes and runs a Named-Data Networking (NDN) data verification or dissemination process.
func main() {
	cmd.CmdDv.Execute()
}
