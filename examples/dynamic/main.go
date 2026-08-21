package main

import (
	"bytes"
	"context"
	"errors"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/KimMachineGun/automemlimit/memlimit"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	_, err := memlimit.Set(
		memlimit.WithProvider(
			fileProvider("limit.txt"),
		),
		memlimit.WithRefreshInterval(ctx, 5*time.Second),
		memlimit.WithLogger(slog.Default()),
	)
	if err != nil {
		log.Fatal(err)
	}

	<-ctx.Done()
	slog.Info("shutdown")
}

func fileProvider(path string) memlimit.Provider {
	return func() (uint64, error) {
		b, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return memlimit.ApplyFallback(memlimit.FromCgroup, memlimit.FromSystem)()
			}
			return 0, err
		}

		b = bytes.TrimSpace(b)
		if len(b) == 0 {
			return 0, memlimit.ErrNoLimit
		}

		return strconv.ParseUint(string(b), 10, 64)
	}
}
