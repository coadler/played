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
	"time"

	"github.com/codercom/retry"
	"github.com/dustin/go-humanize"
	"go.uber.org/zap"

	"github.com/ThyLeader/played/pb"
	"github.com/boltdb/bolt"
	"github.com/dgraph-io/badger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PlayedServer struct {
	log *zap.Logger

	DB   *badger.DB
	Bolt *bolt.DB

	WhitelistBucket []byte
}

func Start() {
	opts := badger.DefaultOptions
	opts.Dir = "badger/"
	opts.ValueDir = "badger/"
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

	bdb, err := bolt.Open("bolt/whitelist.db", 0600, nil)
	if err != nil {
		logger.Error("failed to open bolt/whitelist.db", zap.Error(err))
		return
	}
	defer db.Close()

	key := []byte("whitelist")
	err = bdb.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(key)
		return err
	})
	if err != nil {
		logger.Error("failed to create bolt bucket", zap.Error(err))
		return
	}

	lis, err := net.Listen("tcp", "localhost:8089")
	if err != nil {
		logger.Error("failed to listen", zap.Error(err))
		return
	}

	// go http.ListenAndServe(":8089", nil)

	srv := grpc.NewServer()
	played := &PlayedServer{logger, db, bdb, key}
	pb.RegisterPlayedServer(srv, played)
	logger.Info("Listening on port :8089")
	srv.Serve(lis)
}

func (s *PlayedServer) processPlayed(user, game string) error {
	if user == "" {
		s.log.Error("processPlayed called with empty user", zap.String("user", user))
		return errors.New("can't process empty user or game")
	}

	var (
		err error
		end bool
	)
	// bolt is much better for heavy random reads so i
	// store the whitelist bucket in a separate bolt db.
	// also helps lock contentions
	err = s.Bolt.View(func(tx *bolt.Tx) error {
		// user isnt whitelisted, get out
		end = tx.Bucket(s.WhitelistBucket).Get([]byte(user)) == nil
		return nil
	})
	if err != nil {
		return err
	}

	if end {
		return nil
	}

	// s.log.Info("past whitelist", zap.String("user", user), zap.String("game", game))

	err = s.DB.View(func(tx *badger.Txn) error {
		current, err := tx.Get(UserCurrentKey(user))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				s.log.Info("current game not found", zap.String("game", game))
				return nil
			}

			return err
		}

		v, err := current.Value()
		if err != nil {
			return err
		}

		// we can get up to 100 repeat presences,
		// end early if we're getting an update for the current game.
		// we also do this within a read transaction so we don't
		// lock up the db
		end = bytes.Equal(v, []byte(user))
		return nil
	})
	if err != nil {
		return err
	}

	if end {
		return nil
	}

	s.log.Info("past same game", zap.String("user", user), zap.String("game", game))

	err = s.DB.Update(func(tx *badger.Txn) error {
		var (
			timeNow = time.Now()
			// 64 bit
			now [8]byte
			err error
		)

		binary.BigEndian.PutUint64(now[:], uint64(timeNow.Unix()))

		if _, err := tx.Get(UserFirstSeenKey(user)); err == badger.ErrKeyNotFound {
			err = tx.Set(UserFirstSeenKey(user), now[:])
			if err != nil {
				return err
			}
		}

		lastChanged := timeNow
		if item, err := tx.Get(UserLastChangedKey(user)); err != nil {
			if err != badger.ErrKeyNotFound {
				return err
			}

			err = tx.Set(UserLastChangedKey(user), now[:])
			if err != nil {
				return err
			}
		} else {
			raw, err := item.Value()
			if err != nil {
				return err
			}

			lastChanged = time.Unix(int64(binary.BigEndian.Uint64(raw)), 0)
		}

		item, err := tx.Get(UserCurrentKey(user))
		if err != nil {
			if err != badger.ErrKeyNotFound {
				return err
			}

			err = tx.Set(UserCurrentKey(user), []byte(user))
			if err != nil {
				return err
			}

			err = tx.Set(UserLastChangedKey(user), now[:])
			if err != nil {
				return err
			}

			return nil
		}
		rawCurrent, err := item.Value()
		if err != nil {
			return err
		}

		err = tx.Set(UserLastChangedKey(user), now[:])
		if err != nil {
			return err
		}

		err = tx.Set(UserCurrentKey(user), []byte(user))
		if err != nil {
			return err
		}

		game := string(rawCurrent)
		if game != "" {
			item, err := tx.Get(UserEntryKey(user, game))
			if err != nil {
				if err != badger.ErrKeyNotFound {
					return err
				}

				raw, err := (&pb.GameEntry{
					Name: game,
					Dur:  int32(timeNow.Sub(lastChanged).Seconds()),
				}).Marshal()
				if err != nil {
					return err
				}

				err = tx.Set(UserEntryKey(user, game), raw)
				if err != nil {
					return err
				}

				return nil
			}

			entry := new(pb.GameEntry)
			rawEntry, err := item.Value()
			if err != nil {
				return err
			}
			err = entry.Unmarshal(rawEntry)
			if err != nil {
				return err
			}

			entry.Dur += int32(timeNow.Sub(lastChanged).Seconds())

			rawEntry, err = entry.Marshal()
			if err != nil {
				return err
			}

			err = tx.Set(UserEntryKey(user, game), rawEntry)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *PlayedServer) SendPlayed(stream pb.Played_SendPlayedServer) error {
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
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
					err := s.processPlayed(user, game)
					if err != nil {
						s.log.Error("failed to process played message", zap.Error(err))
					}
					return err
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

	err := s.DB.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		prefix := []byte(fmt.Sprintf("played:%s:games:", req.User))
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			v, err := it.Item().Value()
			if err != nil {
				return err
			}

			entry := new(pb.GameEntry)
			err = entry.Unmarshal(v)
			if err != nil {
				return err
			}

			gms = append(gms, entry)
		}
		{
			i, err := txn.Get(UserFirstSeenKey(req.User))
			if err != nil {
				return err
			}

			first, err := i.Value()
			if err != nil {
				return err
			}

			resp.First = humanize.Time(time.Unix(int64(binary.BigEndian.Uint64(first)), 0))
		}
		{
			i, err := txn.Get(UserLastChangedKey(req.User))
			if err != nil {
				return err
			}

			last, err := i.Value()
			if err != nil {
				return err
			}

			resp.Last = humanize.Time(time.Unix(int64(binary.BigEndian.Uint64(last)), 0))
		}

		return nil
	})
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return resp, nil
		}

		s.log.Error("failed to get played data", zap.Error(err))
		return &pb.GetPlayedResponse{}, grpc.Errorf(codes.Internal, err.Error())
	}

	sort.Sort(gms)
	for _, e := range gms {
		resp.Games = append(resp.Games, &pb.GameEntryPublic{
			Name: e.Name,
			Dur:  (time.Duration(e.Dur) * time.Second).String(),
		})
	}
	return resp, nil
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
