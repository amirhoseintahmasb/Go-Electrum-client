package db

import (
	"database/sql"
	"path"
	"sync"

	"main/wallet"

	_ "github.com/mattn/go-sqlite3"
)

// This database is an SqLite3 implementation of Datastore.
// A different database could be plugged in .. bbolt maybe
type SQLiteDatastore struct {
	cfg            wallet.Cfg
	enc            wallet.Enc
	keys           wallet.Keys
	utxos          wallet.Utxos
	stxos          wallet.Stxos
	txns           wallet.Txns
	watchedScripts wallet.WatchedScripts
	db             *sql.DB
	lock           *sync.RWMutex
}

func Create(repoPath string) (*SQLiteDatastore, error) {
	dbPath := path.Join(repoPath, "wallet.db")
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	l := new(sync.RWMutex)
	sqliteDB := &SQLiteDatastore{
		cfg: &CfgDB{
			db:   conn,
			lock: l,
		},
		enc: &EncDB{
			db:   conn,
			lock: l,
		},
		keys: &KeysDB{
			db:   conn,
			lock: l,
		},
		utxos: &UtxoDB{
			db:   conn,
			lock: l,
		},
		stxos: &StxoDB{
			db:   conn,
			lock: l,
		},
		txns: &TxnsDB{
			db:   conn,
			lock: l,
		},
		watchedScripts: &WatchedScriptsDB{
			db:   conn,
			lock: l,
		},
		db:   conn,
		lock: l,
	}
	initDatabaseTables(conn)
	return sqliteDB, nil
}

func (db *SQLiteDatastore) Cfg() wallet.Cfg {
	return db.cfg
}
func (db *SQLiteDatastore) Enc() wallet.Enc {
	return db.enc
}
func (db *SQLiteDatastore) Keys() wallet.Keys {
	return db.keys
}
func (db *SQLiteDatastore) Utxos() wallet.Utxos {
	return db.utxos
}
func (db *SQLiteDatastore) Stxos() wallet.Stxos {
	return db.stxos
}
func (db *SQLiteDatastore) Txns() wallet.Txns {
	return db.txns
}
func (db *SQLiteDatastore) WatchedScripts() wallet.WatchedScripts {
	return db.watchedScripts
}

func initDatabaseTables(db *sql.DB) error {
	var sqlStmt string
	sqlStmt = sqlStmt + `
	create table if not exists keys (scriptAddress text primary key not null, purpose integer, keyIndex integer, used integer, key text);
	create table if not exists utxos (outpoint text primary key not null, value integer, height integer, scriptPubKey text, watchOnly integer);
	create table if not exists stxos (outpoint text primary key not null, value integer, height integer, scriptPubKey text, watchOnly integer, spendHeight integer, spendTxid text);
	create table if not exists txns (txid text primary key not null, value integer, height integer, timestamp integer, watchOnly integer, tx blob);
	create table if not exists watchedScripts (scriptPubKey text primary key not null);
	create table if not exists config(key text primary key not null, value blob);
	create table if not exists enc(key text primary key not null, value blob);
	`
	_, err := db.Exec(sqlStmt)
	if err != nil {
		return err
	}
	return nil
}
