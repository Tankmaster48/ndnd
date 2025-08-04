package tools

import (
	"fmt"
	"math/rand/v2"
	"os"
	"os/signal"
	"syscall"
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/engine"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
	"github.com/named-data/ndnd/std/types/optional"
	"github.com/named-data/ndnd/std/utils"
	"github.com/spf13/cobra"
)

type PingClient struct {
	prefix enc.Name
	name   enc.Name
	app    ndn.Engine

	// command line configuration
	interval int
	timeout  int
	count    int
	seq      uint64 // changes with each ping

	// stat counters
	nRecv    int
	nNack    int
	nTimeout int

	// stat time counters
	totalTime  time.Duration
	totalCount int

	// stat rtt counters
	rttMin time.Duration
	rttMax time.Duration
	rttAvg time.Duration
}

// Constructs a CLI command for sending NDN Interest packets to a specified prefix with configurable interval, timeout, count, and starting sequence number to measure network latency.
func CmdPingClient() *cobra.Command {
	pc := PingClient{}

	cmd := &cobra.Command{
		GroupID: "tools",
		Use:     "ping PREFIX",
		Short:   "Send Interests to a ping server",
		Long: `Ping a name prefix using Interests like /prefix/ping/number
The numbers in the Interests are randomly generated`,
		Args:    cobra.ExactArgs(1),
		Example: `  ndnd ping /my/prefix -c 5`,
		Run:     pc.run,
	}

	cmd.Flags().IntVarP(&pc.interval, "interval", "i", 1000, "ping interval, in milliseconds")
	cmd.Flags().IntVarP(&pc.timeout, "timeout", "t", 4000, "timeout for each ping, in milliseconds")
	cmd.Flags().IntVarP(&pc.count, "count", "c", 0, "number of pings to send")
	cmd.Flags().Uint64Var(&pc.seq, "seq", 0, "start sequence number")
	return cmd
}

// Returns the string representation of the PingClient as "ping", used for identification in logging or debugging contexts.
func (pc *PingClient) String() string {
	return "ping"
}

// Sends an Interest with a sequence-numbered name, tracks round-trip time and packet outcomes (NACK, timeout, success) for network performance monitoring.
func (pc *PingClient) send(seq uint64) {
	name := pc.name.Append(enc.NewSequenceNumComponent(seq))

	cfg := &ndn.InterestConfig{
		Lifetime: optional.Some(time.Duration(pc.timeout) * time.Millisecond),
		Nonce:    utils.ConvertNonce(pc.app.Timer().Nonce()),
	}

	interest, err := pc.app.Spec().MakeInterest(name, cfg, nil, nil)
	if err != nil {
		log.Fatal(pc, "Unable to make Interest", "err", err)
		return
	}

	pc.totalCount++
	t1 := time.Now()

	err = pc.app.Express(interest, func(args ndn.ExpressCallbackArgs) {
		t2 := time.Now()

		switch args.Result {
		case ndn.InterestResultNack:
			fmt.Printf("nack from %s: seq=%d with reason=%d\n", pc.prefix, seq, args.NackReason)
			pc.nNack++
		case ndn.InterestResultTimeout:
			fmt.Printf("timeout from %s: seq=%d\n", pc.prefix, seq)
			pc.nTimeout++
		case ndn.InterestCancelled:
			fmt.Printf("canceled from %s: seq=%d\n", pc.prefix, seq)
			pc.nNack++
		case ndn.InterestResultData:
			fmt.Printf("content from %s: seq=%d, time=%f ms\n",
				pc.prefix, seq,
				float64(t2.Sub(t1).Microseconds())/1000.0)

			if pc.nRecv == 0 { // lateinit
				pc.rttMin = time.Duration(1<<63 - 1)
			}

			pc.nRecv++
			pc.totalTime += t2.Sub(t1)
			pc.rttMin = min(pc.rttMin, t2.Sub(t1))
			pc.rttMax = max(pc.rttMax, t2.Sub(t1))
			pc.rttAvg = pc.totalTime / time.Duration(pc.nRecv)
		}
	})
	if err != nil {
		log.Fatal(pc, "Unable to send Interest", "err", err)
		return
	}
}

// Prints ping statistics including transmitted/received packets, packet loss percentage, and round-trip time (min/avg/max) in milliseconds for a Named Data Networking ping client.
func (pc *PingClient) stats() {
	if pc.totalCount == 0 {
		fmt.Printf("No interests transmitted\n")
		return
	}

	fmt.Printf("\n--- %s ping statistics ---\n", pc.prefix)
	fmt.Printf("%d interests transmitted, %d received, %d%% lost\n",
		pc.totalCount, pc.nRecv, (pc.nNack+pc.nTimeout)*100/pc.totalCount)
	fmt.Printf("rtt min/avg/max = %f/%f/%f ms\n",
		float64(pc.rttMin.Microseconds())/1000.0,
		float64(pc.rttAvg.Microseconds())/1000.0,
		float64(pc.rttMax.Microseconds())/1000.0)
}

// Sends periodic NDN Interest packets with incrementing sequence numbers to a specified prefix, tracks statistics, and terminates on signal or after reaching a specified ping count.
func (pc *PingClient) run(_ *cobra.Command, args []string) {
	prefix, err := enc.NameFromStr(args[0])
	if err != nil {
		log.Fatal(pc, "Invalid prefix", "name", args[0])
		return
	}
	pc.prefix = prefix
	pc.name = prefix.Append(enc.NewGenericComponent("ping"))

	// initialize sequence number
	if pc.seq == 0 {
		pc.seq = rand.Uint64()
	}

	// start the engine
	pc.app = engine.NewBasicEngine(engine.NewDefaultFace())
	err = pc.app.Start()
	if err != nil {
		log.Fatal(pc, "Unable to start engine", "err", err)
		return
	}
	defer pc.app.Stop()

	// quit on signal
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt, syscall.SIGTERM)

	// start main ping routine
	fmt.Printf("PING %s\n", pc.name)
	defer pc.stats()

	// initial ping
	pc.send(pc.seq)

	// send ping periodically
	ticker := time.NewTicker(time.Duration(pc.interval) * time.Millisecond)
	for {
		select {
		case <-ticker.C:
			pc.seq++
			pc.send(pc.seq)
			if pc.count > 0 && pc.totalCount >= pc.count {
				time.Sleep(time.Duration(pc.timeout) * time.Millisecond)
				return
			}
		case <-sigchan:
			return
		}
	}
}
