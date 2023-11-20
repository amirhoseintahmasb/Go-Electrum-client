package btc

import (
	"encoding/hex"
	"fmt"
	"log"
	"time"
)

// SyncHeaders uodates the client headers and then subscribes for new update
// tip notifications and listens for them
func (ec *BtcElectrumClient) SyncHeaders() error {
	err := ec.SyncClientHeaders()
	if err != nil {
		return err
	}
	return ec.SubscribeClientHeaders()
}

// SyncClientHeaders reads blockchain_headers file, then gets any missing block from
// end of file to current tip from server. The current set of headers is also
// stored in headers map and the chain verified by checking previous block
// hashes backwards from Tip.
// SyncClientHeaders is part of the ElectrumClient interface inmplementation
func (ec *BtcElectrumClient) SyncClientHeaders() error {
	h := ec.clientHeaders

	// 1. Read last stored blockchain_headers file for this network

	b, err := h.ReadAllBytesFromFile()
	if err != nil {
		return err
	}
	lb := len(b)
	fmt.Println("read header bytes", lb)
	numHeaders, err := h.BytesToNumHdrs(lb)
	if err != nil {
		return err
	}
	b = nil // gc

	maybeTip := numHeaders - 1

	// 2. Gather new block headers we did not have in file up to current tip

	// Do not make block count too big or electrumX may throttle response
	// as an anti ddos measure. Magic number 2016 from electrum code
	const blockDelta = 20 // 20 dev 2016 pro
	var doneGathering = false
	var startHeight = uint32(numHeaders)
	var blockCount = uint32(20)

	node := ec.GetNode()

	hdrsRes, err := node.BlockHeaders(startHeight, blockCount)
	if err != nil {
		return err
	}
	count := hdrsRes.Count

	fmt.Print("Count: ", count, " read from server at Height: ", startHeight)

	if count > 0 {
		b, err := hex.DecodeString(hdrsRes.HexConcat)
		if err != nil {
			log.Fatal(err)
		}
		nh, err := h.AppendHeaders(b)
		if err != nil {
			log.Fatal(err)
		}
		maybeTip += int32(count)

		fmt.Println(" Appended: ", nh, " headers at ", startHeight, " maybeTip ", maybeTip)
	}

	if count < blockDelta {
		fmt.Println("\nDone gathering")
		doneGathering = true
	}

	svrCtx := node.GetServerConn().SvrCtx

	for !doneGathering {

		startHeight += blockDelta

		select {

		case <-svrCtx.Done():
			fmt.Println("Server shutdown - gathering")
			node.Stop()
			return nil

		case <-time.After(time.Millisecond * 33):
			hdrsRes, err := node.BlockHeaders(startHeight, blockCount)
			if err != nil {
				return err
			}
			count = hdrsRes.Count

			fmt.Print("Count: ", count, " read from server at Height: ", startHeight)

			if count > 0 {
				b, err := hex.DecodeString(hdrsRes.HexConcat)
				if err != nil {
					return err
				}
				nh, err := h.AppendHeaders(b)
				if err != nil {
					return err
				}
				maybeTip += int32(count)

				fmt.Println(" Appended: ", nh, " headers at ", startHeight, " maybeTip ", maybeTip)
			}

			if count < blockDelta {
				fmt.Println("\nDone gathering")
				doneGathering = true
			}
		}
	}

	// 3. Read up to date blockchain_headers file - this can be improved since
	//    we already read most of it but for now: simplicity
	b2, err := h.ReadAllBytesFromFile()
	if err != nil {
		return err
	}

	// 4. Store all headers in a map
	err = h.Store(b2, 0)
	if err != nil {
		return err
	}
	h.hdrsTip = maybeTip

	// 5. Verify headers in headers map
	fmt.Printf("starting verify at height %d\n", h.hdrsTip)
	err = h.VerifyAll()
	if err != nil {
		return err
	}
	fmt.Println("header chain verified")

	h.synced = true
	fmt.Println("headers synced up to tip ", h.hdrsTip)
	return nil
}

// SubscribeClientHeaders subscribes to new block tip notifications from the
// electrumx server and handles them as they arrive. The client local 'blockhain
// _headers file is appended and the headers map updated and verified.
//
// Note:
// should a new block arrive quickly, perhaps while the server is still processing
// prior blocks, the server may only notify of the most recent chain tip. The
// protocol does not guarantee notification of all intermediate block headers.
//
// SubscribeClientHeaders is part of the ElectrumClient interface implementation
func (ec *BtcElectrumClient) SubscribeClientHeaders() error {
	h := ec.clientHeaders

	// local tip for calculation before storage
	maybeTip := h.hdrsTip

	node := ec.GetNode()

	hdrResNotifyCh, err := node.GetHeadersNotify()
	if err != nil {
		return err
	}

	hdrRes, err := node.SubscribeHeaders()
	if err != nil {
		return err
	}

	fmt.Println("Subscribe Headers")
	fmt.Println("hdrRes.Height", hdrRes.Height, "maybeTip", maybeTip, "diff", hdrRes.Height-maybeTip)
	fmt.Println("hdrRes.Hex", hdrRes.Hex)

	svrCtx := node.GetServerConn().SvrCtx

	go func() {
		fmt.Println("=== Waiting for headers ===")
		for {
			select {

			case <-svrCtx.Done():
				fmt.Println("Server shutdown - subscribe headers notify")
				node.Stop()
				return

			case <-hdrResNotifyCh:
				// read whatever is in the queue, usually one header at tip
				for x := range hdrResNotifyCh {
					fmt.Println("New Block: ", x.Height, x.Hex)
					if x.Height > maybeTip {
						n := x.Height - maybeTip
						if n == 1 {
							// simple case: just store it
							fmt.Println("Storing header for height: ", x.Height)
							b, err := hex.DecodeString(x.Hex)
							if err != nil {
								panic(err)
							}
							hdrsAppended, err := h.AppendHeaders(b)
							if err != nil {
								panic(err)
							}
							if hdrsAppended != 1 {
								panic("appended less headers than read")
							}
							err = h.Store(b, x.Height)
							if err != nil {
								panic("could not store header in map")
							}

							// update tip / local tip
							h.hdrsTip = x.Height
							maybeTip = x.Height

							// verify added header back from new tip
							h.VerifyFromTip(2, false)

						} else {
							// Server can skip any amount of headers but we should
							// trust that this SingleNode's tip is the tip.
							fmt.Println("More than one header..")
							numMissing := uint32(x.Height - maybeTip)
							from := uint32(maybeTip + 1)
							numToGet := numMissing
							fmt.Printf("Filling from height %d to height %d inclusive\n", from, x.Height)
							// go get them with 'block.headers'
							hdrsRes, err := node.BlockHeaders(from, numToGet)
							if err != nil {
								panic(err)
							}
							count := hdrsRes.Count

							fmt.Println("Storing: ", count, " headers ", from, "..", from+count-1)

							if count > 0 {
								b, err := hex.DecodeString(hdrsRes.HexConcat)
								if err != nil {
									panic(err)
								}
								hdrsAppended, err := h.AppendHeaders(b)
								if err != nil {
									panic(err)
								}
								if hdrsAppended != int32(count) {
									panic("only appended less headers than read")
								}
								err = h.Store(b, int32(from))
								if err != nil {
									panic("could not store headers in map")
								}

								// update tip / local tip
								h.hdrsTip = x.Height
								maybeTip = x.Height

								// verify added headers back from new tip
								h.VerifyFromTip(int32(count+1), false)
							}
						}
					} else {
						fmt.Printf("Already got a header for height %d - possible reorg (unhandled)\n", x.Height)
					}
				}
			}
		}
		// serve until ^C
	}()

	return nil
}
