package wltbtc

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"main/client"
	"main/wallet"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcwallet/wallet/txrules"
	"github.com/tyler-smith/go-bip39"
)

//////////////////////////////////////////////////////////////////////////////
//	ElectrumWallet

// BtcElectrumWallet implements ElectrumWallet

// TODO: adjust interface while developing because .. simpler
var _ = wallet.ElectrumWallet(&BtcElectrumWallet{})

const WalletVersion = "0.1.0"

type BtcElectrumWallet struct {
	params *chaincfg.Params

	masterPrivateKey *hdkeychain.ExtendedKey
	masterPublicKey  *hdkeychain.ExtendedKey

	feeProvider *wallet.FeeProvider

	repoPath string

	// TODO: maybe a scaled down blockchain with headers of interest to wallet?
	// blockchain  *Blockchain
	storageManager *StorageManager
	txstore        *TxStore
	keyManager     *KeyManager

	mutex *sync.RWMutex

	creationDate time.Time

	running bool
}

// NewBtcElectrumWallet mskes new wallet with a new seed. The Mnemonic should
// be saved offline by the user.
func NewBtcElectrumWallet(config *wallet.WalletConfig, pw string) (*BtcElectrumWallet, error) {
	if pw == "" {
		return nil, errors.New("empty password")
	}

	ent, err := bip39.NewEntropy(128)
	if err != nil {
		return nil, err
	}
	mnemonic, err := bip39.NewMnemonic(ent)
	if err != nil {
		return nil, err
	}
	// TODO: dbg remove
	fmt.Println("Save: ", mnemonic)

	seed := bip39.NewSeed(mnemonic, "")

	return makeBtcElectrumWallet(config, pw, seed)
}

// RecreateElectrumWallet mskes new wallet with a mnenomic seed from an existing wallet.
// pw does not need to be the same as the old wallet
func RecreateElectrumWallet(config *wallet.WalletConfig, pw, mnemonic string) (*BtcElectrumWallet, error) {
	if pw == "" {
		return nil, errors.New("empty password")
	}
	seed, err := bip39.NewSeedWithErrorChecking(mnemonic, "")
	if err != nil {
		return nil, err
	}

	return makeBtcElectrumWallet(config, pw, seed)
}

func LoadBtcElectrumWallet(config *wallet.WalletConfig, pw string) (*BtcElectrumWallet, error) {
	if pw == "" {
		return nil, errors.New("empty password")
	}

	return loadBtcElectrumWallet(config, pw)
}

func makeBtcElectrumWallet(config *wallet.WalletConfig, pw string, seed []byte) (*BtcElectrumWallet, error) {

	// dbg
	fmt.Println("seed: ", hex.EncodeToString(seed))

	mPrivKey, err := hdkeychain.NewMaster(seed, config.Params)
	if err != nil {
		return nil, err
	}
	mPubKey, err := mPrivKey.Neuter()
	if err != nil {
		return nil, err
	}
	w := &BtcElectrumWallet{
		repoPath:         config.DataDir,
		masterPrivateKey: mPrivKey,
		masterPublicKey:  mPubKey,
		params:           config.Params,
		creationDate:     time.Now(),
		feeProvider: wallet.NewFeeProvider(
			config.MaxFee,
			config.HighFee,
			config.MediumFee,
			config.LowFee,
			// move to client
			"",
			nil,
			// config.FeeAPI.String(),
			// config.Proxy,
		),
		mutex: new(sync.RWMutex),
	}

	sm := NewStorageManager(config.DB.Enc(), config.Params)
	sm.store.Version = "0,1"
	sm.store.Xprv = mPrivKey.String()
	sm.store.Xpub = mPubKey.String()
	if config.StoreEncSeed {
		sm.store.Seed = make([]byte, len(seed))
		copy(sm.store.Seed, seed)
	}
	err = sm.Put(pw)
	if err != nil {
		return nil, err
	}
	w.storageManager = sm

	w.keyManager, err = NewKeyManager(config.DB.Keys(), w.params, w.masterPrivateKey)
	if err != nil {
		return nil, err
	}

	w.txstore, err = NewTxStore(w.params, config.DB, w.keyManager)
	if err != nil {
		return nil, err
	}

	err = config.DB.Cfg().PutCreationDate(w.creationDate)
	if err != nil {
		return nil, err
	}

	// Debug: remove
	if config.Params != &chaincfg.MainNetParams {
		fmt.Println("Created: ", w.creationDate)
		fmt.Println(hex.EncodeToString(sm.store.Seed))
		fmt.Println("Created Addresses:")
		for i, adr := range w.txstore.adrs {
			fmt.Printf("%d %v\n", i, adr)
			if i == client.LOOKAHEADWINDOW-1 {
				fmt.Println(" ---")
			}
		}
	}
	return w, nil
}

func loadBtcElectrumWallet(config *wallet.WalletConfig, pw string) (*BtcElectrumWallet, error) {

	sm := NewStorageManager(config.DB.Enc(), config.Params)

	err := sm.Get(pw)
	if err != nil {
		return nil, err
	}

	mPrivKey, err := hdkeychain.NewKeyFromString(sm.store.Xprv)
	if err != nil {
		return nil, err
	}
	mPubKey, err := hdkeychain.NewKeyFromString(sm.store.Xpub)
	if err != nil {
		return nil, err
	}

	w := &BtcElectrumWallet{
		repoPath:         config.DataDir,
		masterPrivateKey: mPrivKey,
		masterPublicKey:  mPubKey,
		storageManager:   sm,
		params:           config.Params,
		feeProvider: wallet.NewFeeProvider(
			config.MaxFee,
			config.HighFee,
			config.MediumFee,
			config.LowFee,
			// move to client
			"",
			nil,
			// config.FeeAPI.String(),
			// config.Proxy,
		),
		mutex: new(sync.RWMutex),
	}

	w.keyManager, err = NewKeyManager(config.DB.Keys(), w.params, w.masterPrivateKey)
	if err != nil {
		return nil, err
	}

	w.txstore, err = NewTxStore(w.params, config.DB, w.keyManager)
	if err != nil {
		return nil, err
	}

	w.creationDate, err = config.DB.Cfg().GetCreationDate()
	if err != nil {
		return nil, err
	}

	// Debug: remove
	if config.Params != &chaincfg.MainNetParams {
		fmt.Println("Stored Creation Date: ", w.creationDate)
		fmt.Println(hex.EncodeToString(sm.store.Seed))
		fmt.Println("Loaded Addresses:")
		for i, adr := range w.txstore.adrs {
			fmt.Printf("%d %v\n", i, adr)
			if i == client.LOOKAHEADWINDOW-1 {
				fmt.Println(" ---")
			}
		}
	}

	return w, nil
}

func (w *BtcElectrumWallet) Start() {
	w.running = true

	/* start the Chain Manager here maybe */
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//
// API
//
//////////////

func (w *BtcElectrumWallet) CreationDate() time.Time {
	return w.creationDate
}

func (w *BtcElectrumWallet) CurrencyCode() string {
	if w.params.Name == chaincfg.MainNetParams.Name {
		return "btc"
	} else {
		return "tbtc"
	}
}

func (w *BtcElectrumWallet) IsDust(amount int64) bool {
	// This is a per mempool policy thing .. < 1000 sats for now
	return btcutil.Amount(amount) < txrules.DefaultRelayFeePerKb
}

func (w *BtcElectrumWallet) MasterPrivateKey() *hdkeychain.ExtendedKey {
	return w.masterPrivateKey
}

func (w *BtcElectrumWallet) MasterPublicKey() *hdkeychain.ExtendedKey {
	return w.masterPublicKey
}

func (w *BtcElectrumWallet) ChildKey(keyBytes []byte, chaincode []byte, isPrivateKey bool) (*hdkeychain.ExtendedKey, error) {
	parentFP := []byte{0x00, 0x00, 0x00, 0x00}
	var id []byte
	if isPrivateKey {
		id = w.params.HDPrivateKeyID[:]
	} else {
		id = w.params.HDPublicKeyID[:]
	}
	hdKey := hdkeychain.NewExtendedKey(
		id,
		keyBytes,
		chaincode,
		parentFP,
		0,
		0,
		isPrivateKey)
	return hdKey.Derive(0)
}

func (w *BtcElectrumWallet) CurrentAddress(purpose wallet.KeyPurpose) btcutil.Address {
	key, _ := w.keyManager.GetCurrentKey(purpose)
	addr, _ := key.Address(w.params)
	return btcutil.Address(addr)
}

func (w *BtcElectrumWallet) NewAddress(purpose wallet.KeyPurpose) btcutil.Address {
	i, _ := w.txstore.Keys().GetUnused(purpose)
	key, _ := w.keyManager.generateChildKey(purpose, uint32(i[1]))
	addr, _ := key.Address(w.params)
	w.txstore.Keys().MarkKeyAsUsed(addr.ScriptAddress())
	w.txstore.PopulateAdrs()
	return btcutil.Address(addr)
}

func (w *BtcElectrumWallet) DecodeAddress(addr string) (btcutil.Address, error) {
	return btcutil.DecodeAddress(addr, w.params)
}

func (w *BtcElectrumWallet) ScriptToAddress(script []byte) (btcutil.Address, error) {
	return scriptToAddress(script, w.params)
}

func scriptToAddress(script []byte, params *chaincfg.Params) (btcutil.Address, error) {
	_, addrs, _, err := txscript.ExtractPkScriptAddrs(script, params)
	if err != nil {
		return &btcutil.AddressPubKeyHash{}, err
	}
	if len(addrs) == 0 {
		return &btcutil.AddressPubKeyHash{}, errors.New("unknown script")
	}
	return addrs[0], nil
}

func (w *BtcElectrumWallet) AddressToScript(addr btcutil.Address) ([]byte, error) {
	return txscript.PayToAddrScript(addr)
}

func (w *BtcElectrumWallet) HasKey(addr btcutil.Address) bool {
	_, err := w.keyManager.GetKeyForScript(addr.ScriptAddress())
	return err == nil
}

func (w *BtcElectrumWallet) GetKey(addr btcutil.Address) (*btcec.PrivateKey, error) {
	key, err := w.keyManager.GetKeyForScript(addr.ScriptAddress())
	if err != nil {
		return nil, err
	}
	return key.ECPrivKey()
}

func (w *BtcElectrumWallet) ListAddresses() []btcutil.Address {
	keys := w.keyManager.GetKeys()
	addrs := []btcutil.Address{}
	for _, k := range keys {
		addr, err := k.Address(w.params)
		if err != nil {
			continue
		}
		addrs = append(addrs, addr)
	}
	return addrs
}

func (w *BtcElectrumWallet) ListKeys() []btcec.PrivateKey {
	keys := w.keyManager.GetKeys()
	list := []btcec.PrivateKey{}
	for _, k := range keys {
		priv, err := k.ECPrivKey()
		if err != nil {
			continue
		}
		list = append(list, *priv)
	}
	return list
}

func (w *BtcElectrumWallet) Balance() (confirmed, unconfirmed int64) {
	utxos, _ := w.txstore.Utxos().GetAll()
	stxos, _ := w.txstore.Stxos().GetAll()
	for _, utxo := range utxos {
		if !utxo.WatchOnly {
			if utxo.AtHeight > 0 {
				confirmed += utxo.Value
			} else {
				if w.checkIfStxoIsConfirmed(utxo, stxos) {
					confirmed += utxo.Value
				} else {
					unconfirmed += utxo.Value
				}
			}
		}
	}
	return confirmed, unconfirmed
}

func (w *BtcElectrumWallet) Transactions() ([]wallet.Txn, error) {
	height := w.ChainTip()
	txns, err := w.txstore.Txns().GetAll(false)
	if err != nil {
		return txns, err
	}
	for i, tx := range txns {
		var confirmations int64
		var status wallet.StatusCode
		confs := height - tx.Height + 1
		if tx.Height <= 0 {
			confs = tx.Height
		}
		switch {
		case confs < 0:
			status = wallet.StatusDead
		case confs == 0 && time.Since(tx.Timestamp) <= time.Hour*6:
			status = wallet.StatusUnconfirmed
		case confs == 0 && time.Since(tx.Timestamp) > time.Hour*6:
			status = wallet.StatusDead
		case confs > 0 && confs < 6:
			status = wallet.StatusPending
			confirmations = confs
		case confs > 5:
			status = wallet.StatusConfirmed
			confirmations = confs
		}
		tx.Confirmations = confirmations
		tx.Status = status
		txns[i] = tx
	}
	return txns, nil
}
func (w *BtcElectrumWallet) HasTransaction(txid chainhash.Hash) bool {
	_, err := w.txstore.Txns().Get(txid)
	// error only for 'no rows in rowset'
	return err == nil
}

func (w *BtcElectrumWallet) GetTransaction(txid chainhash.Hash) (wallet.Txn, error) {
	txn, err := w.txstore.Txns().Get(txid)
	if err == nil {
		tx := wire.NewMsgTx(1)
		rbuf := bytes.NewReader(txn.Bytes)
		err := tx.BtcDecode(rbuf, wire.ProtocolVersion, wire.WitnessEncoding)
		if err != nil {
			return txn, err
		}
		outs := []wallet.TransactionOutput{}
		for i, out := range tx.TxOut {
			var addr btcutil.Address
			_, addrs, _, err := txscript.ExtractPkScriptAddrs(out.PkScript, w.params)
			if err != nil {
				fmt.Printf("error extracting address from txn pkscript: %v\n", err)
				return txn, err
			}
			if len(addrs) == 0 {
				addr = nil
			} else {
				addr = addrs[0]
			}
			tout := wallet.TransactionOutput{
				Address: addr,
				Value:   out.Value,
				Index:   uint32(i),
			}
			outs = append(outs, tout)
		}
		txn.Outputs = outs
	}
	return txn, err
}

func (w *BtcElectrumWallet) GetConfirmations(txid chainhash.Hash) (int64, int64, error) {
	txn, err := w.txstore.Txns().Get(txid)
	if err != nil {
		return 0, 0, err
	}
	if txn.Height == 0 {
		return 0, 0, nil
	}
	chainTip := w.ChainTip()
	return chainTip - txn.Height + 1, txn.Height, nil
}

func (w *BtcElectrumWallet) checkIfStxoIsConfirmed(utxo wallet.Utxo, stxos []wallet.Stxo) bool {
	for _, stxo := range stxos {
		if !stxo.Utxo.WatchOnly {
			if stxo.SpendTxid.IsEqual(&utxo.Op.Hash) {
				if stxo.SpendHeight > 0 {
					return true
				} else {
					return w.checkIfStxoIsConfirmed(stxo.Utxo, stxos)
				}
			} else if stxo.Utxo.IsEqual(&utxo) {
				if stxo.Utxo.AtHeight > 0 {
					return true
				} else {
					return false
				}
			}
		}
	}
	return false
}

func (w *BtcElectrumWallet) Params() *chaincfg.Params {
	return w.params
}

func (w *BtcElectrumWallet) ChainTip() int64 {
	// not yet implemented - Get from ElectrumX
	return 0
}

func (w *BtcElectrumWallet) ExchangeRates() wallet.ExchangeRates {
	// not yet implemented
	return nil
}

// Get the current fee per byte
func (w *BtcElectrumWallet) GetFeePerByte(feeLevel wallet.FeeLevel) uint64 {
	// not yet implemented
	return 0
}

// Send bitcoins to an external wallet
func (w *BtcElectrumWallet) Spend(amount int64, addr btcutil.Address, feeLevel wallet.FeeLevel) (*chainhash.Hash, error) {
	// not yet implemented
	return nil, wallet.ErrWalletFnNotImplemented
}

// Bump the fee for the given transaction
func (w *BtcElectrumWallet) BumpFee(txid chainhash.Hash) (*chainhash.Hash, error) {
	return nil, wallet.ErrWalletFnNotImplemented
}

// Calculates the estimated size of the transaction and returns the total fee for the given feePerByte
func (w *BtcElectrumWallet) EstimateFee(ins []wallet.TransactionInput, outs []wallet.TransactionOutput, feePerByte uint64) uint64 {
	// not yet implemented
	return 0
}

// Build and broadcast a transaction that sweeps all coins from an address. If it is a p2sh multisig, the redeemScript must be included
func (w *BtcElectrumWallet) SweepAddress(utxos []wallet.Utxo, address *btcutil.Address, key *hdkeychain.ExtendedKey, redeemScript *[]byte, feeLevel wallet.FeeLevel) (*chainhash.Hash, error) {
	// not yet implemented
	return nil, wallet.ErrWalletFnNotImplemented
}

// Create a signature for a multisig transaction
func (w *BtcElectrumWallet) CreateMultisigSignature(ins []wallet.TransactionInput, outs []wallet.TransactionOutput, key *hdkeychain.ExtendedKey, redeemScript []byte, feePerByte uint64) ([]wallet.Signature, error) {
	// not yet implemented
	return nil, wallet.ErrWalletFnNotImplemented
}

// Combine signatures and optionally broadcast
func (w *BtcElectrumWallet) Multisign(ins []wallet.TransactionInput, outs []wallet.TransactionOutput, sigs1 []wallet.Signature, sigs2 []wallet.Signature, redeemScript []byte, feePerByte uint64, broadcast bool) ([]byte, error) {
	// not yet implemented
	return nil, wallet.ErrWalletFnNotImplemented
}

// Generate a multisig script from public keys. If a timeout is included the returned script should be a timelocked escrow which releases using the timeoutKey.
func (w *BtcElectrumWallet) GenerateMultisigScript(keys []hdkeychain.ExtendedKey, threshold int, timeout time.Duration, timeoutKey *hdkeychain.ExtendedKey) (addr btcutil.Address, redeemScript []byte, err error) {
	// not yet implemented
	return nil, nil, wallet.ErrWalletFnNotImplemented

}

// Add a script to the wallet and get notifications back when coins are received or spent from it
func (w *BtcElectrumWallet) AddWatchedScript(script []byte) error {
	err := w.txstore.WatchedScripts().Put(script)
	if err != nil {
		return err
	}
	err = w.txstore.PopulateAdrs()
	if err != nil {
		return err
	}
	return nil
}

// AddTransactionListener
func (w *BtcElectrumWallet) AddTransactionListener(listener func(wallet.TransactionCallback)) {
	// not yet implemented
}

// NotifyTransactionListners
func (w *BtcElectrumWallet) NotifyTransactionListners(cb wallet.TransactionCallback) {
	// not yet implemented
}

func (w *BtcElectrumWallet) ReSyncBlockchain(fromHeight uint64) {
	panic("ReSyncBlockchain: Not implemented - Non-SPV wallet")
}

func (w *BtcElectrumWallet) AddWatchedAddresses(addrs ...btcutil.Address) error {
	var err error
	var watchedScripts [][]byte

	for _, addr := range addrs {
		script, err := w.AddressToScript(addr)
		if err != nil {
			return err
		}
		watchedScripts = append(watchedScripts, script)
	}

	err = w.txstore.WatchedScripts().PutAll(watchedScripts)
	w.txstore.PopulateAdrs()

	// w.wireService.MsgChan() <- updateFiltersMsg{} // not SPV

	return err
}

func (w *BtcElectrumWallet) DumpHeaders(writer io.Writer) {
	// w.blockchain.db.Print(writer)
	panic("DumpHeaders: Non-SPV wallet")
}

func (w *BtcElectrumWallet) Close() {
	if w.running {
		// Any other tear down here
		w.running = false
	}
}
