# automemlimit

[![Go Reference](https://pkg.go.dev/badge/github.com/KimMachineGun/automemlimit.svg)](https://pkg.go.dev/github.com/KimMachineGun/automemlimit)
[![Go Report Card](https://goreportcard.com/badge/github.com/KimMachineGun/automemlimit)](https://goreportcard.com/report/github.com/KimMachineGun/automemlimit)
[![Test](https://github.com/KimMachineGun/automemlimit/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/KimMachineGun/automemlimit/actions/workflows/test.yml)

Automatically set `GOMEMLIMIT` to match Linux [cgroups(7)](https://man7.org/linux/man-pages/man7/cgroups.7.html) memory limit.

See more details about `GOMEMLIMIT` [here](https://tip.golang.org/doc/gc-guide#Memory_limit).

## Notice

Version `v1.0.0` introduces breaking changes to simplify the API. The library now provides a single `memlimit.Set` function, and system memory limits are used automatically as a fallback when cgroup limits are unavailable.

## Installation

```shell
go get github.com/KimMachineGun/automemlimit@latest
```

## Usage

```go
package main

// By default, it sets `GOMEMLIMIT` to 90% of cgroup's memory limit.
// System memory is used as a fallback if cgroup is not available.
import _ "github.com/KimMachineGun/automemlimit"
```

> **Note:** The automatic system-memory fallback applies only when you use the convenience import shown above. Calling `memlimit.Set` directly defaults to the cgroup provider; add an explicit fallback if you need it.

or

```go
package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/KimMachineGun/automemlimit/memlimit"
)

func main() {
	// Set GOMEMLIMIT to 90% of cgroup's memory limit (no automatic fallback)
	memlimit.Set()

	// With explicit system memory fallback
	memlimit.Set(
		memlimit.WithProvider(
			memlimit.ApplyFallback(memlimit.FromCgroup, memlimit.FromSystem),
		),
		memlimit.WithRatio(0.9),
		memlimit.WithLogger(slog.Default()),
	)

	// With refresh interval
	refreshCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	memlimit.Set(
		memlimit.WithRefreshInterval(refreshCtx, 1*time.Minute),
	)
}
```
