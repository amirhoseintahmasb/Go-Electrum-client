package client

import (
	"net"
	"net/url"
	"os"
	"path/filepath"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"golang.org/x/net/proxy"

	"main/electrumx"
	"main/wallet"
)

const (
	appName = "goele"
)

type ClientConfig struct {
	// The blockchain, Bitcoin, Dash, etc
	Chain wallet.CoinType

	// Network parameters. Set mainnet, testnet using this.
	Params *chaincfg.Params

	// Store the seed in encrypted storage
	StoreEncSeed bool

	// The user-agent that shall be visible to the network
	UserAgent string

	// Location of the data directory
	DataDir string

	// An implementation of the Datastore interface
	DB wallet.Datastore

	// If you wish to connect to a single trusted electrumX peer server set this.
	// SingleNode servers will error if not provided
	TrustedPeer net.Addr

	// A Tor proxy can be set here causing the wallet will use Tor. TODO:
	Proxy proxy.Dialer

	// The default fee-per-byte for each level
	LowFee    uint64
	MediumFee uint64
	HighFee   uint64

	// The highest allowable fee-per-byte
	MaxFee uint64

	// External API to query to look up fees. If this field is nil then the default fees will be used.
	// If the API is unreachable then the default fees will likewise be used. If the API returns a fee
	// greater than MaxFee then the MaxFee will be used in place. The API response must be formatted as
	// { "fastestFee": 40, "halfHourFee": 20, "hourFee": 10 }
	FeeAPI url.URL

	// Disable the exchange rate provider
	DisableExchangeRates bool

	// If not testing do not overwrite existing wallet files
	Testing bool
}

func NewDefaultConfig() *ClientConfig {
	return &ClientConfig{
		Chain:                wallet.Bitcoin,
		Params:               &chaincfg.MainNetParams,
		UserAgent:            appName,
		DataDir:              btcutil.AppDataDir(appName, false),
		DB:                   nil, // concrete impl
		DisableExchangeRates: true,
	}
}
func (cc *ClientConfig) MakeWalletConfig() *wallet.WalletConfig {
	wc := wallet.WalletConfig{
		Chain:        cc.Chain,
		Params:       cc.Params,
		StoreEncSeed: cc.StoreEncSeed,
		DataDir:      cc.DataDir,
		DB:           cc.DB,
		LowFee:       cc.LowFee,
		MediumFee:    cc.MediumFee,
		HighFee:      cc.HighFee,
		MaxFee:       cc.MaxFee,
		Testing:      cc.Testing,
	}
	return &wc
}

func (cc *ClientConfig) MakeNodeConfig() *electrumx.NodeConfig {
	nc := electrumx.NodeConfig{
		Chain:       cc.Chain,
		Params:      cc.Params,
		UserAgent:   cc.UserAgent,
		DataDir:     cc.DataDir,
		TrustedPeer: cc.TrustedPeer,
		Proxy:       cc.Proxy,
		Testing:     cc.Testing,
	}
	return &nc
}

func GetConfigPath() (string, error) {
	userCfgDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	appPath := filepath.Join(userCfgDir, appName)
	err = os.MkdirAll(appPath, os.ModeDir|0777)
	if err != nil {
		return "", err
	}
	return appPath, nil
}
