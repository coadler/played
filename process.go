package played

import (
	"bytes"
	"encoding/binary"
	"time"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
)

func (s *Server) processPlayed(user, game string) error {
	// w, err := s.rdb.Exists(fmtWhitelistKey(user)).Result()
	// if err != nil {
	// 	return err
	// }

	// if w != 1 {
	// 	return nil
	// }

	var (
		fsKey   = s.fmtFirstSeenKey(user)
		curKey  = s.fmtCurrentGameKey(user)
		lastKey = s.fmtLastUpdatedKey(user)
		err     error
	)

	s.db.Transact(func(t fdb.Transaction) (_ interface{}, err error) {
		// because of the low cost of time.Now and PutUint64 i'd rather
		// prefer idempotence because this will be retried if there is a conflict
		//
		// little endian is used because foundationdb's atomic add function
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
		if len(curVal) == 0 {
			return
		}

		toAdd := [8]byte{}
		binary.LittleEndian.PutUint32(
			toAdd[:],
			uint32(timeNow.Sub(lastChanged).Seconds()),
		)

		t.Add(s.fmtPlayedUserGame(user, string(curVal)), toAdd[:])
		return
	})

	return err
}
