package nfdc

import (
	"fmt"
	"os"
	"strings"
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	mgmt "github.com/named-data/ndnd/std/ndn/mgmt_2022"
	"github.com/spf13/cobra"
)

// Lists all routes in the Routing Information Base (RIB) of an NDN node, displaying each route's prefix, next hop, origin, cost, flags, and expiration time in a human-readable format.
func (t *Tool) ExecRouteList(_ *cobra.Command, args []string) {
	t.Start()
	defer t.Stop()

	suffix := enc.Name{
		enc.NewGenericComponent("rib"),
		enc.NewGenericComponent("list"),
	}

	data, err := t.fetchStatusDataset(suffix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching status dataset: %+v\n", err)
		return
	}

	status, err := mgmt.ParseRibStatus(enc.NewWireView(data), true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing RIB status: %+v\n", err)
		return
	}

	for _, entry := range status.Entries {
		for _, route := range entry.Routes {
			expiry := "never"
			if rexpiry, ok := route.ExpirationPeriod.Get(); ok {
				expiry = (time.Duration(rexpiry) * time.Millisecond).String()
			}

			flagList := make([]string, 0)
			for flag, name := range mgmt.RouteFlagList {
				if flag.IsSet(route.Flags) {
					flagList = append(flagList, name)
				}
			}
			flags := strings.Join(flagList, ", ")

			origin := mgmt.RouteOrigin(route.Origin)

			fmt.Printf("prefix=%s nexthop=%d origin=%s cost=%d flags={%s} expires=%s\n",
				entry.Name, route.FaceId, origin, route.Cost, flags, expiry)
		}
	}
}

// Fetches and displays the current Forwarding Information Base (FIB) entries from an NDN daemon, listing each entry's name along with next-hop face IDs and their associated costs.
func (t *Tool) ExecFibList(_ *cobra.Command, args []string) {
	t.Start()
	defer t.Stop()

	suffix := enc.Name{
		enc.NewGenericComponent("fib"),
		enc.NewGenericComponent("list"),
	}

	data, err := t.fetchStatusDataset(suffix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching status dataset: %+v\n", err)
		os.Exit(1)
		return
	}

	status, err := mgmt.ParseFibStatus(enc.NewWireView(data), true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing FIB status: %+v\n", err)
		os.Exit(1)
		return
	}

	fmt.Println("FIB:")
	for _, entry := range status.Entries {
		nexthops := make([]string, 0)
		for _, record := range entry.NextHopRecords {
			nexthops = append(nexthops, fmt.Sprintf("faceid=%d (cost=%d)", record.FaceId, record.Cost))
		}
		fmt.Printf("  %s nexthops={%s}\n", entry.Name, strings.Join(nexthops, ", "))
	}
}
