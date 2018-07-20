package played

import (
	"bytes"
	"context"
	"encoding/binary"
	_ "expvar"
	"fmt"
	"io"
	"log"
	"net"
	"sort"
	"time"

	"github.com/dustin/go-humanize"

	"github.com/ThyLeader/played/pb"
	"github.com/boltdb/bolt"
	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type PlayedServer struct {
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

	bdb, err := bolt.Open("bolt/whitelist.db", 0600, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer db.Close()

	key := []byte("whitelist")
	err = bdb.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(key)
		return err
	})
	if err != nil {
		log.Printf("failed to create bolt bucket: %v", err)
		return
	}

	lis, err := net.Listen("tcp", ":8089")
	if err != nil {
		log.Printf("failed to listen: %v", err)
		return
	}

	// go http.ListenAndServe(":8089", nil)

	srv := grpc.NewServer()
	played := &PlayedServer{db, bdb, key}
	pb.RegisterPlayedServer(srv, played)
	fmt.Println("Listening on port :8080")
	srv.Serve(lis)
}

func (s *PlayedServer) SendPlayed(stream pb.Played_SendPlayedServer) error {
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.SendPlayedResponse{})
		}

		if err != nil {
			fmt.Println(err)
			return err
		}

		var end bool
		// bolt is much better for heavy random reads so i
		// store the whitelist bucket in a separate bolt db.
		// also helps lock contentions
		err = s.Bolt.View(func(tx *bolt.Tx) error {
			// user isnt whitelisted, get out
			end = tx.Bucket(s.WhitelistBucket).Get([]byte(msg.User)) == nil
			return nil
		})
		if err != nil {
			return grpc.Errorf(codes.Internal, err.Error())
		}

		// allow approx 1% of requests through
		// u, _ := strconv.ParseInt(msg.User, 10, 64)

		end = end && !(msg.User[:2] == "24")

		if end {
			continue
		}

		fmt.Printf("got msg: %+v\n", *msg)

		err = s.DB.View(func(tx *badger.Txn) error {
			current, err := tx.Get(UserCurrentKey(msg.User))
			if err != nil {
				if err == badger.ErrKeyNotFound {
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
			end = bytes.Equal(v, []byte(msg.Game))
			return nil
		})

		if end {
			continue
		}

		err = s.DB.Update(func(tx *badger.Txn) error {
			var (
				timeNow = time.Now()
				// 64 bit
				now [8]byte
				err error
			)

			binary.BigEndian.PutUint64(now[:], uint64(timeNow.Unix()))

			if _, err := tx.Get(UserFirstSeenKey(msg.User)); err == badger.ErrKeyNotFound {
				err = tx.Set(UserFirstSeenKey(msg.User), now[:])
				if err != nil {
					return err
				}
			}

			lastChanged := timeNow
			if item, err := tx.Get(UserLastChangedKey(msg.User)); err != nil {
				if err != badger.ErrKeyNotFound {
					return err
				}

				err = tx.Set(UserLastChangedKey(msg.User), now[:])
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

			item, err := tx.Get(UserCurrentKey(msg.User))
			if err != nil {
				if err != badger.ErrKeyNotFound {
					return err
				}

				err = tx.Set(UserCurrentKey(msg.User), []byte(msg.Game))
				if err != nil {
					return err
				}

				err = tx.Set(UserLastChangedKey(msg.User), now[:])
				if err != nil {
					return err
				}

				return nil
			}
			rawCurrent, err := item.Value()
			if err != nil {
				return err
			}

			err = tx.Set(UserLastChangedKey(msg.User), now[:])
			if err != nil {
				return err
			}

			err = tx.Set(UserCurrentKey(msg.User), []byte(msg.Game))
			if err != nil {
				return err
			}

			game := string(rawCurrent)
			if game != "" {
				item, err := tx.Get(UserEntryKey(msg.User, game))
				if err != nil {
					if err != badger.ErrKeyNotFound {
						return err
					}

					raw, err := proto.Marshal(&pb.GameEntry{
						Name: game,
						Dur:  int32(timeNow.Sub(lastChanged).Seconds()),
					})
					if err != nil {
						return err
					}

					err = tx.Set(UserEntryKey(msg.User, game), raw)
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
				err = proto.Unmarshal(rawEntry, entry)
				if err != nil {
					return err
				}

				entry.Dur += int32(timeNow.Sub(lastChanged).Seconds())

				rawEntry, err = proto.Marshal(entry)
				if err != nil {
					return err
				}

				err = tx.Set(UserEntryKey(msg.User, game), rawEntry)
				if err != nil {
					return err
				}
			}

			return nil
		})
		if err != nil {
			fmt.Println(err)
			return grpc.Errorf(codes.Internal, err.Error())
		}
	}

	return nil
}

func (s *PlayedServer) GetPlayed(c context.Context, req *pb.GetPlayedRequest) (*pb.GetPlayedResponse, error) {
	resp := new(pb.GetPlayedResponse)
	resp.Games = []*pb.GameEntryPublic{}

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
			err = proto.Unmarshal(v, entry)
			if err != nil {
				return err
			}

			resp.Games = append(resp.Games, &pb.GameEntryPublic{
				Name: entry.Name,
				Dur:  (time.Duration(entry.Dur) * time.Second).String(),
			})
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

		fmt.Println(err)
		return &pb.GetPlayedResponse{}, grpc.Errorf(codes.Internal, err.Error())
	}

	sort.Sort(resp)
	return resp, nil
}
