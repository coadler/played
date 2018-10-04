package main

import (
	"fmt"
	"log"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/go-redis/redis"
)

func main() {
	fmt.Println("dean sheather")
	fdb.MustAPIVersion(510)
	db := fdb.MustOpenDefault()
	fmt.Println("dean sheather")

	// _, err := db.Transact(func(tr fdb.Transaction) (ret interface{}, e error) {
	// 	tr.Set(fdb.Key("hello"), []byte("world"))
	// 	return
	// })
	// if err != nil {
	// 	log.Fatalf("Unable to set FDB database value (%v)", err)
	// }
	fmt.Println("dean sheather")

	ret, err := db.Transact(func(tr fdb.Transaction) (ret interface{}, e error) {
		ret = tr.Get(fdb.Key("hello32")).MustGet()
		return
	})
	if err != nil {
		log.Fatalf("Unable to read FDB database value (%v)", err)
	}

	v := ret.([]byte)
	fmt.Printf("hello, %s\n", string(v))

	r := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	e, err := r.Get("hello").Result()
	if err != nil {
		fmt.Println("error", err)
		return
	}
	fmt.Println(e)
}
