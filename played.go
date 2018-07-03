package played

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/ThyLeader/played/pb"
	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc"
)

type PlayedServer struct {
	DB *badger.DB
}

func Start() {
	opts := badger.DefaultOptions
	opts.Dir = "."
	opts.ValueDir = "."
	opts.SyncWrites = false
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	srv := grpc.NewServer()
	played := &PlayedServer{db}
	pb.RegisterPlayedServer(srv, played)
	fmt.Println("Listening on port :8080")
	srv.Serve(lis)
}

func (s *PlayedServer) SendPlayed(stream pb.Played_SendPlayedServer) error {
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(nil)
		}

		if err != nil {
			fmt.Println(err)
			return err
		}

		err = s.DB.Update(func(tx *badger.Txn) error {
			timeNow := time.Now()
			var now [8]byte
			var err error
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

				fmt.Println("first")
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
			fmt.Println("current:", game)
			if game != "" {
				item, err := tx.Get(UserEntryKey(msg.User, game))
				if err != nil {
					if err != badger.ErrKeyNotFound {
						return err
					}

					raw, err := proto.Marshal(&pb.GameEntry{
						Name: game,
						Dur:  int64(timeNow.Sub(lastChanged).Seconds()),
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

				entry.Dur += int64(timeNow.Sub(lastChanged).Seconds())

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
			return nil
		}
	}

	return nil
}

func (s *PlayedServer) GetPlayed(c context.Context, req *pb.GetPlayedRequest) (*pb.GetPlayedResponse, error) {
	err := s.DB.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		prefix := []byte(fmt.Sprintf("played:%s:games:", req.User))
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			k := item.Key()
			v, err := item.Value()
			if err != nil {
				return err
			}

			entry := new(pb.GameEntry)
			err = proto.Unmarshal(v, entry)
			if err != nil {
				return err
			}
			fmt.Printf("key=%s, value=%+v\n", k, entry)
		}
		return nil
	})
	if err != nil {
		fmt.Println(err)
	}

	return &pb.GetPlayedResponse{}, nil
}
