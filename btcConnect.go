package main

// Run goele as an app

import (
	"errors"
	"flag"
	"fmt"
	"main/client"
	"main/client/btc"
	"main/electrumx"
	"main/wallet"
	"os"
	"path/filepath"

	"github.com/btcsuite/btcd/chaincfg"
)

var (
	coins = []string{"btc"} // add as implemented
	nets  = []string{"mainnet", "testnet", "regtest", "simnet"}
)

func makeBasicConfig(coin, net string) (*client.ClientConfig, error) {
	contains := func(s []string, str string) bool {
		for _, v := range s {
			if v == str {
				return true
			}
		}
		return false
	}
	if !contains(coins, coin) {
		return nil, errors.New("invalid coin")
	}
	if !contains(nets, net) {
		return nil, errors.New("invalid net")
	}
	switch coin {
	case "btc":
	default:
		return nil, errors.New("invalid coin")
	}
	cfg := client.NewDefaultConfig()
	cfg.Chain = wallet.Bitcoin
	cfg.StoreEncSeed = true
	appDir, err := client.GetConfigPath()
	if err != nil {
		return nil, err
	}
	coinNetDir := filepath.Join(appDir, coin, net)
	err = os.MkdirAll(coinNetDir, os.ModeDir|0777)
	if err != nil {
		return nil, err
	}
	cfg.DataDir = coinNetDir
	switch net {
	case "regtest", "simnet":
		cfg.Params = &chaincfg.RegressionNetParams
		cfg.TrustedPeer = electrumx.ServerAddr{
			Net: "tcp", Addr: "bch.imaginary.cash:50001", //"127.0.0.1:53002",
		}
		cfg.StoreEncSeed = true
		cfg.Testing = true
	case "testnet":
		cfg.Params = &chaincfg.TestNet3Params
		cfg.TrustedPeer = electrumx.ServerAddr{
			Net: "tcp", Addr: "testnet.aranguren.org:51001",
		}
		cfg.StoreEncSeed = true
		cfg.Testing = true
	case "mainnet":
		cfg.Params = &chaincfg.MainNetParams
		cfg.TrustedPeer = electrumx.ServerAddr{
			Net: "tcp", Addr: "bch.imaginary.cash:50001", //"elx.bitske.com:50002",
		}
		cfg.StoreEncSeed = false
		cfg.Testing = false
	}
	return cfg, nil
}

func configure() (*client.ClientConfig, error) {
	coin := flag.String("coin", "btc", "coin name")
	net := flag.String("net", "mainnet", "network type; testnet, mainnet, regtest")
	flag.Parse()
	fmt.Println("coin:", *coin)
	fmt.Println("net:", *net)
	return makeBasicConfig(*coin, *net)
}

func main() {
	cfg, err := configure()
	if err != nil {
		fmt.Println(err, " - exiting")
		os.Exit(1)
	}
	fmt.Println(cfg.Chain, cfg.Params.Name)

	// make basic client
	ec := btc.NewBtcElectrumClient(cfg)

	// make the client's node
	ec.CreateNode(client.SingleNode)
	err = ec.GetNode().Start()
	if err != nil {
		fmt.Println(err, " - exiting")
		os.Exit(1)
	}

	// get up to date with client's copy of the needed blockchain headers
	fmt.Println("syncing headers")
	err = ec.SyncHeaders()
	if err != nil {
		ec.GetNode().Stop()
		fmt.Println(err, " - exiting")
		os.Exit(1)
	}

	// make the client's wallet
	var mnemonic = "jungle pair grass super coral bubble tomato sheriff pulp cancel luggage wagon"
	err = ec.RecreateWallet("abc", mnemonic)

	//or

	// load the client's wallet
	// err = ec.LoadWallet("abc")
	if err != nil {
		ec.GetNode().Stop()
		fmt.Println(err, " - exiting")
		os.Exit(1)
	}

	// set up Notify for all our receive addresses and get any changes to the
	// state of the transaction history from the node
	err = ec.SyncWallet()
	if err != nil {
		ec.GetNode().Stop()
		fmt.Println(err, " - exiting")
		os.Exit(1)
	}

	//TODO:

	sc := ec.GetNode().GetServerConn().SvrConn
	<-sc.Done()
}
