package played

import (
	"context"
	"encoding/binary"
	"sort"
	"time"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/coadler/played/pb"
)

func (s *Server) GetPlayed(c context.Context, req *pb.GetPlayedRequest) (*pb.GetPlayedResponse, error) {
	var (
		resp = &pb.GetPlayedResponse{}
		gms  = Games{}
	)

	_, err := s.db.ReadTransact(func(t fdb.ReadTransaction) (interface{}, error) {
		t = t.Snapshot()

		r := t.GetRange(
			s.fmtPlayedUserRange(req.User),
			RangeWantAll,
		).Iterator()

		for r.Advance() {
			k := r.MustGet()

			_, game, err := s.unpackPlayed(k.Key)
			if err != nil {
				return nil, err
			}

			gms = append(gms, &pb.GameEntry{
				Name: game,
				Dur:  int32(binary.LittleEndian.Uint32(k.Value)),
			})
		}

		resp.First = s.humanTimeFromUnix(t.Get(s.fmtFirstSeenKey(req.User)).MustGet())
		resp.Last = s.humanTimeFromUnix(t.Get(s.fmtLastUpdatedKey(req.User)).MustGet())

		return nil, nil
	})
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "failed to get played entries: %v", err)
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
