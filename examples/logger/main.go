package main

import (
	"log/slog"
	"os"

	"github.com/KimMachineGun/automemlimit/memlimit"
)

func init() {
	memlimit.Set(
		memlimit.WithLogger(slog.New(slog.NewJSONHandler(os.Stderr, nil))),
	)
}

func main() {}
