package played

import (
	_ "expvar"
	"fmt"
	"log"
	"net"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/apple/foundationdb/bindings/go/src/fdb/directory"
	"github.com/apple/foundationdb/bindings/go/src/fdb/subspace"
	"github.com/coadler/played/pb"
	"github.com/go-redis/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Server struct {
	log *zap.Logger

	db  fdb.Database
	rdb *redis.Client

	subs Subspaces
}

type Subspaces struct {
	FirstSeen   subspace.Subspace
	LastUpdated subspace.Subspace
	Current     subspace.Subspace
	Played      subspace.Subspace
}

func NewServer(logger *zap.Logger, db fdb.Database, redis *redis.Client) (*Server, error) {
	dir, err := directory.CreateOrOpen(db, []string{"played"}, nil)
	if err != nil {
		logger.Fatal("failed to create fdb directory", zap.Error(err))
	}

	return &Server{
		log: logger,
		db:  db,
		rdb: redis,
		subs: Subspaces{
			FirstSeen:   dir.Sub("first-seen"),
			LastUpdated: dir.Sub("last_updated"),
			Current:     dir.Sub("current"),
			Played:      dir.Sub("played"),
		},
	}, nil
}

func Start() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("failed to create zap logger:", err)
	}

	fdb.MustAPIVersion(610)
	db := fdb.MustOpenDefault()

	rc := redis.NewClient(
		&redis.Options{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		},
	)

	lis, err := net.Listen("tcp", "0.0.0.0:8089")
	if err != nil {
		logger.Error("failed to listen", zap.Error(err))
		return
	}

	// go http.ListenAndServe(":8089", nil)

	srv := grpc.NewServer()
	played := &PlayedServer{logger, db, rc, firstSeen, lastUpdated, current, playedSub}
	pb.RegisterPlayedServer(srv, played)
	logger.Info("Listening on port :8089")
	srv.Serve(lis)
}

func fmtWhitelistKey(user string) string {
	return fmt.Sprintf("played:whitelist:%s", user)
}
