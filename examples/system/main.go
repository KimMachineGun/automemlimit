package main

import "github.com/KimMachineGun/automemlimit/memlimit"

func init() {
	memlimit.Set(
		memlimit.WithProvider(
			memlimit.ApplyFallback(
				memlimit.FromCgroup,
				memlimit.FromSystem,
			),
		),
	)
}

func main() {}
