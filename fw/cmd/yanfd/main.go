/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package main

import (
	"github.com/named-data/ndnd/fw/cmd"
)

// Serves as the entry point to execute the YaNFD NDN forwarder command, initiating the network daemon's operation.
func main() {
	cmd.CmdYaNFD.Execute()
}
