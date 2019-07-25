package played

import (
	_ "expvar"
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

func NewServer(logger *zap.Logger, db fdb.Database, rdb *redis.Client) (*Server, error) {
	dir, err := directory.CreateOrOpen(db, []string{"played"}, nil)
	if err != nil {
		logger.Fatal("failed to create fdb directory", zap.Error(err))
	}

	return &Server{
		log: logger,
		db:  db,
		rdb: rdb,
		subs: Subspaces{
			FirstSeen:   dir.Sub("first-seen"),
			LastUpdated: dir.Sub("last_updated"),
			Current:     dir.Sub("current"),
			Played:      dir.Sub("played"),
		},
	}, nil
}

func (s *Server) Start() {
	lis, err := net.Listen("tcp", "0.0.0.0:8091")
	if err != nil {
		s.log.Error("failed to listen", zap.Error(err))
		return
	}

	s.startReadRoutines(5)

	srv := grpc.NewServer()
	pb.RegisterPlayedServer(srv, s)

	s.log.Info("Listening on port :8091")
	srv.Serve(lis)
}
