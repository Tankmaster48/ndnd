/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package fw

import (
	"time"

	"github.com/named-data/ndnd/fw/core"
	"github.com/named-data/ndnd/fw/defn"
	"github.com/named-data/ndnd/fw/table"
)

// MulticastSuppressionTime is the time to suppress retransmissions of the same Interest.
const MulticastSuppressionTime = 500 * time.Millisecond

// Multicast is a forwarding strategy that forwards Interests to all nexthop faces.
type Multicast struct {
	StrategyBase
}

// Registers the Multicast strategy with version 1, adding its constructor to the initialization list and mapping it to the "multicast" name in the strategy version registry.
func init() {
	strategyInit = append(strategyInit, func() Strategy { return &Multicast{} })
	StrategyVersions["multicast"] = []uint64{1}
}

// Initializes the base multicast forwarding strategy with the specified thread, naming it "multicast" and using version 1.
func (s *Multicast) Instantiate(fwThread *Thread) {
	s.NewStrategyBase(fwThread, "multicast", 1)
}

// Handles a Content Store hit by logging the event and sending the cached Data packet to the faces specified in the PIT entry, indicating the Content Store as the source (0).
func (s *Multicast) AfterContentStoreHit(
	packet *defn.Pkt,
	pitEntry table.PitEntry,
	inFace uint64,
) {
	core.Log.Trace(s, "AfterContentStoreHit", "name", packet.Name, "faceid", inFace)
	s.SendData(packet, pitEntry, inFace, 0) // 0 indicates ContentStore is source
}

// Forwards the received Data packet to all faces listed in the PIT entry's incoming records to satisfy pending Interests in a multicast scenario.
func (s *Multicast) AfterReceiveData(
	packet *defn.Pkt,
	pitEntry table.PitEntry,
	inFace uint64,
) {
	core.Log.Trace(s, "AfterReceiveData", "name", packet.Name, "inrecords", len(pitEntry.InRecords()))
	for faceID := range pitEntry.InRecords() {
		core.Log.Trace(s, "Forwarding Data", "name", packet.Name, "faceid", faceID)
		s.SendData(packet, pitEntry, faceID, inFace)
	}
}

// Suppresses retransmitted Interests with differing nonces within the suppression interval and forwards new Interests to all nexthops in a multicast scenario.
func (s *Multicast) AfterReceiveInterest(
	packet *defn.Pkt,
	pitEntry table.PitEntry,
	inFace uint64,
	nexthops []*table.FibNextHopEntry,
) {
	if len(nexthops) == 0 {
		core.Log.Debug(s, "No nexthop for Interest", "name", packet.Name)
		return
	}

	// If there is an out record less than suppression interval ago, drop the
	// retransmission to suppress it (only if the nonce is different)
	now := time.Now()
	for _, outRecord := range pitEntry.OutRecords() {
		if outRecord.LatestNonce != packet.L3.Interest.NonceV.Unwrap() &&
			outRecord.LatestTimestamp.Add(MulticastSuppressionTime).After(now) {
			core.Log.Debug(s, "Suppressed Interest", "name", packet.Name)
			return
		}
	}

	// Send interest to all nexthops
	for _, nexthop := range nexthops {
		core.Log.Trace(s, "Forwarding Interest", "name", packet.Name, "faceid", nexthop.Nexthop)
		s.SendInterest(packet, pitEntry, nexthop.Nexthop, inFace)
	}
}

// This function is a no-op in the Multicast strategy, serving as a placeholder for pre-satisfaction logic that is unnecessary for multicast interest handling.
func (s *Multicast) BeforeSatisfyInterest(pitEntry table.PitEntry, inFace uint64) {
	// This does nothing in Multicast
}
