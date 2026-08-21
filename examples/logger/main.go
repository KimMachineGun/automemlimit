package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/KimMachineGun/automemlimit/memlimit"
)

func main() {
	_, err := memlimit.Set(
		memlimit.WithProvider(
			memlimit.Limit(1024*1024*1024),
		),
		memlimit.WithLogger(slog.New(slog.NewJSONHandler(os.Stderr, nil))),
	)
	if err != nil {
		log.Fatal(err)
	}
}
