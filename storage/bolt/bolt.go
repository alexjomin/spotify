package bolt

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/alexjomin/spotify/storage"
	bolt "github.com/coreos/bbolt"
)

var ErrNotFound = errors.New("Entity not found")

type Bolt struct {
	db     *bolt.DB
	bucket []byte
}

func New(path, bucket string) (storage.Storage, error) {
	db, err := bolt.Open(path, 0600, nil)

	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucket))
		return err
	})

	if err != nil {
		return nil, err
	}

	return &Bolt{
		db:     db,
		bucket: []byte(bucket),
	}, nil
}

func (s *Bolt) Get(key string) ([]byte, error) {

	var v []byte

	_ = s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(s.bucket)

		if b == nil {
			fmt.Println("bucket is nil")
			return nil
		}
		v = b.Get([]byte(key))
		return nil
	})

	if v == nil {
		return nil, ErrNotFound
	}

	return v, nil

}

func (s *Bolt) Insert(key string, value interface{}) error {

	buf, err := json.Marshal(value)

	if err != nil {
		return err
	}

	_ = s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(s.bucket)
		err := b.Put([]byte(key), buf)
		return err
	})

	return nil
}

func (s *Bolt) Delete(key string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(s.bucket)
		return b.Delete([]byte(key))
	})
}
