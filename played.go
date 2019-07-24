package played

import (
	"bytes"
	"encoding/binary"
	"errors"
	_ "expvar"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/apple/foundationdb/bindings/go/src/fdb/directory"
	"github.com/apple/foundationdb/bindings/go/src/fdb/subspace"
	"github.com/apple/foundationdb/bindings/go/src/fdb/tuple"
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

func (s *Server) processPlayed(user, game string) error {
	defer func() {
		err := recover()
		if err != nil {
			fmt.Println(err)
		}
	}()

	if user == "" {
		s.log.Error("processPlayed called with empty user", zap.String("user", user))
		return errors.New("can't process empty user or game")
	}

	var (
		err error
	)

	w, err := s.rdb.Exists(fmtWhitelistKey(user)).Result()
	if err != nil {
		return err
	}

	if w != 1 {
		return nil
	}

	var (
		fsKey   = s.subs.FirstSeen.Pack(tuple.Tuple{user})
		curKey  = s.subs.Current.Pack(tuple.Tuple{user})
		lastKey = s.subs.LastUpdated.Pack(tuple.Tuple{user})
	)

	s.DB.Transact(func(t fdb.Transaction) (ret interface{}, err error) {
		// because of the low cost of time.Now and PutUint64 i'd rather
		// prefer idempotence because this will be retried if there is a conflict
		//
		// also little endian is used because foundationdb's atomic add function
		// requires little endian encoded uints
		var (
			timeNow = time.Now()
			// 64 bit
			now [8]byte
		)
		binary.LittleEndian.PutUint64(now[:], uint64(timeNow.Unix()))

		first := t.Get(fsKey).MustGet()
		if first == nil {
			t.Set(fsKey, now[:])
		}

		curVal := t.Get(curKey).MustGet()
		if curVal == nil {
			t.Set(curKey, []byte(game))
			t.Set(lastKey, now[:])
			return
		}

		if bytes.Equal(curVal, []byte(game)) {
			return
		}

		// we know now that the user changed games

		lastChanged := timeNow
		lastChangedRaw := t.Get(lastKey).MustGet()
		if lastChangedRaw != nil && len(lastChangedRaw) == 8 {
			lastChanged = time.Unix(int64(binary.LittleEndian.Uint64(lastChangedRaw)), 0)
		}

		t.Set(lastKey, now[:])
		t.Set(curKey, []byte(game))

		// if they just changed from not playing a game, theres no need to compute time played.
		if bytes.Equal(curVal, []byte{}) {
			return
		}

		curGameKey := s.Played.Pack(tuple.Tuple{user, string(curVal)})
		toAdd := [8]byte{}
		binary.LittleEndian.PutUint32(toAdd[:], uint32(timeNow.Sub(lastChanged).Seconds()))
		t.Add(curGameKey, toAdd[:])
		return
	})

	return err
}
