///go:build harness

package elxbtc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"os/signal"
	"testing"
	"time"

	ex "main/electrumx"
)

var (
	// https://github.com/spesmilo/electrum/blob/cffbe44c07a59a7d6a3d5183181659a57de8d2c0/electrum/servers_testnet.json
	testServerAddr = "testnet.aranguren.org:51001"
	// testServerAddr = "blockstream.info:993"
	// testServerAddr = "gsw6sn27quwf6u3swgra6o7lrp5qau6kt3ymuyoxgkth6wntzm2bjwyd.onion:51001"
	testTx      = "581d837b8bcca854406dc5259d1fb1e0d314fcd450fb2d4654e78c48120e0135"
	testGenesis = "000000000933ea01ad0ee984209779baaec3ced90fa3f408719526f8d77f4943"
	// https://github.com/spesmilo/electrum/blob/cffbe44c07a59a7d6a3d5183181659a57de8d2c0/electrum/servers.json
	mainServerAddr = "elx.bitske.com:50002"
	// mainServerAddr = "eai.coincited.net:50002"
	// mainServerAddr = "bitcoin.aranguren.org:50002" // IPv6
	// mainServerAddr = "blockstream.info:700"
	mainTx      = "f53a8b83f85dd1ce2a6ef4593e67169b90aaeb402b3cf806b37afc634ef71fbc"
	mainGenesis = "000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f"
)

func TestRunMainnetNode(t *testing.T) {
	RunNode(t, ex.Mainnet, mainServerAddr, mainTx, mainGenesis, true)
}

func TestRunTestnetNode(t *testing.T) {
	RunNode(t, ex.Testnet, testServerAddr, testTx, testGenesis, false)
}

func RunNode(t *testing.T, network ex.Network, addr, tx, genesis string, useTls bool) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	// ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	ex.DebugMode = true

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatal(err)
	}

	rootCAs, _ := x509.SystemCertPool()
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		RootCAs:            rootCAs,
		MinVersion:         tls.VersionTLS12, // works ok
		ServerName:         host,
	}

	if !useTls {
		tlsConfig = nil
	}

	opts := &ex.ConnectOpts{
		TLSConfig:   tlsConfig,
		DebugLogger: ex.StdoutPrinter,
	}

	sc, err := ex.ConnectServer(ctx, addr, opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(sc.Proto())

	fmt.Printf("\n\n ** Connected to %s **\n\n", network)

	_, err = sc.Banner(ctx)
	if err != nil {
		t.Fatal(err)
	}

	feats, err := sc.Features(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if feats.Genesis != genesis {
		t.Fatalf("wrong genesis hash for Bitcoin on %s: %s",
			feats.Genesis, network)
	}
	t.Log("Genesis correct")

	_, err = sc.GetTransaction(ctx, tx)
	if err != nil {
		t.Fatal(err)
	}

	const until = time.Second * 7
	fmt.Printf("leaving connection open for %s\n", until)
	select {
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	case <-time.After(until):
	}
}
