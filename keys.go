package played

import (
	"fmt"
	"sync"
	"time"
)

type playedEntry struct {
	Name     string
	Duration time.Duration
}

type byDuration []*playedEntry

func (a byDuration) Len() int           { return len(a) }
func (a byDuration) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byDuration) Less(i, j int) bool { return a[i].Duration >= a[j].Duration }

type playedUser struct {
	sync.RWMutex
	Entries     map[string]*playedEntry
	Current     string
	LastChanged time.Time
	FirstSeen   time.Time
}

var (
	UserEntryKey = func(user, game string) []byte {
		return []byte(fmt.Sprintf("played:%s:games:%s", user, game))
	}

	UserCurrentKey = func(user string) []byte {
		return []byte(fmt.Sprintf("played:%s:current", user))
	}

	UserLastChangedKey = func(user string) []byte {
		return []byte(fmt.Sprintf("played:%s:lastchanged", user))
	}

	UserFirstSeenKey = func(user string) []byte {
		return []byte(fmt.Sprintf("played:%s:firstseen", user))
	}
)
