package played

import (
	"fmt"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/apple/foundationdb/bindings/go/src/fdb/tuple"
	"golang.org/x/xerrors"
)

func (s *Server) fmtPlayedUserRange(user string) fdb.KeyRange {
	pre, _ := fdb.PrefixRange(s.subs.Played.Pack(tuple.Tuple{user}))
	return pre
}

func (s *Server) fmtPlayedUserGame(user, game string) fdb.Key {
	return s.subs.Played.Pack(tuple.Tuple{user, game})
}

func (s *Server) unpackPlayed(k fdb.Key) (user string, game string, err error) {
	tup, err := s.subs.Played.Unpack(k)
	if err != nil {
		return "", "", xerrors.Errorf("failed to unpack played key: %w", err)
	}

	if len(tup) != 2 {
		return "", "", xerrors.Errorf("unknown played key length. expected 2, got %v", len(tup))
	}

	return tup[0].(string), tup[1].(string), nil
}

func (s *Server) fmtFirstSeenKey(user string) fdb.Key {
	return s.subs.FirstSeen.Pack(tuple.Tuple{user})
}

func (s *Server) fmtLastUpdatedKey(user string) fdb.Key {
	return s.subs.LastUpdated.Pack(tuple.Tuple{user})
}

func (s *Server) fmtCurrentGameKey(user string) fdb.Key {
	return s.subs.Current.Pack(tuple.Tuple{user})
}

func fmtWhitelistKey(user string) string {
	return fmt.Sprintf("played:whitelist:%s", user)
}

const whitelistPrefix = "played:whitelist:*"
