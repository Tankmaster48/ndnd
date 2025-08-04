package table

import enc "github.com/named-data/ndnd/std/encoding"

// Readvertising instances
var readvertisers = make([]RibReadvertise, 0)

type RibReadvertise interface {
	// Advertise a route in the RIB
	Announce(name enc.Name, route *Route)
	// Remove a route from the RIB
	Withdraw(name enc.Name, route *Route)
}

// Registers a new route readvertiser to the list for propagating route updates.
func AddReadvertiser(r RibReadvertise) {
	readvertisers = append(readvertisers, r)
}

// Announces the specified route for the given name to all registered readvertisers.
func readvertiseAnnounce(name enc.Name, route *Route) {
	for _, r := range readvertisers {
		r.Announce(name, route)
	}
}

// Withdraws the specified route advertisement under the given name by notifying all registered readvertisers to remove the route.
func readvertiseWithdraw(name enc.Name, route *Route) {
	for _, r := range readvertisers {
		r.Withdraw(name, route)
	}
}
