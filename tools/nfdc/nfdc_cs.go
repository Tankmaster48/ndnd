package nfdc

import (
	"fmt"
	"os"

	enc "github.com/named-data/ndnd/std/encoding"
	mgmt "github.com/named-data/ndnd/std/ndn/mgmt_2022"
	"github.com/named-data/ndnd/std/utils/toolutils"
	"github.com/spf13/cobra"
)

// Fetches and prints content store (CS) status information, including capacity, admission/serve flags, and hit/miss statistics, from an NDN network using a status dataset query.
func (t *Tool) ExecCsInfo(_ *cobra.Command, args []string) {
	t.Start()
	defer t.Stop()

	suffix := enc.Name{
		enc.NewGenericComponent("cs"),
		enc.NewGenericComponent("info"),
	}

	data, err := t.fetchStatusDataset(suffix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching status dataset: %+v\n", err)
		os.Exit(1)
		return
	}

	status, err := mgmt.ParseCsInfoMsg(enc.NewWireView(data), true)
	if err != nil || status.CsInfo == nil {
		fmt.Fprintf(os.Stderr, "Error parsing CS info: %+v\n", err)
		os.Exit(1)
		return
	}

	info := status.CsInfo

	p := toolutils.StatusPrinter{File: os.Stdout, Padding: 10}
	fmt.Println("CS information:")
	p.Print("capacity", info.Capacity)
	p.Print("admit", info.Flags&mgmt.CsEnableAdmit != 0)
	p.Print("serve", info.Flags&mgmt.CsEnableServe != 0)
	p.Print("nEntries", info.NCsEntries)
	p.Print("nHits", info.NHits)
	p.Print("nMisses", info.NMisses)
}
