package main

import (
	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/coadler/played"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	fdb.MustAPIVersion(610)
	db := fdb.MustOpenDefault()

	p, err := played.NewServer(logger, db, nil)
	if err != nil {
		logger.Fatal("failed to create played server", zap.Error(err))
	}

	p.Start()
}
