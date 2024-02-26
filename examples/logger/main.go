package main

import (
	"log/slog"
	"os"

	"github.com/KimMachineGun/automemlimit/memlimit"
)

func init() {
	memlimit.SetGoMemLimitWithOpts(
		memlimit.WithProvider(
			memlimit.Limit(1024*1024*1024),
		),
		memlimit.WithLogger(slog.New(slog.NewJSONHandler(os.Stderr, nil))),
	)
}

func main() {}
