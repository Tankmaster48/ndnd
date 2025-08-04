//go:build wasm

/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package impl

import (
	"syscall"
)

// Sets the SO_REUSEADDR socket option on the provided raw connection to allow address reuse, enabling the binding of sockets to addresses that are already in use.
func SyscallReuseAddr(network string, address string, c syscall.RawConn) error {
	return nil
}

// Returns the size of the socket's send queue via a system call.
func SyscallGetSocketSendQueueSize(c syscall.RawConn) uint64 {
	return 0
}
