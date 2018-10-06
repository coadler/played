package played

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	_ "expvar"
	"fmt"
	"io"
	"log"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/codercom/retry"
	"github.com/dustin/go-humanize"
	"go.uber.org/zap"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/apple/foundationdb/bindings/go/src/fdb/directory"
	"github.com/apple/foundationdb/bindings/go/src/fdb/tuple"
	"github.com/apple/foundationdb/bindings/go/src/fdb/subspace"
	"github.com/coadler/played/pb"
	"github.com/dgraph-io/badger"
	"github.com/go-redis/redis"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PlayedServer struct {
	log *zap.Logger

	DB    fdb.Database
	Redis *redis.Client

	FirstSeen   subspace.Subspace
	LastUpdated subspace.Subspace
	Current     subspace.Subspace
	Played      subspace.Subspace
}

func Start() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Println(err.Error())
	}

	fdb.MustAPIVersion(510)
	db := fdb.MustOpenDefault()

	playedDir, err := directory.CreateOrOpen(db, []string{"played"}, nil)
	if err != nil {
		logger.Error("failed to create fdb directory", zap.Error(err))
		return
	}

	current := playedDir.Sub("current")
	lastUpdated := playedDir.Sub("last-updated")
	firstSeen := playedDir.Sub("first-seen")
	playedSub := playedDir.Sub("played")

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

func (s *PlayedServer) processPlayed(user, game string) error {
	if user == "" {
		s.log.Error("processPlayed called with empty user", zap.String("user", user))
		return errors.New("can't process empty user or game")
	}

	var (
		err error
	)

	w, err := s.Redis.Exists(fmtWhitelistKey(user)).Result()
	if err != nil {
		return err
	}

	_ = strings.Compare
	_ = w
	// if w != 1 && !strings.HasPrefix(user, "10") {
	// 	return nil
	// }

	var (
		timeNow = time.Now()
		// 64 bit
		now [8]byte
	)
	binary.LittleEndian.PutUint64(now[:], uint64(timeNow.Unix()))

	var (
		fsKey   = s.FirstSeen.Pack(tuple.Tuple{user})
		curKey  = s.Current.Pack(tuple.Tuple{user})
		lastKey = s.LastUpdated.Pack(tuple.Tuple{user})
	)
	s.DB.Transact(func(t fdb.Transaction) (ret interface{}, err error) {
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

		fmt.Println(user, game)
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

func (s *PlayedServer) SendPlayed(stream pb.Played_SendPlayedServer) error {
	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return stream.SendAndClose(&pb.SendPlayedResponse{})
			}

			s.log.Error("error receiving", zap.Error(err))
			return stream.SendAndClose(&pb.SendPlayedResponse{})
		}

		if msg == nil {
			s.log.Error("msg is nil (!)")
			return status.Errorf(codes.Internal, "msg became nil")
		}

		user, game := msg.User, msg.Game
		go func() {
			err = retry.
				New(5 * time.Millisecond).
				Timeout(2 * time.Second).
				Backoff(200 * time.Millisecond).
				Condition(func(err error) bool {
					return err == badger.ErrConflict
				}).
				Run(func() error {
					return s.processPlayed(user, game)
				})
			if err != nil {
				s.log.Error("failed to process played message", zap.Error(err))
			}
		}()

	}
}

func (s *PlayedServer) GetPlayed(c context.Context, req *pb.GetPlayedRequest) (*pb.GetPlayedResponse, error) {
	resp := new(pb.GetPlayedResponse)
	resp.Games = []*pb.GameEntryPublic{}
	gms := Games{}

	ranger := s.Played.Pack(tuple.Tuple{req.User})
	s.DB.ReadTransact(func(t fdb.ReadTransaction) (ret interface{}, err error) {
		pre, _ := fdb.PrefixRange(ranger.FDBKey())
		r := t.GetRange(pre, fdb.RangeOptions{
			Mode: fdb.StreamingModeWantAll,
		}).Iterator()
		for r.Advance() {
			k := r.MustGet()

			parts, err := s.Played.Unpack(k.Key)
			if err != nil {
				return nil, err
			}

			gms = append(gms, &pb.GameEntry{
				Name: parts[1].(string),
				Dur: int32(binary.LittleEndian.Uint32(k.Value)),
			})
		}

		first := t.Get(s.FirstSeen.Pack(tuple.Tuple{req.User})).MustGet()
		resp.First = humanize.Time(time.Unix(int64(binary.LittleEndian.Uint64(first)), 0))

		last := t.Get(s.LastUpdated.Pack(tuple.Tuple{req.User})).MustGet()
		resp.Last = humanize.Time(time.Unix(int64(binary.LittleEndian.Uint64(last)), 0))

		return
	})

	sort.Sort(gms)
	for _, e := range gms {
		resp.Games = append(resp.Games, &pb.GameEntryPublic{
			Name: e.Name,
			Dur:  (time.Duration(e.Dur) * time.Second).String(),
		})
	}

	return resp, nil
}

func (s *PlayedServer) CheckHealth(c context.Context, req *pb.CheckHealthRequest) (*pb.CheckHealthResponse, error) {
	s.log.Debug("got health check")
	return &pb.CheckHealthResponse{}, nil
}

type Games []*pb.GameEntry

func (g Games) Len() int {
	return len(g)
}

func (g Games) Swap(i, j int) {
	g[i], g[j] = g[j], g[i]
}

func (g Games) Less(i, j int) bool {
	return g[i].Dur >= g[j].Dur
}
