package client

///////////////////////////////// Client interface ///////////////////////////
//
//	architecture
//
//	   Client
//
//	     /\
//	 (controller)
//	   /    \
//	  /      \
//	 /        \
//
// Wallet     Node
//
// The client interface describes the behaviors of the client controller.
// It is implemented for each coin asset client.

import (
	"main/electrumx"
	"main/wallet"
)

type NodeType int

const (
	// ElectrumX Server(s)
	SingleNode NodeType = iota
	MultiNode  NodeType = 1
)

const (
	// Electrum Wallet
	LOOKAHEADWINDOW = 10
)

type ElectrumClient interface {
	GetConfig() *ClientConfig
	GetWallet() wallet.ElectrumWallet
	GetNode() electrumx.ElectrumXNode
	//
	CreateNode(nodeType NodeType)
	//
	SyncHeaders() error
	SubscribeClientHeaders() error
	//
	CreateWallet(pw string) error
	RecreateWallet(pw, mnenomic string) error
	LoadWallet(pw string) error
	//
	SyncWallet() error
	//
	// Small subset of electrum python console methods
	Broadcast(rawTx string) (string, error)
	//...
}
