package main

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/KimMachineGun/automemlimit/memlimit"
)

func init() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	refreshCtx := context.Background()
	memlimit.Set(
		memlimit.WithProvider(
			FileProvider("limit.txt"),
		),
		memlimit.WithRefreshInterval(refreshCtx, 5*time.Second),
		memlimit.WithLogger(slog.Default()),
	)
}

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	s := <-c
	slog.Info("signal captured", slog.Any("signal", s))
}

func FileProvider(path string) memlimit.Provider {
	return func() (uint64, error) {
		b, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return memlimit.FromCgroup()
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
