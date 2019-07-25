package played

import (
	"reflect"
	"strconv"
	"unsafe"

	"github.com/tatsuworks/gateway/discordetf"
	"go.uber.org/zap"
)

func (s *Server) startReadRoutines(parallel int) {
	for i := 0; i < parallel; i++ {
		go s.blpopForever()
	}
}

func (s *Server) blpopForever() {
	for {
		res, err := s.rdb.BLPop(0, "gateway:events:presence_update").Result()
		if err != nil {
			s.log.Error("failed to lpop", zap.Error(err))
			continue
		}

		pres, err := discordetf.DecodePlayedPresence(unsafeBytesFromString(res[1]))
		if err != nil {
			s.log.Error("failed to decode presence", zap.Error(err))
			continue
		}

		s.log.Info("presence", zap.Int64("user", pres.UserID), zap.String("game", pres.Game))
		err = s.processPlayed(strconv.FormatInt(pres.UserID, 10), pres.Game)
		if err != nil {
			s.log.Error("failed to process presence", zap.Error(err))
		}
	}
}

// string -> []byte requires a copy since strings are constant.
// this function extracts the underlying slice of the string
// and returns a pointer to it. at 1kb this is 125x faster.
func unsafeBytesFromString(s string) []byte {
	hdr := *(*reflect.StringHeader)(unsafe.Pointer(&s))
	return *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Data: hdr.Data,
		Len:  hdr.Len,
		Cap:  hdr.Len,
	}))
}
