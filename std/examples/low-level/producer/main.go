package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/engine"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
	sec_pib "github.com/named-data/ndnd/std/security/pib"
	"github.com/named-data/ndnd/std/types/optional"
)

var app ndn.Engine
var pib *sec_pib.SqlitePib

// Constructs a signed Data packet with "Hello, world!" content, 10s freshness, and Blob content type using the /test identity's certificate, then replies to the Interest and logs the exchange details.
func onInterest(args ndn.InterestHandlerArgs) {
	interest := args.Interest

	fmt.Printf(">> I: %s\n", interest.Name().String())
	content := []byte("Hello, world!")

	idName, _ := enc.NameFromStr("/test")
	identity := pib.GetIdentity(idName)
	cert := identity.FindCert(func(_ sec_pib.Cert) bool { return true })
	signer := cert.AsSigner()

	data, err := app.Spec().MakeData(
		interest.Name(),
		&ndn.DataConfig{
			ContentType: optional.Some(ndn.ContentTypeBlob),
			Freshness:   optional.Some(10 * time.Second),
		},
		enc.Wire{content},
		signer)
	if err != nil {
		log.Error(nil, "Unable to encode data", "err", err)
		return
	}
	err = args.Reply(data.Wire)
	if err != nil {
		log.Error(nil, "Unable to reply with data", "err", err)
		return
	}
	fmt.Printf("<< D: %s\n", interest.Name().String())
	fmt.Printf("Content: (size: %d)\n", len(content))
	fmt.Printf("\n")
}

// Initializes and starts an NDN application with security modules, registers an interest handler for the "/example/testApp" prefix, and runs until interrupted.
func main() {
	app = engine.NewBasicEngine(engine.NewDefaultFace())
	err := app.Start()
	if err != nil {
		log.Fatal(nil, "Unable to start engine", "err", err)
		return
	}
	defer app.Stop()

	homedir, _ := os.UserHomeDir()
	tpm := sec_pib.NewFileTpm(filepath.Join(homedir, ".ndn/ndnsec-key-file"))
	pib = sec_pib.NewSqlitePib(filepath.Join(homedir, ".ndn/pib.db"), tpm)

	prefix, _ := enc.NameFromStr("/example/testApp")
	err = app.AttachHandler(prefix, onInterest)
	if err != nil {
		log.Error(nil, "Unable to register handler", "err", err)
		return
	}
	err = app.RegisterRoute(prefix)
	if err != nil {
		log.Error(nil, "Unable to register route", "err", err)
		return
	}

	fmt.Print("Start serving ...")
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, os.Interrupt, syscall.SIGTERM)
	receivedSig := <-sigChannel
	log.Info(nil, "Received signal - exiting", "signal", receivedSig)
}
