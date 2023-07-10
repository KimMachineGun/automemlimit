package memlimit

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"runtime/debug"
	"strconv"
)

const (
	envGOMEMLIMIT         = "GOMEMLIMIT"
	envAUTOMEMLIMIT       = "AUTOMEMLIMIT"
	envAUTOMEMLIMIT_DEBUG = "AUTOMEMLIMIT_DEBUG"

	defaultAUTOMEMLIMIT = 0.9
)

var (
	// ErrNoLimit is returned when the memory limit is not set.
	ErrNoLimit = errors.New("memory is not limited")
	// ErrNoCgroup is returned when the process is not in cgroup.
	ErrNoCgroup = errors.New("process is not in cgroup")
	// ErrCgroupsNotSupported is returned when the system does not support cgroups.
	ErrCgroupsNotSupported = errors.New("cgroups is not supported on this system")

	logger = log.New(io.Discard, "", log.LstdFlags)
)

type config struct {
	printf   func(string, ...interface{})
	ratio    float64
	provider Provider
	env      bool
}

// An Option alters the behavior of SetGoMemLimitWithEnv.
type Option func(cfg *config)

// WithLogger uses the supplied printf implementation for log output.
// By default log.Printf.
func WithLogger(printf func(string, ...interface{})) Option {
	return Option(func(cfg *config) {
		cfg.printf = printf
	})
}

// WithRatio configure memory limit.
// By default `defaultAUTOMEMLIMIT`.
func WithRatio(ratio float64) Option {
	return Option(func(cfg *config) {
		cfg.ratio = ratio
	})
}

// WithEnv configure memory limit from environment variable.
func WithEnv() Option {
	return Option(func(cfg *config) {
		cfg.env = true
	})
}

// WithProvider configure provider.
// By default `FromCgroup`.
func WithProvider(provider Provider) Option {
	return Option(func(cfg *config) {
		cfg.provider = provider
	})
}

// SetGoMemLimitWithEnv sets GOMEMLIMIT with the value from the environment variable.
// You can configure how much memory of the cgroup's memory limit to set as GOMEMLIMIT
// through AUTOMEMLIMIT in the half-open range (0.0,1.0].
//
// If AUTOMEMLIMIT is not set, it defaults to 0.9. (10% is the headroom for memory sources the Go runtime is unaware of.)
// If GOMEMLIMIT is already set or AUTOMEMLIMIT=off, this function does nothing.
func SetGoMemLimitWithEnv() {
	snapshot := debug.SetMemoryLimit(-1)
	defer func() {
		err := recover()
		if err != nil {
			logger.Printf("panic during SetGoMemLimitWithEnv, rolling back to previous value %d: %v\n", snapshot, err)
			debug.SetMemoryLimit(snapshot)
		}
	}()

	if os.Getenv(envAUTOMEMLIMIT_DEBUG) == "true" {
		logger = log.Default()
	}

	if val, ok := os.LookupEnv(envGOMEMLIMIT); ok {
		logger.Printf("GOMEMLIMIT is set already, skipping: %s\n", val)
		return
	}

	ratio := defaultAUTOMEMLIMIT
	if val, ok := os.LookupEnv(envAUTOMEMLIMIT); ok {
		if val == "off" {
			logger.Printf("AUTOMEMLIMIT is set to off, skipping\n")
			return
		}
		_ratio, err := strconv.ParseFloat(val, 64)
		if err != nil {
			logger.Printf("cannot parse AUTOMEMLIMIT: %s\n", val)
			return
		}
		ratio = _ratio
	}
	if ratio <= 0 || ratio > 1 {
		logger.Printf("invalid AUTOMEMLIMIT: %f\n", ratio)
		return
	}

	limit, err := SetGoMemLimit(ratio)
	if err != nil {
		logger.Printf("failed to set GOMEMLIMIT: %v\n", err)
		return
	}

	logger.Printf("GOMEMLIMIT=%d\n", limit)
}

// SetGoMemLimitWithOptions sets GOMEMLIMIT with options.
// Options:
// - Environment variable (see more `SetGoMemLimitWithEnv`)
// - Provider (see more `SetGoMemLimitWithProvider`)
// - Logger
// - Ratio
func SetGoMemLimitWithOptions(opts ...Option) {
	cfg := &config{
		printf:   logger.Printf,
		ratio:    defaultAUTOMEMLIMIT,
		provider: FromCgroup,
		env:      false,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	ratio := cfg.ratio

	snapshot := debug.SetMemoryLimit(-1)
	defer func() {
		err := recover()
		if err != nil {
			cfg.printf("panic during SetGoMemLimitWithEnv, rolling back to previous value %d: %v\n", snapshot, err)
			debug.SetMemoryLimit(snapshot)
		}
	}()

	if cfg.env {
		if val, ok := os.LookupEnv(envGOMEMLIMIT); ok {
			cfg.printf("GOMEMLIMIT is set already, skipping: %s\n", val)
			return
		}

		if val, ok := os.LookupEnv(envAUTOMEMLIMIT); ok {
			if val == "off" {
				cfg.printf("AUTOMEMLIMIT is set to off, skipping\n")
				return
			}
			_ratio, err := strconv.ParseFloat(val, 64)
			if err != nil {
				cfg.printf("cannot parse AUTOMEMLIMIT: %s\n", val)
				return
			}
			ratio = _ratio
		}

		if ratio <= 0 || ratio > 1 {
			cfg.printf("invalid AUTOMEMLIMIT: %f\n", ratio)
			return
		}
	}

	limit, err := SetGoMemLimitWithProvider(cfg.provider, ratio)
	if err != nil {
		cfg.printf("failed to set GOMEMLIMIT: %v\n", err)
		return
	}

	cfg.printf("GOMEMLIMIT=%s\n", prettyByteSize(limit))
}

// SetGoMemLimit sets GOMEMLIMIT with the value from the cgroup's memory limit and given ratio.
func SetGoMemLimit(ratio float64) (int64, error) {
	return SetGoMemLimitWithProvider(FromCgroup, ratio)
}

// Provider is a function that returns the memory limit.
type Provider func() (uint64, error)

// SetGoMemLimitWithProvider sets GOMEMLIMIT with the value from the given provider and ratio.
func SetGoMemLimitWithProvider(provider Provider, ratio float64) (int64, error) {
	limit, err := provider()
	if err != nil {
		return 0, err
	}
	goMemLimit := cappedFloat2Int(float64(limit) * ratio)
	debug.SetMemoryLimit(goMemLimit)
	return goMemLimit, nil
}

func cappedFloat2Int(f float64) int64 {
	if f > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(f)
}

// Limit is a helper Provider function that returns the given limit.
func Limit(limit uint64) func() (uint64, error) {
	return func() (uint64, error) {
		return limit, nil
	}
}

func prettyByteSize(b int64) string {
	bf := float64(b)
	for _, unit := range []string{"", "Ki", "Mi", "Gi", "Ti", "Pi", "Ei", "Zi"} {
		if bf < 1024.0 {
			return fmt.Sprintf("%3.1f%sB", bf, unit)
		}
		bf /= 1024.0
	}
	return fmt.Sprintf("%.1fYiB", bf)
}
