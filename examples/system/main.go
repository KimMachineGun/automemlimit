package main

import (
	"log"

	"github.com/KimMachineGun/automemlimit/memlimit"
)

func main() {
	_, err := memlimit.Set(
		memlimit.WithProvider(
			memlimit.ApplyFallback(
				memlimit.FromCgroup,
				memlimit.FromSystem,
			),
		),
	)
	if err != nil {
		log.Fatal(err)
	}
}
