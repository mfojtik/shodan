package config

import (
	"fmt"
	"strings"

	"github.com/boltdb/bolt"
	"github.com/spf13/pflag"
)

type Storage interface {
	Get(string) ([]byte, error)
	List(string) ([]string, error)
	Set(string, []byte) error
}

var StorageNotFoundErr = fmt.Errorf("key not found")

func AddBoltFlags(fs *pflag.FlagSet) {
	fs.StringVar(&globalConfig.boltPath, "boltdb-path", "", "Path to where boltdb store data")
}

func (o *CommonOptions) initializeBoltDB(path string) error {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return fmt.Errorf("unable to open boltdb at %q: %v", o.boltPath, err)
	}
	o.Storage = &boltStorage{
		db: db,
	}
	return nil
}

type boltStorage struct {
	db *bolt.DB
}

func (s *boltStorage) Get(name string) ([]byte, error) {
	var res []byte
	var resErr error
	if viewErr := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(name))
		if b == nil {
			res, resErr = nil, nil
			return nil
		}
		v := b.Get([]byte("data"))
		res, resErr = make([]byte, len(v)), nil
		copy(res, v)
		return nil
	}); viewErr != nil {
		return nil, viewErr
	}
	if len(res) == 0 {
		return nil, StorageNotFoundErr
	}
	return res, resErr
}

func (s *boltStorage) List(namePrefix string) ([]string, error) {
	result := []string{}
	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			if len(namePrefix) > 0 && strings.HasPrefix(string(name), namePrefix) {
				return nil
			}
			result = append(result, string(name))
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *boltStorage) Set(name string, data []byte) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(name))
		if b == nil {
			b, _ = tx.CreateBucket([]byte(name))
		}
		return b.Put([]byte("data"), data)
	})
}
