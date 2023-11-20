package db

import (
	"database/sql"
	"sync"
	"time"
)

type CfgDB struct {
	db   *sql.DB
	lock *sync.RWMutex
}

func (s *CfgDB) GetCreationDate() (time.Time, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	var t time.Time
	stmt, err := s.db.Prepare("select value from config where key=?")
	if err != nil {
		return t, err
	}
	defer stmt.Close()
	var creationDate []byte
	err = stmt.QueryRow("creationDate").Scan(&creationDate)
	if err != nil {
		return t, err
	}
	return time.Parse(time.RFC3339, string(creationDate))
}

func (s *CfgDB) PutCreationDate(creationDate time.Time) error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("insert or replace into config(key, value) values(?,?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec("creationDate", creationDate.Format(time.RFC3339))
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}
