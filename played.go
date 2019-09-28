package played

import (
	"bytes"
	"net"
	"net/http"
	"strconv"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/apple/foundationdb/bindings/go/src/fdb/directory"
	"github.com/apple/foundationdb/bindings/go/src/fdb/subspace"
	"github.com/coadler/played/pb"
	"github.com/go-redis/redis"
	"github.com/tatsuworks/gateway/discordetf"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"nhooyr.io/websocket"
)

type Server struct {
	log *zap.Logger

	db  fdb.Database
	rdb *redis.Client

	grpcAddr string
	wsAddr   string

	subs Subspaces
}

type Subspaces struct {
	FirstSeen   subspace.Subspace
	LastUpdated subspace.Subspace
	Current     subspace.Subspace
	Played      subspace.Subspace
}

func NewServer(logger *zap.Logger, db fdb.Database, rdb *redis.Client, grpcAddr, wsAddr string) (*Server, error) {
	dir, err := directory.CreateOrOpen(db, []string{"played"}, nil)
	if err != nil {
		logger.Fatal("failed to create fdb directory", zap.Error(err))
	}

	return &Server{
		log:      logger,
		db:       db,
		rdb:      rdb,
		grpcAddr: grpcAddr,
		wsAddr:   wsAddr,
		subs: Subspaces{
			FirstSeen:   dir.Sub("first-seen"),
			LastUpdated: dir.Sub("last_updated"),
			Current:     dir.Sub("current"),
			Played:      dir.Sub("played"),
		},
	}, nil
}

func (s *Server) Start() {
	go func() {
		lis, err := net.Listen("tcp", s.grpcAddr)
		if err != nil {
			s.log.Error("failed to listen", zap.Error(err))
			return
		}

		srv := grpc.NewServer()
		pb.RegisterPlayedServer(srv, s)

		s.log.Info("grpc listening", zap.String("addr", s.grpcAddr))
		srv.Serve(lis)
	}()

	go func() {
		s.log.Info("ws listening", zap.String("addr", s.wsAddr))
		http.ListenAndServe(s.wsAddr, nil)
	}()

	<-make(chan struct{})
}

func (s *Server) listenWS(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		s.log.Error("failed to accept websocket", zap.Error(err))
		return
	}

	buf := new(bytes.Buffer)
	for {
		_, r, err := c.Reader(r.Context())
		if err != nil {
			s.log.Error("failed to get reader", zap.Error(err))
			c.Close(websocket.StatusAbnormalClosure, "z")
			return
		}

		_, err = buf.ReadFrom(r)
		if err != nil {
			s.log.Error("failed to read message", zap.Error(err))
			c.Close(websocket.StatusAbnormalClosure, "z")
			return
		}

		pres, err := discordetf.DecodePlayedPresence(buf.Bytes())
		if err != nil {
			s.log.Error("failed to decode presence", zap.Error(err))
			continue
		}

		go func() {
			s.log.Info("presence", zap.Int64("user", pres.UserID), zap.String("game", pres.Game))
			err = s.processPlayed(strconv.FormatInt(pres.UserID, 10), pres.Game)
			if err != nil {
				s.log.Error("failed to process presence", zap.Error(err))
			}
		}()
	}
}
