package nfdc

import (
	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
	"github.com/named-data/ndnd/std/object"
)

// Fetches a status dataset by consuming a Data packet with the given suffix appended to the tool's prefix, returning the packet's content or an error.
func (t *Tool) fetchStatusDataset(suffix enc.Name) (enc.Wire, error) {
	// consume-only client, no need for a store
	client := object.NewClient(t.engine, nil, nil)
	client.Start()
	defer client.Stop()

	ch := make(chan ndn.ConsumeState)
	client.ConsumeExt(ndn.ConsumeExtArgs{
		Name:       t.Prefix().Append(suffix...),
		NoMetadata: true, // NFD has no RDR metadata
		Callback:   func(status ndn.ConsumeState) { ch <- status },
	})

	state := <-ch
	if err := state.Error(); err != nil {
		return nil, err
	}

	return state.Content(), nil
}
