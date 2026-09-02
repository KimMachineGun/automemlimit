// Package automemlimit automatically sets GOMEMLIMIT based on the cgroup memory limit.
//
//	import _ "github.com/KimMachineGun/automemlimit"
//
// By default, it sets GOMEMLIMIT to 90% of the cgroup memory limit.
// Set the AUTOMEMLIMIT environment variable to a ratio in (0.0, 1.0], or "off".
// Use the memlimit package directly for more control.
package automemlimit

import (
	"log/slog"

	"github.com/KimMachineGun/automemlimit/memlimit"
)

func init() {
	memlimit.Set(
		memlimit.WithLogger(slog.Default()),
	)
}
