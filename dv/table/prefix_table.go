package table

import (
	"slices"

	"github.com/named-data/ndnd/dv/config"
	"github.com/named-data/ndnd/dv/tlv"
	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
)

type PrefixTable struct {
	config  *config.Config
	publish func(enc.Wire)
	routers map[uint64]*PrefixTableRouter
	me      *PrefixTableRouter
}

type PrefixTableRouter struct {
	Prefixes map[string]*PrefixEntry
}

type PrefixEntry struct {
	Name enc.Name
	Cost uint64

	// Only known for the local router
	NextHops []PrefixNextHop
}

type PrefixNextHop struct {
	Face uint64
	Cost uint64
}

// Constructs a new PrefixTable initialized with the provided configuration and publish function, setting up an empty routers map and assigning the 'me' field to the router specified by the configuration's RouterName.
func NewPrefixTable(config *config.Config, publish func(enc.Wire)) *PrefixTable {
	pt := &PrefixTable{
		config:  config,
		publish: publish,
		routers: make(map[uint64]*PrefixTableRouter),
		me:      nil,
	}
	pt.me = pt.GetRouter(config.RouterName())
	return pt
}

// Returns the string representation of the PrefixTable as "dv-prefix".  

This function implements the `String()` method for the `PrefixTable` type, returning the fixed string "dv-prefix" to represent the object.
func (pt *PrefixTable) String() string {
	return "dv-prefix"
}

// Returns the router associated with the specified name, creating a new router with an empty `Prefixes` map if it does not exist.
func (pt *PrefixTable) GetRouter(name enc.Name) *PrefixTableRouter {
	hash := name.Hash()
	router := pt.routers[hash]
	if router == nil {
		router = &PrefixTableRouter{
			Prefixes: make(map[string]*PrefixEntry),
		}
		pt.routers[hash] = router
	}
	return router
}

// Resets the prefix table by clearing all stored prefixes and publishing a network-wide reset operation that designates the current router as the exit point for the affected prefixes.
func (pt *PrefixTable) Reset() {
	log.Info(pt, "Reset table")
	clear(pt.me.Prefixes)

	op := tlv.PrefixOpList{
		ExitRouter:    &tlv.Destination{Name: pt.config.RouterName()},
		PrefixOpReset: true,
	}
	pt.publish(op.Encode())
}

// Announces a prefix on a specific face with a given cost, updating the prefix table's entry and publishing the updated entry if the computed cost changes.
func (pt *PrefixTable) Announce(name enc.Name, face uint64, cost uint64) {
	log.Info(pt, "Local announce", "name", name, "face", face, "cost", cost)
	hash := name.TlvStr()

	// Create nexthop to store
	nexthop := PrefixNextHop{
		Face: face,
		Cost: cost,
	}

	// Check if matching entry already exists
	entry := pt.me.Prefixes[hash]
	if entry == nil {
		entry = &PrefixEntry{
			Name: name,
			Cost: config.CostPfxInfinity,
		}
		pt.me.Prefixes[hash] = entry
	}

	// Update entry with nexthop
	found := false
	for i, nh := range entry.NextHops {
		if nh.Face == face {
			found = true
			entry.NextHops[i] = nexthop
			break
		}
	}
	if !found {
		entry.NextHops = append(entry.NextHops, nexthop)
	}

	// Compute cost and publish if dirty
	if entry.computeCost() {
		pt.publishEntry(hash)
	}
}

// Withdraws a prefix from a specific face by removing the face from the prefix's next hops and republishes the prefix entry if its cost changes as a result.
func (pt *PrefixTable) Withdraw(name enc.Name, face uint64) {
	log.Info(pt, "Local withdraw", "name", name, "face", face)
	hash := name.TlvStr()

	// Check if entry does not exist
	entry := pt.me.Prefixes[hash]
	if entry == nil {
		return
	}

	// Remove nexthop from entry
	for i, nh := range entry.NextHops {
		if nh.Face == face {
			entry.NextHops = slices.Delete(entry.NextHops, i, i+1)
			break
		}
	}

	// Compute cost and publish if dirty
	if entry.computeCost() {
		pt.publishEntry(hash)
	}
}

// Publishes the update to the network.
func (pt *PrefixTable) publishEntry(hash string) {
	entry := pt.me.Prefixes[hash]
	if entry == nil {
		return // never
	}

	if entry.Cost < config.CostPfxInfinity {
		log.Info(pt, "Global announce", "name", entry.Name, "cost", entry.Cost)
		op := tlv.PrefixOpList{
			ExitRouter: &tlv.Destination{Name: pt.config.RouterName()},
			PrefixOpAdds: []*tlv.PrefixOpAdd{{
				Name: entry.Name,
				Cost: entry.Cost,
			}},
		}
		pt.publish(op.Encode())
	} else {
		log.Info(pt, "Global withdraw", "name", entry.Name)
		op := tlv.PrefixOpList{
			ExitRouter:      &tlv.Destination{Name: pt.config.RouterName()},
			PrefixOpRemoves: []*tlv.PrefixOpRemove{{Name: entry.Name}},
		}
		pt.publish(op.Encode())
		delete(pt.me.Prefixes, hash) // dead
	}
}

// Applies ops from a list. Returns if dirty.
func (pt *PrefixTable) Apply(wire enc.Wire) (dirty bool) {
	ops, err := tlv.ParsePrefixOpList(enc.NewWireView(wire), true)
	if err != nil {
		log.Warn(pt, "Failed to parse PrefixOpList", "err", err)
		return false
	}

	if ops.ExitRouter == nil || len(ops.ExitRouter.Name) == 0 {
		log.Error(pt, "Received PrefixOpList has no ExitRouter")
		return false
	}

	router := pt.GetRouter(ops.ExitRouter.Name)

	if ops.PrefixOpReset {
		log.Info(pt, "Reset remote prefixes", "router", ops.ExitRouter.Name)
		router.Prefixes = make(map[string]*PrefixEntry)
		dirty = true
	}

	for _, add := range ops.PrefixOpAdds {
		log.Info(pt, "Add remote prefix", "router", ops.ExitRouter.Name, "name", add.Name, "cost", add.Cost)
		router.Prefixes[add.Name.TlvStr()] = &PrefixEntry{
			Name: add.Name.Clone(),
			Cost: add.Cost,
		}
		dirty = true
	}

	for _, remove := range ops.PrefixOpRemoves {
		log.Info(pt, "Remove remote prefix", "router", ops.ExitRouter.Name, "name", remove.Name)
		delete(router.Prefixes, remove.Name.TlvStr())
		dirty = true
	}

	return dirty
}

// Generates an encoded prefix operation list snapshot containing the exit router and all registered prefixes with their associated costs, resetting the target's state upon application.
func (pt *PrefixTable) Snap() enc.Wire {
	snap := tlv.PrefixOpList{
		ExitRouter:    &tlv.Destination{Name: pt.config.RouterName()},
		PrefixOpReset: true,
		PrefixOpAdds:  make([]*tlv.PrefixOpAdd, 0, len(pt.me.Prefixes)),
	}

	for _, entry := range pt.me.Prefixes {
		snap.PrefixOpAdds = append(snap.PrefixOpAdds, &tlv.PrefixOpAdd{
			Name: entry.Name,
			Cost: entry.Cost,
		})
	}

	return snap.Encode()
}

// Computes the minimum cost among all next hops for the prefix entry and returns true if the entry's stored cost is updated as a result.
func (e *PrefixEntry) computeCost() (dirty bool) {
	cost := ^uint64(0)
	for _, nh := range e.NextHops {
		if nh.Cost < cost {
			cost = nh.Cost
		}
	}
	if cost == e.Cost {
		return false
	}
	e.Cost = cost
	return true
}
