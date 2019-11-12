package main

import (
	"flag"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/coadler/played"
	"github.com/go-redis/redis"
	"go.uber.org/zap"
)

var (
	grpcAddr  string
	wsAddr    string
	redisAddr string
)

func init() {
	flag.StringVar(&grpcAddr, "grpcAddr", "0.0.0.0:8080", "")
	flag.StringVar(&wsAddr, "wsAddr", "0.0.0.0:80", "")
	flag.StringVar(&redisAddr, "redisAddr", "localhost:6379", "")
	flag.Parse()
}

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	fdb.MustAPIVersion(610)
	db := fdb.MustOpenDefault()

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	if _, err := rdb.Ping().Result(); err != nil {
		logger.Fatal("failed to connect to redis", zap.Error(err))
	}

	p, err := played.NewServer(logger, db, rdb, grpcAddr, wsAddr)
	if err != nil {
		logger.Fatal("failed to create played server", zap.Error(err))
	}

	logger.Info("starting", zap.String("grpc", grpcAddr), zap.String("ws", wsAddr))
	p.Start()
}
