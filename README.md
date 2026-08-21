# automemlimit

[![Go Reference](https://pkg.go.dev/badge/github.com/KimMachineGun/automemlimit.svg)](https://pkg.go.dev/github.com/KimMachineGun/automemlimit)
[![Test](https://github.com/KimMachineGun/automemlimit/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/KimMachineGun/automemlimit/actions/workflows/test.yml)

Automatically set `GOMEMLIMIT` based on the Linux [cgroups(7)](https://man7.org/linux/man-pages/man7/cgroups.7.html) memory limit.

See more details about `GOMEMLIMIT` [here](https://tip.golang.org/doc/gc-guide#Memory_limit).

## Notice

Version `v1.0.0` simplifies the public API and improves cgroup memory limit detection.

Key changes from v0.x:
- `memlimit.Set` replaces the `memlimit.SetGoMemLimit*` functions
- `memlimit.Set` returns the previous `GOMEMLIMIT` instead of `0` when configuration is skipped or an error occurs
- `memlimit.Set` sets `GOMEMLIMIT` to `math.MaxInt64` when the provider returns `memlimit.ErrNoLimit`
- `memlimit.FromCgroup` replaces the version-specific cgroup providers
- `memlimit.WithEnv`, `AUTOMEMLIMIT_EXPERIMENT`, and `AUTOMEMLIMIT_DEBUG` were removed
  - Use `memlimit.FromSystem` for system memory fallback
- `memlimit.WithRefreshInterval` now takes a `context.Context` for cancellation
- `memlimit.WithMin` was added to set a lower bound for `GOMEMLIMIT`

## Installation

```shell
go get github.com/KimMachineGun/automemlimit@latest
```

## Usage

```go
package main

// Importing this package sets GOMEMLIMIT automatically from the cgroup memory limit.
import _ "github.com/KimMachineGun/automemlimit"
```

For more control, use `memlimit.Set` directly:

```go
package main

import (
	"log"

	"github.com/KimMachineGun/automemlimit/memlimit"
)

func main() {
	_, err := memlimit.Set(
		memlimit.WithRatio(0.9),
		memlimit.WithMin(100*1024*1024),
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
```
