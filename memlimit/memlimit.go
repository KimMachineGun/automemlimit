package memlimit

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"runtime/debug"
	"strconv"
)

const (
	envGOMEMLIMIT   = "GOMEMLIMIT"
	envAUTOMEMLIMIT = "AUTOMEMLIMIT"
	// Deprecated: use memlimit.WithLogger instead
	envAUTOMEMLIMIT_DEBUG = "AUTOMEMLIMIT_DEBUG"

	defaultAUTOMEMLIMIT = 0.9
)

var (
	// ErrNoLimit is returned when the memory limit is not set.
	ErrNoLimit = errors.New("memory is not limited")
)

type config struct {
	logger   *slog.Logger
	ratio    float64
	provider Provider
}

// Option is a function that configures the behavior of SetGoMemLimitWithOptions.
type Option func(cfg *config)

// WithRatio configures the ratio of the memory limit to set as GOMEMLIMIT.
//
// Default: 0.9
func WithRatio(ratio float64) Option {
	return func(cfg *config) {
		cfg.ratio = ratio
	}
}

// WithEnv configures whether to use environment variables.
//
// Default: false
//
// Deprecated: currently this does nothing.
func WithEnv() Option {
	return func(cfg *config) {}
}

// WithProvider configures the provider.
//
// Default: FromCgroup
func WithProvider(provider Provider) Option {
	return func(cfg *config) {
		cfg.provider = provider
	}
}

// WithLogger configures the logger.
// It automatically attaches the "package" attribute to the logs.
//
// Default: slog.New(noopLogger{})
func WithLogger(logger *slog.Logger) Option {
	return func(cfg *config) {
		cfg.logger = memlimitLogger(logger)
	}
}

func memlimitLogger(logger *slog.Logger) *slog.Logger {
	if logger == nil {
		return slog.New(noopLogger{})
	}
	return logger.With(slog.String("package", "memlimit"))
}

// SetGoMemLimitWithOpts sets GOMEMLIMIT with options and environment variables.
//
// You can configure how much memory of the cgroup's memory limit to set as GOMEMLIMIT
// through AUTOMEMLIMIT envrironment variable in the half-open range (0.0,1.0].
//
// If AUTOMEMLIMIT is not set, it defaults to 0.9. (10% is the headroom for memory sources the Go runtime is unaware of.)
// If GOMEMLIMIT is already set or AUTOMEMLIMIT=off, this function does nothing.
//
// Options:
//   - WithRatio
//   - WithProvider
//   - WithLogger
func SetGoMemLimitWithOpts(opts ...Option) (_ int64, _err error) {
	cfg := &config{
		logger:   slog.New(noopLogger{}),
		ratio:    defaultAUTOMEMLIMIT,
		provider: FromCgroup,
	}
	// TODO: remove this
	if debug, ok := os.LookupEnv(envAUTOMEMLIMIT_DEBUG); ok {
		logger := memlimitLogger(slog.Default())
		logger.Warn("AUTOMEMLIMIT_DEBUG is deprecated, use memlimit.WithLogger instead")
		if debug == "true" {
			cfg.logger = logger
		}
	}
	for _, opt := range opts {
		opt(cfg)
	}
	defer func() {
		if _err != nil {
			cfg.logger.Error("failed to set GOMEMLIMIT", slog.Any("error", _err))
		}
	}()

	snapshot := debug.SetMemoryLimit(-1)
	defer func() {
		err := recover()
		if err != nil {
			if _err != nil {
				cfg.logger.Error("failed to set GOMEMLIMIT", slog.Any("error", _err))
			}
			_err = fmt.Errorf("panic during setting the Go's memory limit, rolling back to previous value %d: %v", snapshot, err)
			debug.SetMemoryLimit(snapshot)
		}
	}()

	if val, ok := os.LookupEnv(envGOMEMLIMIT); ok {
		cfg.logger.Info("GOMEMLIMIT is set already, skipping", slog.String("GOMEMLIMIT", val))
		return 0, nil
	}

	ratio := cfg.ratio
	if val, ok := os.LookupEnv(envAUTOMEMLIMIT); ok {
		if val == "off" {
			cfg.logger.Debug("AUTOMEMLIMIT is set to off, skipping")
			return 0, nil
		}
		_ratio, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0, fmt.Errorf("cannot parse AUTOMEMLIMIT: %s", val)
		}
		ratio = _ratio
	}

	limit, err := setGoMemLimit(ApplyRatio(cfg.provider, ratio))
	if err != nil {
		return 0, fmt.Errorf("failed to set GOMEMLIMIT: %w", err)
	}

	cfg.logger.Info("GOMEMLIMIT is set", slog.Int64("GOMEMLIMIT", limit))

	return limit, nil
}

func SetGoMemLimitWithEnv() {
	_, _ = SetGoMemLimitWithOpts()
}

// SetGoMemLimit sets GOMEMLIMIT with the value from the cgroup's memory limit and given ratio.
func SetGoMemLimit(ratio float64) (int64, error) {
	return SetGoMemLimitWithOpts(WithRatio(ratio))
}

// SetGoMemLimitWithProvider sets GOMEMLIMIT with the value from the given provider and ratio.
func SetGoMemLimitWithProvider(provider Provider, ratio float64) (int64, error) {
	return SetGoMemLimitWithOpts(WithProvider(provider), WithRatio(ratio))
}

func setGoMemLimit(provider Provider) (int64, error) {
	limit, err := provider()
	if err != nil {
		return 0, err
	}
	capped := cappedU64ToI64(limit)
	debug.SetMemoryLimit(capped)
	return capped, nil
}

func cappedU64ToI64(limit uint64) int64 {
	if limit > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(limit)
}
