package automemlimit

import (
	"log/slog"

	"github.com/KimMachineGun/automemlimit/memlimit"
)

func init() {
	memlimit.SetGoMemLimitWithOpts(
		memlimit.WithEnv(),
		memlimit.WithLogger(slog.Default()),
	)
}
