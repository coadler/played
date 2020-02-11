package played

import (
	"bytes"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/apple/foundationdb/bindings/go/src/fdb/directory"
	"github.com/apple/foundationdb/bindings/go/src/fdb/subspace"
	"github.com/coadler/played/pb"
	"github.com/go-redis/redis"
	"github.com/paulbellamy/ratecounter"
	"github.com/tatsuworks/gateway/discord"
	"github.com/tatsuworks/gateway/discord/discordetf"
	"go.uber.org/zap"
	"golang.org/x/xerrors"
	"google.golang.org/grpc"
	"nhooyr.io/websocket"
)

type Server struct {
	log *zap.Logger

	db  fdb.Database
	rdb *redis.Client

	grpcAddr string
	wsAddr   string
	enc      discord.Encoding

	subs Subspaces
	rate *ratecounter.RateCounter

	whitelists  map[int64]struct{}
	whitelistMu sync.RWMutex
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
		log:      logger.Named("played"),
		db:       db,
		rdb:      rdb,
		grpcAddr: grpcAddr,
		wsAddr:   wsAddr,
		enc:      discordetf.Encoding,
		subs: Subspaces{
			FirstSeen:   dir.Sub("first-seen"),
			LastUpdated: dir.Sub("last_updated"),
			Current:     dir.Sub("current"),
			Played:      dir.Sub("played"),
		},
		rate: ratecounter.NewRateCounter(60 * time.Second),
	}, nil
}

func (s *Server) logRoutine() {
	for {
		time.Sleep(time.Minute)
		s.log.Info("event report", zap.Float32("rate", float32(s.rate.Rate()/60)))
	}
}

func (s *Server) updateWhitelistRoutine() {
	for {
		time.Sleep(time.Minute)
		newlen, err := s.updateWhitelistCache()
		if err != nil {
			s.log.Error("failed to update whitelist cache", zap.Error(err))
		}
		s.log.Info("updated whitelist cache", zap.Int("len", newlen))
	}
}

func (s *Server) updateWhitelistCache() (int, error) {
	keys, err := s.rdb.Keys(whitelistPrefix).Result()
	if err != nil {
		return 0, xerrors.Errorf("list whitelist keys: %w", err)
	}

	newMap := make(map[int64]struct{}, len(keys))
	for _, key := range keys {
		sp := strings.Split(key, ":")
		if len(sp) != 3 {
			s.log.Error("found malformed key, expected 3 parts", zap.Int("parts", len(sp)), zap.String("key", key))
			continue
		}

		id, err := strconv.ParseInt(sp[2], 10, 64)
		if err != nil {
			return 0, xerrors.Errorf("parse user id: %w", err)
		}

		newMap[id] = struct{}{}
	}

	newLen := len(newMap)
	s.whitelistMu.Lock()
	s.whitelists = newMap
	s.whitelistMu.Unlock()
	return newLen, nil
}

func (s *Server) Start() {
	whitelistLen, err := s.updateWhitelistCache()
	if err != nil {
		s.log.Fatal("failed to update whitelist cache", zap.Error(err))
	}
	s.log.Info("initialized whitelist cache", zap.Int("len", whitelistLen))

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
		http.ListenAndServe(s.wsAddr, &wsserver{s})
	}()

	go s.logRoutine()

	<-make(chan struct{})
}

var _ http.Handler = &wsserver{}

type wsserver struct {
	*Server
}

func (s *wsserver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/healthz" {
		w.WriteHeader(200)
		w.Write([]byte("OK!"))
		return
	}

	s.log.Info("accepting websocket")

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

		pres, err := s.enc.DecodePlayedPresence(buf.Bytes())
		if err != nil {
			s.log.Error("failed to decode presence", zap.Error(err))
			continue
		}
		buf.Reset()
		s.rate.Incr(1)

		go func() {
			err = s.processPlayed(pres.UserID, pres.Game)
			if err != nil {
				s.log.Error("failed to process presence", zap.Error(err))
			}
		}()
	}
}
