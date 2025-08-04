package sync

import enc "github.com/named-data/ndnd/std/encoding"

// SnapshotNull is a non-snapshot strategy.
type SnapshotNull struct {
}

// Returns the current snapshot instance without modification, indicating this object is already a snapshot.
func (s *SnapshotNull) Snapshot() Snapshot {
	return s
}

// Initializes a SnapshotNull instance with the provided snapshot state and state vector map for data states.
func (s *SnapshotNull) initialize(snapPsState, SvMap[svsDataState]) {
}

// This function serves as a no-op placeholder method for handling update events in the SnapshotNull implementation, accepting state and name parameters without performing any action.
func (s *SnapshotNull) onUpdate(SvMap[svsDataState], enc.Name) {
}

// Handles publication events by accepting a state map and name, but performs no action in the default implementation.
func (s *SnapshotNull) onPublication(SvMap[svsDataState], enc.Name) {
}
