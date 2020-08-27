package proc

import (
	"encoding/json"
	"fmt"

	log "github.com/go-pkgz/lgr"
	bolt "go.etcd.io/bbolt"

	"github.com/lafin/estonia-news/model"
)

// SaveEntry - add new or update an existing record
func (b Processor) SaveEntry(item model.Entry) (*model.Entry, error) {
	name := "entry"
	item.Date = item.GetDate()
	key, err := item.GetKey([]byte(item.GUID))
	if err != nil {
		return nil, err
	}
	err = b.Store.Update(func(tx *bolt.Tx) error {
		var bucket *bolt.Bucket
		bucket, err = tx.CreateBucketIfNotExists([]byte(name))
		if err != nil {
			return err
		}
		var data []byte
		data, err = json.Marshal(&item)
		if err != nil {
			return err
		}
		log.Printf("[INFO] save %s - %s - %s", string(key), name, item.GUID)
		err = bucket.Put(key, data)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// LoadEntry - get all entries
func (b Processor) LoadEntry() (*[]model.Entry, error) {
	name := "entry"
	result := []model.Entry{}
	err := b.Store.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(name))
		if bucket == nil {
			return fmt.Errorf("no bucket for %s", name)
		}
		c := bucket.Cursor()
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			item := model.Entry{}
			if err := json.Unmarshal(v, &item); err != nil {
				log.Printf("[WARN] failed to unmarshal, %v", err)
				continue
			}
			result = append(result, item)
		}
		return nil
	})
	return &result, err
}

// DeleteEntry - delete a record and associated access
func (b Processor) DeleteEntry(guid string) error {
	err := b.Store.Update(func(tx *bolt.Tx) error {
		name := "entry"
		bucket := tx.Bucket([]byte(name))
		if bucket == nil {
			return fmt.Errorf("no bucket for %s", name)
		}

		c := bucket.Cursor()
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			item := model.Entry{}
			if err := json.Unmarshal(v, &item); err != nil {
				log.Printf("[WARN] failed to unmarshal, %v", err)
				continue
			}
			if guid == item.GUID {
				if err := bucket.Delete(k); err != nil {
					log.Printf("[WARN] failed to remove entry, %v", err)
					continue
				}
			}
		}
		return nil
	})
	return err
}
