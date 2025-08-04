package mgmt_2022

import "fmt"

type Persistency uint64

const (
	PersistencyPersistent Persistency = 0
	PersistencyOnDemand   Persistency = 1
	PersistencyPermanent  Persistency = 2
)

var PersistencyList = map[Persistency]string{
	PersistencyPersistent: "persistent",
	PersistencyOnDemand:   "on-demand",
	PersistencyPermanent:  "permanent",
}

// Returns the string representation of the Persistency value, or "unknown" if it is not found in the predefined PersistencyList map.
func (p Persistency) String() string {
	if s, ok := PersistencyList[p]; ok {
		return s
	}
	return "unknown"
}

// Parses a string representation into a Persistency value by matching it against predefined string mappings, returning an error if no match is found.
func ParsePersistency(s string) (Persistency, error) {
	for k, v := range PersistencyList {
		if v == s {
			return k, nil
		}
	}
	return 0, fmt.Errorf("unknown persistency")
}
