package wltbtc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"main/wallet"

	"github.com/btcsuite/btcd/chaincfg"
)

// Encrypted storage for btc. Stored as an encrypted blob in database.

type Storage struct {
	Version  string   `json:"version"`
	Xprv     string   `json:"xprv"`
	Xpub     string   `json:"xpub"`
	Seed     []byte   `json:"seed,omitempty"`
	Imported []string `json:"imported,omitempty"`
}

// String returns the string representation of the Storage but only of the
// fields that are always present
func (s *Storage) String() string {
	b := new(bytes.Buffer)
	fmt.Fprintf(b, "{\n%s\n%s\n%s\n}\n", s.Version, s.Xprv, s.Xpub)
	return b.String()
}

type StorageManager struct {
	datastore wallet.Enc
	params    *chaincfg.Params
	store     *Storage
}

func NewStorageManager(db wallet.Enc, params *chaincfg.Params) *StorageManager {
	sm := &StorageManager{
		datastore: db,
		params:    params,
		store: &Storage{
			Version: "0.1",
		},
	}
	return sm
}

func (sm *StorageManager) Put(pw string) error {
	if len(pw) == 0 {
		return errors.New("no password")
	}

	if sm.store == nil {
		return errors.New("nothing to store")
	}

	b, err := json.Marshal(sm.store)
	if err != nil {
		return err
	}

	return sm.datastore.PutEncrypted(b, pw)
}

func (sm *StorageManager) Get(pw string) error {
	if len(pw) == 0 {
		return errors.New("no password")
	}

	b, err := sm.datastore.GetDecrypted(pw)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, sm.store)
}
