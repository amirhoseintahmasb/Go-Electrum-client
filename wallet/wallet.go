package wallet

import (
	"errors"
	"math/big"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	hd "github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

type WalletConfig struct {
	// The blockchain, Bitcoin, Dash, etc
	Chain CoinType

	// Network parameters. Set mainnet, testnet using this.
	Params *chaincfg.Params

	// Store the seed in encrypted storage
	StoreEncSeed bool

	// Location of the data directory
	DataDir string

	// An implementation of the Datastore interface
	DB Datastore

	// The default fee-per-byte for each level
	LowFee    uint64
	MediumFee uint64
	HighFee   uint64

	// The highest allowable fee-per-byte
	MaxFee uint64

	// If not testing do not overwrite existing wallet files
	Testing bool
}

type ElectrumWallet interface {

	// Start the wallet
	Start()

	// Return the network parameters
	Params() *chaincfg.Params

	// Returns the type of crytocurrency this wallet implements
	CurrencyCode() string

	// Check if this amount is considered dust < 1000 sats/equivalent for now
	IsDust(amount int64) bool

	// CurrentAddress returns an address suitable for receiving payments. `purpose` specifies
	// whether the address should be internal or external. External addresses are typically
	// requested when receiving funds from outside the wallet .Internal addresses are typically
	// change addresses. For utxo based coins we expect this function will return the same
	// address so long as that address is unused. Whenever the address receives a payment,
	// CurrentAddress should start returning a new, unused address.
	CurrentAddress(purpose KeyPurpose) btcutil.Address

	// NewAddress returns a new, never-before-returned address.
	NewAddress(purpose KeyPurpose) btcutil.Address

	// DecodeAddress parses the address string and return an address interface.
	DecodeAddress(addr string) (btcutil.Address, error)

	// ScriptToAddress takes a raw output script (the full script, not just a hash160) and
	// returns the corresponding address. This should be considered deprecated.
	ScriptToAddress(script []byte) (btcutil.Address, error)

	// Turn the given address into an output script
	AddressToScript(addr btcutil.Address) ([]byte, error)

	// Returns if the wallet has the key for the given address
	HasKey(addr btcutil.Address) bool

	// Balance returns the confirmed and unconfirmed aggregate balance for the wallet.
	// For utxo based wallets, if a spend of confirmed coins is made, the resulting "change"
	// should be also counted as confirmed even if the spending transaction is unconfirmed.
	Balance() (confirmed, unconfirmed int64)

	// Returns a list of addresses for this wallet
	ListAddresses() []btcutil.Address

	// Returns a list of transactions for this wallet
	Transactions() ([]Txn, error)

	// Does the wallet have a specific transaction
	HasTransaction(txid chainhash.Hash) bool

	// Get info on a specific transaction
	GetTransaction(txid chainhash.Hash) (Txn, error)

	// Return the number of confirmations and the height for a transaction
	GetConfirmations(txid chainhash.Hash) (confirms, atHeight int64, err error)

	// Get the height of the blockchain from chain manager
	ChainTip() int64

	// Get the current fee per byte
	GetFeePerByte(feeLevel FeeLevel) uint64

	// Send bitcoins to an external wallet
	Spend(amount int64, addr btcutil.Address, feeLevel FeeLevel) (*chainhash.Hash, error)

	// BumpFee should attempt to bump the fee on a given unconfirmed transaction (if possible) to
	// try to get it confirmed and return the txid of the new transaction (if one exists).
	// Since this method is only called in response to user action, it is acceptable to
	// return an error if this functionality is not available in this wallet or on the network.
	BumpFee(txid chainhash.Hash) (*chainhash.Hash, error)

	// Calculates the estimated size of the transaction and returns the total fee for the given feePerByte
	EstimateFee(ins []TransactionInput, outs []TransactionOutput, feePerByte uint64) uint64

	// Build and broadcast a transaction that sweeps all coins from an address. If it is a p2sh multisig, the redeemScript must be included
	SweepAddress(utxos []Utxo, address *btcutil.Address, key *hd.ExtendedKey, redeemScript *[]byte, feeLevel FeeLevel) (*chainhash.Hash, error)

	// Add a script to the wallet and get notifications back when coins are received or spent from it
	AddWatchedScript(script []byte) error

	// Add a callback for incoming transactions
	AddTransactionListener(func(TransactionCallback))

	// NotifyTransactionListners
	NotifyTransactionListners(cb TransactionCallback)

	// ReSyncBlockchain is called in response to a user action to rescan transactions. API based
	// wallets should do another scan of their addresses to find anything missing. Full node, or SPV
	// wallets should rescan/re-download blocks starting at the fromTime.
	// Get info from chain manager
	ReSyncBlockchain(fromHeight uint64)

	// Generate a multisig script from public keys. If a timeout is included the returned script should be a timelocked
	// escrow which releases using the timeoutKey.
	// GenerateMultisigScript should deterministically create a redeem script and address from the information provided.
	// This method should be strictly limited to taking the input data, combining it to produce the redeem script and
	// address
	GenerateMultisigScript(keys []hd.ExtendedKey, threshold int, timeout time.Duration, timeoutKey *hd.ExtendedKey) (addr btcutil.Address, redeemScript []byte, err error)

	// Create a signature for a multisig transaction
	CreateMultisigSignature(ins []TransactionInput, outs []TransactionOutput, key *hd.ExtendedKey, redeemScript []byte, feePerByte uint64) ([]Signature, error)

	// Combine signatures and optionally broadcast
	Multisign(ins []TransactionInput, outs []TransactionOutput, sigs1 []Signature, sigs2 []Signature, redeemScript []byte, feePerByte uint64, broadcast bool) ([]byte, error)

	// Cleanly disconnect from the wallet
	Close()
}

// Errors
var (
	ErrorDustAmount error = errors.New("amount is below network dust treshold")
	// ErrInsufficientFunds is returned when the wallet is unable to send the
	// amount specified due to the balance being too low
	ErrInsufficientFunds = errors.New("ERROR_INSUFFICIENT_FUNDS")

	// ErrWalletFnNotImplemented is returned from some unimplemented functions.
	// This is due to a concrete wallet not implementing the finctionality or
	// temporarily during development.
	ErrWalletFnNotImplemented = errors.New("wallet function is not implemented")
)

type FeeLevel int

const (
	PRIOIRTY       FeeLevel = 0
	NORMAL         FeeLevel = 1
	ECONOMIC       FeeLevel = 2
	FEE_BUMP       FeeLevel = 3
	SUPER_ECONOMIC FeeLevel = 4
)

// The end leaves on the HD wallet have only two possible values. External keys are those given
// to other people for the purpose of receiving transactions. These may include keys used for
// refund addresses. Internal keys are used only by the wallet, primarily for change addresses
// but could also be used for shuffling around UTXOs.
type KeyPurpose int

const (
	EXTERNAL KeyPurpose = 0
	INTERNAL KeyPurpose = 1
)

// This callback is passed to any registered transaction listeners when a transaction is detected
// for the wallet.
type TransactionCallback struct {
	Txid      string
	Outputs   []TransactionOutput
	Inputs    []TransactionInput
	Height    int64
	Timestamp time.Time
	Value     int64
	WatchOnly bool
	BlockTime time.Time
}

type TransactionOutput struct {
	Address btcutil.Address
	Value   int64
	Index   uint32
	OrderID string
}

type TransactionInput struct {
	OutpointHash  []byte
	OutpointIndex uint32
	LinkedAddress btcutil.Address
	Value         int64
	OrderID       string
}

// OpenBazaar uses p2sh addresses for escrow. This object can be used to store a record of a
// transaction going into or out of such an address. Incoming transactions should have a positive
// value and be market as spent when the UXTO is spent. Outgoing transactions should have a
// negative value. The spent field isn't relevant for outgoing transactions.
type TransactionRecord struct {
	Txid      string
	Index     uint32
	Value     big.Int
	Address   string
	Spent     bool
	Timestamp time.Time
}

// This object contains a single signature for a multisig transaction. InputIndex specifies
// the index for which this signature applies.
type Signature struct {
	InputIndex uint32
	Signature  []byte
}
