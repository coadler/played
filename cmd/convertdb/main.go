package main

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/apple/foundationdb/bindings/go/src/fdb/directory"
	"github.com/apple/foundationdb/bindings/go/src/fdb/tuple"
	"github.com/boltdb/bolt"
	"github.com/dgraph-io/badger"
	"github.com/go-redis/redis"
	"go.uber.org/zap"
)

func main() {
	opts := badger.DefaultOptions
	opts.Dir = "../playedd/badger/"
	opts.ValueDir = "../playedd/badger/"
	opts.SyncWrites = false
	db, err := badger.Open(opts)
	if err != nil {
		log.Println(err)
		return
	}
	defer db.Close()

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Println(err.Error())
	}

	bdb, err := bolt.Open("../playedd/bolt/whitelist.db", 0600, nil)
	if err != nil {
		logger.Error("failed to open bolt/whitelist.db", zap.Error(err))
		return
	}
	defer bdb.Close()

	rc := redis.NewClient(
		&redis.Options{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		},
	)

	key := []byte("whitelist")
	err = bdb.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(key)
		if err != nil {
			return err
		}

		c := bucket.Cursor()

		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			err := rc.Set(string(k), "", 0).Err()
			if err != nil {
				fmt.Println("redis err:", err)
			}
		}

		return err
	})
	if err != nil {
		logger.Error("bolt error", zap.Error(err))
		return
	}

	fdb.MustAPIVersion(510)
	newdb := fdb.MustOpenDefault()

	playedDir, err := directory.CreateOrOpen(newdb, []string{"played"}, nil)
	if err != nil {
		logger.Error("failed to create fdb directory", zap.Error(err))
		return
	}

	current := playedDir.Sub("current")
	lastUpdated := playedDir.Sub("last-updated")
	firstSeen := playedDir.Sub("first-seen")
	playedSub := playedDir.Sub("played")

	err = db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			go func() {
				_, err = newdb.Transact(func(t fdb.Transaction) (ret interface{}, err error) {
					i := it.Item()

					key := i.Key()

					val, err := i.Value()
					if err != nil {
						return nil, err
					}

					kParts := bytes.Split(key, []byte(":"))
					if len(kParts) == 3 {
						user := kParts[1]
						kind := kParts[2]

						if string(kind) == "firstseen" {
							t.Set(firstSeen.Pack(tuple.Tuple{string(user)}), val)
						}
						if string(kind) == "lastchanged" {
							t.Set(lastUpdated.Pack(tuple.Tuple{string(user)}), val)
						}
						if string(kind) == "current" {
							t.Set(current.Pack(tuple.Tuple{string(user)}), val)
						}
					}

					if len(kParts) > 3 {
						user := kParts[1]
						game := bytes.Join(kParts[3:], []byte(":"))

						t.Set(playedSub.Pack(tuple.Tuple{string(user), string(game)}), val)
					}

					return
				})
				if err != nil {
					fmt.Println(err)
				}
			}()

			time.Sleep(time.Millisecond)
		}

		return nil
	})

	if err != nil {
		fmt.Println(err)
	}
}
