package wltbtc

import (
	"fmt"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
)

func createStorageManager() *StorageManager {
	return NewStorageManager(&mockStorage{}, &chaincfg.MainNetParams)
}

func TestStoreRetreiveBlob(t *testing.T) {
	sm := createStorageManager()
	var req = "ABC"
	err := sm.datastore.PutEncrypted([]byte(req), "abc")
	if err != nil {
		t.Fatal(err)
	}

	ret, err := sm.datastore.GetDecrypted("abc")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(string(req))
	fmt.Println(string(ret))
}

var pw = "abc" // tested
var xprv = "tprv8ZgxMBicQKsPfJU6JyiVdmFAtAzmWmTeEv85nTAHjLQyL35tdP2fAPWDSBBnFqGhhfTHVQMcnZhZDFkzFmCjm1bgf5UDwMAeFUWhJ9Dr8c4"
var xpub = "tpubD6NzVbkrYhZ4YmVtCdP63AuHTCWhg6eYpDis4yCb9cDNAXLfFmrFLt85cLFTwHiDJ9855NiE7cgQdiTGt5mb2RS9RfaxgVDkwBybJWm54Gh"
var seed = []byte{0x01, 0x02, 0x03}
var imported = []string{"wif_0", "wif_1", "wif_2"}

func TestStoreRetrieveEncryptedStore(t *testing.T) {
	sm := createStorageManager()

	sm.store = &Storage{
		Version:  "0.1",
		Xprv:     xprv,
		Xpub:     xpub,
		Seed:     seed,
		Imported: imported,
	}

	before := sm.store.String()
	fmt.Print("req: ", before)

	err := sm.Put(pw)
	if err != nil {
		t.Fatal(err)
	}

	err = sm.Get(pw)
	if err != nil {
		t.Fatal(err)
	}

	after := sm.store.String()
	fmt.Println("ret: ", after)

	if before != after {
		t.Fatal("Storage before != Storage after")
	}
}
