package main

import (
	"github.com/KimMachineGun/automemlimit/memlimit"
	sigar "github.com/cloudfoundry/gosigar"
)

func init() {
	memlimit.SetGoMemLimitWithOpts(
		memlimit.WithEnv(),
		memlimit.WithProvider(
			memlimit.ApplyFallback(
				memlimit.FromCgroup,
				FromGoSigar,
			),
		),
	)
}

func main() {}

func FromGoSigar() (uint64, error) {
	var mem sigar.Mem
	err := mem.Get()
	return mem.Total, err
}
