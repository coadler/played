package main

import (
	"context"
	"os"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/sloghuman"
	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/apple/foundationdb/bindings/go/src/fdb/directory"
	"github.com/coadler/played"
)

func main() {
	var (
		logger = sloghuman.Make(os.Stdout)
		ctx    = context.Background()
	)

	fdb.MustAPIVersion(610)
	db := fdb.MustOpenDefault()

	dir, err := directory.Open(db, []string{"played"}, nil)
	if err != nil {
		logger.Fatal(ctx, "failed to create fdb directory", slog.Error(err))
	}

	subs := played.Subspaces{
		FirstSeen:   dir.Sub("first-seen"),
		LastUpdated: dir.Sub("last_updated"),
		Current:     dir.Sub("current"),
		Played:      dir.Sub("played"),
	}

	_, err = db.Transact(func(t fdb.Transaction) (interface{}, error) {
		r, _ := fdb.PrefixRange(subs.Current.FDBKey())
		t.ClearRange(r)
		r, _ = fdb.PrefixRange(subs.LastUpdated.FDBKey())
		t.ClearRange(r)

		return nil, nil
	})
	if err != nil {
		logger.Fatal(ctx, "failed to transact", slog.Error(err))
	}
}
