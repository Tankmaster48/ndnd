package dvc

import (
	"fmt"
	"os"
	"time"

	spec_dv "github.com/named-data/ndnd/dv/tlv"
	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
	"github.com/named-data/ndnd/std/types/optional"
	"github.com/named-data/ndnd/std/utils"
	"github.com/named-data/ndnd/std/utils/toolutils"
	"github.com/spf13/cobra"
)

// Sends an Interest to the localhost NLSR status endpoint, retrieves the corresponding Data packet, and parses it into a DV Status object.
func (t *Tool) DvStatus() (*spec_dv.Status, error) {
	name := enc.Name{
		enc.LOCALHOST,
		enc.NewGenericComponent("nlsr"),
		enc.NewGenericComponent("status"),
	}
	cfg := &ndn.InterestConfig{
		MustBeFresh: true,
		Lifetime:    optional.Some(time.Second),
		Nonce:       utils.ConvertNonce(t.engine.Timer().Nonce()),
	}

	interest, err := t.engine.Spec().MakeInterest(name, cfg, nil, nil)
	if err != nil {
		panic(err)
	}

	ch := make(chan ndn.ExpressCallbackArgs, 1)
	err = t.engine.Express(interest, func(args ndn.ExpressCallbackArgs) { ch <- args })
	if err != nil {
		panic(err)
	}
	eargs := <-ch

	if eargs.Result != ndn.InterestResultData {
		return nil, fmt.Errorf("interest failed: %s", eargs.Result)
	}

	status, err := spec_dv.ParseStatus(enc.NewWireView(eargs.Data.Content()), false)
	if err != nil {
		return nil, err
	}

	return status, nil
}

// Retrieves and prints the current Distance Vector (DV) routing protocol status, including router version, network names, and routing table statistics.
func (t *Tool) RunDvStatus(_ *cobra.Command, args []string) {
	t.Start()
	defer t.Stop()

	status, err := t.DvStatus()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get router status: %+v\n", err)
		os.Exit(1)
	}

	p := toolutils.StatusPrinter{File: os.Stdout, Padding: 12}
	fmt.Println("General DV status:")
	p.Print("version", status.Version)
	p.Print("routerName", status.RouterName.Name)
	p.Print("networkName", status.NetworkName.Name)
	p.Print("nRibEntries", status.NRibEntries)
	p.Print("nNeighbors", status.NNeighbors)
	p.Print("nFibEntries", status.NFibEntries)
}
