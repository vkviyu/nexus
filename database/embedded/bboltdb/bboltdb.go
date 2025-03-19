package bboltdb

import (
	"errors"
	"os"

	"go.etcd.io/bbolt"
)

type DB struct {
	*bbolt.DB
}

var ErrEmptyBuckets = errors.New("bucket list connot be empty")

func Open(path string, mode os.FileMode, opts *bbolt.Options) (*DB, error) {
	db, err := bbolt.Open(path, mode, opts)
	if err != nil {
		return nil, err
	}
	return &DB{db}, nil
}

// getNestedBucket retrieves nested buckets within tx based on the provided list.
// It does not create any bucket; if any bucket in the hierarchy does not exist, nil is returned.
func getNestedBucket(tx *bbolt.Tx, buckets []string) (*bbolt.Bucket, error) {
	if len(buckets) == 0 {
		return nil, ErrEmptyBuckets
	}

	b := tx.Bucket([]byte(buckets[0]))
	if b == nil {
		return nil, nil
	}
	for _, bucket := range buckets[1:] {
		b = b.Bucket([]byte(bucket))
		if b == nil {
			return nil, nil
		}
	}
	return b, nil
}

// createNestedBucket creates or retrieves nested buckets within tx based on the provided list.
// It creates any bucket that does not exist.
func createNestedBucket(tx *bbolt.Tx, buckets []string) (*bbolt.Bucket, error) {
	if len(buckets) == 0 {
		return nil, ErrEmptyBuckets
	}

	var (
		b   *bbolt.Bucket
		err error
	)
	b, err = tx.CreateBucketIfNotExists([]byte(buckets[0]))
	if err != nil {
		return nil, err
	}
	for _, bucket := range buckets[1:] {
		b, err = b.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

func (db *DB) NestedUpdateTransaction(buckets []string, fn func(tx *bbolt.Tx, b *bbolt.Bucket) error) error {
	return db.Update(func(tx *bbolt.Tx) error {
		// 这里直接构造嵌套桶，失败的错误将由 getNestedBucket 返回
		b, err := createNestedBucket(tx, buckets)
		if err != nil {
			return err
		}
		return fn(tx, b)
	})
}

func (db *DB) NestedViewTransaction(buckets []string, fn func(tx *bbolt.Tx, b *bbolt.Bucket) error) error {
	return db.View(func(tx *bbolt.Tx) error {
		b, err := getNestedBucket(tx, buckets)
		if err != nil {
			return err
		}
		// 若未找到桶，则不执行回调
		if b == nil {
			return nil
		}
		return fn(tx, b)
	})
}
