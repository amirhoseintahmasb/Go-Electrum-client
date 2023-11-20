package btc

import (
	"fmt"

	"github.com/btcsuite/btcd/txscript"
)

// Here is the client interface between the node & wallet for transaction
// broadcast and wallet synchronize

// devdbg: just one known wallet address
func (ec *BtcElectrumClient) SyncWallet() error {

	// just get 1st adddress in wallet
	addresses := ec.GetWallet().ListAddresses()
	address := addresses[0]

	fmt.Println(address.String())

	err := ec.SubscribeAddressNotify(address)
	if err != nil {
		return err
	}
	script, err := txscript.PayToAddrScript(address)
	if err != nil {
		// possibly non-standard: future
		return err
	}
	// AddWatchedScript adds the pay script to db. If it already exists this
	// is a no-op
	err = ec.GetWallet().AddWatchedScript(script)
	if err != nil {
		return err
	}

	history, err := ec.GetAddressHistory(address)
	if err != nil {
		return err
	}
	dumpHistory(address, history)

	// start goroutine to listen for scripthash status change notifications arriving
	err = ec.addressStatusNotify()
	if err != nil {
		return err
	}

	return nil
}

//-------------------------------------------------
// func (ec *BtcElectrumClient) SyncWallet() error {

// 	// - get all receive addresses in wallet
// 	addresses := ec.GetWallet().ListAddresses()

// 	// for each
// 	//   - subscribe for scripthash notifications
// 	//   - on sub the return is hash of all known history known to server
// 	//   - get the prev stored history for the address from db if any
// 	//   - hash prev history and compare to what the server sends back
// 	//   - if different get the up to date history list of txid:height
// 	//     - update db

// 	for _, address := range addresses {

// 		// // fun with dick & jane
// 		// script := address.ScriptAddress()
// 		// s := hex.EncodeToString(script)
// 		// fmt.Println("ScriptAddress", s)
// 		// segwitAddress, swerr := btcutil.NewAddressWitnessPubKeyHash(script, ec.GetConfig().Params)
// 		// if swerr != nil {
// 		// 	fmt.Println(swerr)
// 		// 	continue
// 		// }
// 		// fmt.Println("segwitAddress", segwitAddress.String())
// 		// fmt.Println("segwitAddress", segwitAddress.EncodeAddress())
// 		// address = segwitAddress
// 		// // end fun

// 		err := ec.SubscribeAddressNotify(address)
// 		if err != nil {
// 			return err
// 		}
// 		// AddWatchedScript adds the pkscript to db. If it already exists this
// 		// is a no-op
// 		err = ec.GetWallet().AddWatchedScript(address.ScriptAddress())
// 		if err != nil {
// 			return err
// 		}
// 	}

// 	// start goroutine to listen for scripthash status change notifications arriving

// 	return nil
// }
//------------------------------------------

//////////////////////////////////////////////////////////////////////////////
// Python console
/////////////////

// Broadcast sends a transaction to the server for broadcast on the bitcoin
// network
func (ec *BtcElectrumClient) Broadcast(rawTx string) (string, error) {
	return ec.GetNode().Broadcast(rawTx)
}
