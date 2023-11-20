package btc

import (
	"errors"
	"fmt"
	"os"
	"path"

	"main/client"
	"main/electrumx"
	"main/electrumx/elxbtc"
	"main/wallet"
	"main/wallet/db"
	"main/wallet/wltbtc"
)

// BtcElectrumClient
type BtcElectrumClient struct {
	ClientConfig *client.ClientConfig
	Wallet       wallet.ElectrumWallet
	Node         electrumx.ElectrumXNode
	// client copy of blockchain headers
	clientHeaders *Headers
	// client wallet receive address synchronization with the node
	walletSynchronizer *AddressSynchronizer
}

func NewBtcElectrumClient(cfg *client.ClientConfig) client.ElectrumClient {
	ec := BtcElectrumClient{
		ClientConfig: cfg,
		Wallet:       nil,
		Node:         nil,
	}
	ec.clientHeaders = NewHeaders(cfg)
	ec.walletSynchronizer = NewWalletSychronizer(cfg)
	return &ec
}

//////////////////////////////////////////////////////////////////////////////
// Interface
////////////

func (ec *BtcElectrumClient) GetConfig() *client.ClientConfig {
	return ec.ClientConfig
}

func (ec *BtcElectrumClient) GetWallet() wallet.ElectrumWallet {
	return ec.Wallet
}

func (ec *BtcElectrumClient) GetNode() electrumx.ElectrumXNode {
	return ec.Node
}

// CreateWallet makes a new wallet with a new seed. The password is to encrypt
// stored xpub, xprv and other sensitive data.
func (ec *BtcElectrumClient) CreateWallet(pw string) error {
	cfg := ec.ClientConfig
	datadir := ec.ClientConfig.DataDir
	if _, err := os.Stat(path.Join(datadir, "wallet.db")); err == nil {
		if !ec.ClientConfig.Testing {
			return errors.New("wallet.db already exists")
		}
		fmt.Printf("a file wallet.db probably exists in the datadir: %s .. \n"+
			"test will overwrite\n", cfg.DataDir)
	}

	// Select wallet datastore
	sqliteDatastore, err := db.Create(cfg.DataDir)
	if err != nil {
		return err
	}
	cfg.DB = sqliteDatastore

	walletCfg := cfg.MakeWalletConfig()
	ec.Wallet, err = wltbtc.NewBtcElectrumWallet(walletCfg, pw)
	if err != nil {
		return err
	}
	return nil
}

// RecreateWallet recreates a wallet from an existing mnemonic seed.
// The password is to encrypt the stored xpub, xprv and other sensitive data
// and can be different from the original wallet's password.
func (ec *BtcElectrumClient) RecreateWallet(pw, mnenomic string) error {
	cfg := ec.ClientConfig
	datadir := ec.ClientConfig.DataDir
	if _, err := os.Stat(path.Join(datadir, "wallet.db")); err == nil {
		if !ec.ClientConfig.Testing {
			return errors.New("wallet.db already exists")
		}
		fmt.Printf("a file wallet.db probably exists in the datadir: %s .. \n"+
			"test will overwrite\n", cfg.DataDir)
	}

	// Select wallet datastore
	sqliteDatastore, err := db.Create(cfg.DataDir)
	if err != nil {
		return err
	}
	cfg.DB = sqliteDatastore

	walletCfg := cfg.MakeWalletConfig()
	ec.Wallet, err = wltbtc.RecreateElectrumWallet(walletCfg, pw, mnenomic)
	if err != nil {
		return err
	}
	return nil
}

// LoadWallet loads an existing wallet. The password is required to decrypt
// the stored xpub, xprv and other sensitive data
func (ec *BtcElectrumClient) LoadWallet(pw string) error {
	cfg := ec.ClientConfig

	// Select wallet datastore
	sqliteDatastore, err := db.Create(cfg.DataDir)
	if err != nil {
		return err
	}
	cfg.DB = sqliteDatastore

	walletCfg := cfg.MakeWalletConfig()
	ec.Wallet, err = wltbtc.LoadBtcElectrumWallet(walletCfg, pw)
	if err != nil {
		return err
	}
	return nil
}

// CreateNode creates a single unconnected ElectrumX node
func (ec *BtcElectrumClient) CreateNode(_ client.NodeType) {
	nodeCfg := ec.GetConfig().MakeNodeConfig()
	n := elxbtc.NewSingleNode(nodeCfg)
	ec.Node = n
}

// Interface methods in client_headers.go
//
// SyncHeaders() error

// Interface methods in client_wallet.go
//
// SyncWallet() error
// Broadcast(rawTx string) (string, error)

//////////////////////////////////////////////////////////////////////////////
