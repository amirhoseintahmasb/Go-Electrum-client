package electrumx

import (
	"context"
	"net"

	"main/wallet"

	"github.com/btcsuite/btcd/chaincfg"
	"golang.org/x/net/proxy"
)

type ServerAddr struct {
	Net, Addr string
}

func (a ServerAddr) Network() string {
	return a.Net
}
func (a ServerAddr) String() string {
	return a.Addr
}

// Ensure simpleAddr implements the net.Addr interface.
var _ net.Addr = ServerAddr{}

type NodeConfig struct {
	// The blockchain, Bitcoin, Dash, etc
	Chain wallet.CoinType

	// Network parameters. Set mainnet, testnet using this.
	Params *chaincfg.Params

	// The user-agent that shall be visible to the network
	UserAgent string

	// Location of the data directory
	DataDir string

	// If you wish to connect to a single trusted electrumX peer set this.
	TrustedPeer net.Addr

	// A Tor proxy can be set here causing the wallet will use Tor. TODO:
	Proxy proxy.Dialer

	// If not testing do not overwrite existing wallet files
	Testing bool
}

type Network string

var Regtest Network = "regtest"
var Testnet Network = "testnet"
var Mainnet Network = "mainnet"

var DebugMode bool

type ElectrumXNode interface {
	Start() error
	Stop()
	GetServerConn() *ElectrumXSvrConn
	GetHeadersNotify() (<-chan *HeadersNotifyResult, error)
	SubscribeHeaders() (*HeadersNotifyResult, error)
	BlockHeaders(startHeight, blockCount uint32) (*GetBlockHeadersResult, error)
	GetScripthashNotify() (<-chan *ScripthashStatusResult, error)
	SubscribeScripthashNotify(scripthash string) (*ScripthashStatusResult, error)
	UnsubscribeScripthashNotify(scripthash string)
	GetHistory(scripthash string) (HistoryResult, error)
	//
	Broadcast(rawTx string) (string, error)
}

type ElectrumXSvrConn struct {
	SvrCtx  context.Context
	SvrConn *ServerConn
	Running bool
}
