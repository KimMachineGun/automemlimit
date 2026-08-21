// Package automemlimit automatically sets GOMEMLIMIT based on the cgroup memory limit.
//
//	import _ "github.com/KimMachineGun/automemlimit"
//
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
