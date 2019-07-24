package played

import (
	"encoding/binary"
	"time"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/dustin/go-humanize"
)

var RangeWantAll = fdb.RangeOptions{
	Mode: fdb.StreamingModeWantAll,
}

func humanTimeFromUnix(b []byte) string {
	if b == nil {
		return "Never"
	}

	secs := int64(binary.LittleEndian.Uint64(b))
	return humanize.Time(time.Unix(secs, 0))
}
