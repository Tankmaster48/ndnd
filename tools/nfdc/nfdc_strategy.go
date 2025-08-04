package nfdc

import (
	"fmt"
	"os"

	enc "github.com/named-data/ndnd/std/encoding"
	mgmt "github.com/named-data/ndnd/std/ndn/mgmt_2022"
	"github.com/spf13/cobra"
)

// Fetches and prints the list of configured strategy choices (prefix-to-strategy mappings) from the NDN network's status dataset.
func (t *Tool) ExecStrategyList(_ *cobra.Command, args []string) {
	t.Start()
	defer t.Stop()

	suffix := enc.Name{
		enc.NewGenericComponent("strategy-choice"),
		enc.NewGenericComponent("list"),
	}

	data, err := t.fetchStatusDataset(suffix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching status dataset: %+v\n", err)
		os.Exit(1)
		return
	}

	status, err := mgmt.ParseStrategyChoiceMsg(enc.NewWireView(data), true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing strategy list: %+v\n", err)
		os.Exit(1)
		return
	}

	for _, entry := range status.StrategyChoices {
		if entry.Strategy != nil {
			fmt.Printf("prefix=%s strategy=%s\n", entry.Name, entry.Strategy.Name)
		}
	}
}
