// Package proc provided provided interfaces for working with models
package proc

import (
	"os"
	"path"
	"time"

	log "github.com/go-pkgz/lgr"
	bolt "go.etcd.io/bbolt"
)

// Processor is ...
type Processor struct {
	Store *bolt.DB
}

// NewBoltDB makes persistent boltdb based store
func NewBoltDB(dbFile string) (*Processor, error) {
	log.Printf("[INFO] bolt (persistent) store, %s", dbFile)
	if err := os.MkdirAll(path.Dir(dbFile), 0700); err != nil {
		return nil, err
	}
	result := Processor{}
	db, err := bolt.Open(dbFile, 0600, &bolt.Options{Timeout: 1 * time.Second}) //nolint
	if err != nil {
		return nil, err
	}
	result.Store = db

	err = db.Update(func(tx *bolt.Tx) error {
		name := "entry"
		_, err = tx.CreateBucketIfNotExists([]byte(name))
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &result, err
}
