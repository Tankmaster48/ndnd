/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package mgmt

import (
	"time"

	"github.com/named-data/ndnd/fw/core"
	"github.com/named-data/ndnd/fw/dispatch"
	"github.com/named-data/ndnd/fw/fw"
	"github.com/named-data/ndnd/fw/table"
	enc "github.com/named-data/ndnd/std/encoding"
	mgmt "github.com/named-data/ndnd/std/ndn/mgmt_2022"
	"github.com/named-data/ndnd/std/utils"
)

// ForwarderStatusModule is the module that provide forwarder status information.
type ForwarderStatusModule struct {
	manager *Thread
}

// Implements the Stringer interface by returning the string "mgmt-status" as the identifier for the ForwarderStatusModule.
func (f *ForwarderStatusModule) String() string {
	return "mgmt-status"
}

// Sets the manager field of the ForwarderStatusModule to the provided Thread instance.
func (f *ForwarderStatusModule) registerManager(manager *Thread) {
	f.manager = manager
}

// Returns the manager thread associated with the forwarder status module.
func (f *ForwarderStatusModule) getManager() *Thread {
	return f.manager
}

// Handles incoming Interest requests under the `/localhost` prefix for forwarder status management, dispatching to specific handlers based on the verb in the Interest name or returning an error for unknown verbs.
func (f *ForwarderStatusModule) handleIncomingInterest(interest *Interest) {
	// Only allow from /localhost
	if !LOCAL_PREFIX.IsPrefix(interest.Name()) {
		core.Log.Warn(f, "Received forwarder status management Interest from non-local source - DROP")
		return
	}

	// Dispatch by verb
	verb := interest.Name()[len(LOCAL_PREFIX)+1].String()
	switch verb {
	case "general":
		f.general(interest)
	default:
		f.manager.sendCtrlResp(interest, 501, "Unknown verb", nil)
		return
	}
}

// This function generates and sends a general status dataset for the NDN forwarder in response to an Interest by aggregating thread-specific statistics and encoding them into a Data packet with a predefined name hierarchy.
func (f *ForwarderStatusModule) general(interest *Interest) {
	if len(interest.Name()) > len(LOCAL_PREFIX)+2 {
		// Ignore because contains version and/or segment components
		return
	}

	// Generate new dataset
	status := &mgmt.GeneralStatus{
		NfdVersion:       utils.NDNdVersion,
		StartTimestamp:   time.Duration(core.StartTimestamp.UnixNano()),
		CurrentTimestamp: time.Duration(time.Now().UnixNano()),
		NFibEntries:      uint64(table.FibStrategyTable.GetNumFIBEntries()),
	}
	// Don't set NNameTreeEntries because we don't use a NameTree
	for threadID := 0; threadID < fw.CfgNumThreads(); threadID++ {
		thread := dispatch.GetFWThread(threadID)
		counters := thread.Counters()

		status.NPitEntries += uint64(counters.NPitEntries)
		status.NCsEntries += uint64(counters.NCsEntries)
		status.NInInterests += counters.NInInterests
		status.NInData += counters.NInData
		status.NOutInterests += counters.NOutInterests
		status.NOutData += counters.NOutData
		status.NSatisfiedInterests += counters.NSatisfiedInterests
		status.NUnsatisfiedInterests += counters.NUnsatisfiedInterests
	}

	name := LOCAL_PREFIX.
		Append(enc.NewGenericComponent("status")).
		Append(enc.NewGenericComponent("general"))
	f.manager.sendStatusDataset(interest, name, status.Encode())
}
