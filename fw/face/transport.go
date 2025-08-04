/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"sync/atomic"
	"time"

	defn "github.com/named-data/ndnd/fw/defn"
	spec_mgmt "github.com/named-data/ndnd/std/ndn/mgmt_2022"
)

// transport provides an interface for transports for specific face types
type transport interface {
	String() string
	setFaceID(faceID uint64)
	setLinkService(linkService LinkService)

	RemoteURI() *defn.URI
	LocalURI() *defn.URI
	Persistency() spec_mgmt.Persistency
	SetPersistency(persistency spec_mgmt.Persistency) bool
	Scope() defn.Scope
	LinkType() defn.LinkType
	MTU() int
	SetMTU(mtu int)
	ExpirationPeriod() time.Duration
	FaceID() uint64

	// Get the number of queued outgoing packets
	GetSendQueueSize() uint64
	// Send a frame (make if copy if necessary)
	sendFrame([]byte)
	// Receive frames in an infinite loop
	runReceive()
	// Transport is currently running (up)
	IsRunning() bool
	// Close the transport (runReceive should exit)
	Close()

	// Counters
	NInBytes() uint64
	NOutBytes() uint64
}

// transportBase provides logic common types between transport types
type transportBase struct {
	linkService LinkService
	running     atomic.Bool

	faceID         uint64
	remoteURI      *defn.URI
	localURI       *defn.URI
	scope          defn.Scope
	persistency    spec_mgmt.Persistency
	linkType       defn.LinkType
	mtu            int
	expirationTime *time.Time

	// Counters
	nInBytes  uint64
	nOutBytes uint64
}

// Initializes the transportBase instance with specified remote and local URIs, persistency, scope, link type, and MTU values for transport configuration.
func (t *transportBase) makeTransportBase(
	remoteURI *defn.URI,
	localURI *defn.URI,
	persistency spec_mgmt.Persistency,
	scope defn.Scope,
	linkType defn.LinkType,
	mtu int,
) {
	t.running = atomic.Bool{}
	t.remoteURI = remoteURI
	t.localURI = localURI
	t.persistency = persistency
	t.scope = scope
	t.linkType = linkType
	t.mtu = mtu
}

//
// Setters
//

// Sets the face ID of the transport to the specified value.
func (t *transportBase) setFaceID(faceID uint64) {
	t.faceID = faceID
}

// Sets the link service for the transport, enabling it to utilize the provided `LinkService` implementation for network communication.
func (t *transportBase) setLinkService(linkService LinkService) {
	t.linkService = linkService
}

//
// Getters
//

// Returns the local URI associated with the transport instance.
func (t *transportBase) LocalURI() *defn.URI {
	return t.localURI
}

// Returns the remote URI associated with the transport connection.
func (t *transportBase) RemoteURI() *defn.URI {
	return t.remoteURI
}

// Returns the persistency setting of the transport.
func (t *transportBase) Persistency() spec_mgmt.Persistency {
	return t.persistency
}

// Returns the current scope of the transport base.
func (t *transportBase) Scope() defn.Scope {
	return t.scope
}

// Returns the link type of the transport as a `defn.LinkType` value.
func (t *transportBase) LinkType() defn.LinkType {
	return t.linkType
}

// Returns the maximum transmission unit (MTU) size for the transport.
func (t *transportBase) MTU() int {
	return t.mtu
}

// Sets the Maximum Transmission Unit (MTU) for the transport, specifying the maximum size of data packets that can be transmitted.
func (t *transportBase) SetMTU(mtu int) {
	t.mtu = mtu
}

// ExpirationPeriod returns the time until this face expires.
// If transport not on-demand, returns 0.
func (t *transportBase) ExpirationPeriod() time.Duration {
	if t.expirationTime == nil || t.persistency != spec_mgmt.PersistencyOnDemand {
		return 0
	}
	return time.Until(*t.expirationTime)
}

// Returns the unique identifier of the face associated with this transport.
func (t *transportBase) FaceID() uint64 {
	return t.faceID
}

// Returns whether the transport is currently running.
func (t *transportBase) IsRunning() bool {
	return t.running.Load()
}

//
// Counters
//

// Returns the total number of bytes received by the transport.
func (t *transportBase) NInBytes() uint64 {
	return t.nInBytes
}

// Returns the total number of bytes transmitted by this transport as a 64-bit unsigned integer.
func (t *transportBase) NOutBytes() uint64 {
	return t.nOutBytes
}
