package db

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"runtime"
	"sync"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/nacl/secretbox"
)

const STORAGE = "storage"

var (
	ErrBadPw = errors.New("bad password")
	// Argon2 params
	SALT    = []byte("2977958431d29f2d") // TODO: a good random for dev
	TIME    = uint32(1)
	MEM     = uint32(64 * 1024)
	THREADS = uint8(runtime.NumCPU())
	THRDMAX = uint8(255)
	KEYLEN  = uint32(32)
)

type EncDB struct {
	db   *sql.DB
	lock *sync.RWMutex
}

func (e *EncDB) PutEncrypted(b []byte, pw string) error {
	// encrypt
	eb, err := encryptBytes(b, pw)
	if err != nil {
		return err
	}
	// store in db
	e.lock.RLock()
	defer e.lock.RUnlock()
	tx, err := e.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("insert or replace into enc(key, value) values(?,?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(STORAGE, eb)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (e *EncDB) GetDecrypted(pw string) ([]byte, error) {
	// retreive from db , if exist
	e.lock.RLock()
	defer e.lock.RUnlock()
	stmt, err := e.db.Prepare("select value from enc where key=?")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	var b []byte
	err = stmt.QueryRow(STORAGE).Scan(&b)
	if err != nil {
		return nil, err
	}
	// decrypt
	return decryptBytes(b, pw)
}

func encryptBytes(unencrypted []byte, password string) ([]byte, error) {
	secretKey := getEncryptionKey32(password)
	var nonce [24]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return nil, err
	}
	encrypted := secretbox.Seal(nonce[:], unencrypted, &nonce, &secretKey)
	// nonce is the first 24 bytes of encrypted. the rest is the actual
	// encryption result. [nonce 24][ ...the encryption result...]
	return encrypted, nil
}

func decryptBytes(encrypted []byte, password string) ([]byte, error) {
	secretKey := getEncryptionKey32(password)
	var decryptNonce [24]byte
	copy(decryptNonce[:], encrypted[:24])
	decrypted, ok := secretbox.Open(nil, encrypted[24:], &decryptNonce, &secretKey)
	if !ok {
		return nil, errors.New("secretbox decryption error")
	}
	// decrypted is the decryption of the encrypted bytes with the pre-pended
	// plaintext nonce stripped out
	return decrypted, nil
}

func getEncryptionKey32(password string) [32]byte {
	threads := THREADS
	if threads > THRDMAX {
		threads = THRDMAX
	}
	b := argon2.IDKey([]byte(password), SALT, TIME, MEM, threads, KEYLEN)
	return ([32]byte)(b)
}

/////////////////////////////////
// Testing

func PrivKeyToWif() error {
	var key *btcec.PrivateKey
	key, err := btcec.NewPrivateKey()
	if err != nil {
		return err
	}
	fmt.Println(key.Serialize())
	wif, err := btcutil.NewWIF(key, &chaincfg.MainNetParams, false)
	if err != nil {
		return err
	}
	wifStr := wif.String()
	fmt.Println(wifStr)

	dec, _ := btcutil.DecodeWIF(wifStr)
	fmt.Println(dec.PrivKey.Serialize())

	// Can also do this
	ecKey := key.ToECDSA()
	fmt.Println(ecKey)
	return nil
}
