package memlimit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"runtime/debug"
	"strconv"
	"time"
)

const (
	envGOMEMLIMIT   = "GOMEMLIMIT"
	envAUTOMEMLIMIT = "AUTOMEMLIMIT"

	defaultAUTOMEMLIMIT = 0.9
)

// ErrNoLimit is returned when the memory limit is not set.
var ErrNoLimit = errors.New("memory is not limited")

type config struct {
	logger     *slog.Logger
	ratio      float64
	provider   Provider
	refresh    time.Duration
	refreshCtx context.Context
}

// Option configures the behavior of Set.
type Option func(cfg *config)

// WithRatio configures the ratio of the memory limit to set as GOMEMLIMIT.
//
// Default: 0.9
func WithRatio(ratio float64) Option {
	return func(cfg *config) {
		cfg.ratio = ratio
	}
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
// Default: slog.New(discardHandler{})
func WithLogger(logger *slog.Logger) Option {
	return func(cfg *config) {
		cfg.logger = memlimitLogger(logger)
	}
}

// WithRefreshInterval configures the refresh interval for automemlimit.
// The provided context controls the refresh goroutine lifecycle.
// If a refresh interval is greater than 0, automemlimit periodically fetches
// the memory limit from the provider and reapplies it if it has changed.
// If the provider returns an error, it logs the error and continues.
// ErrNoLimit is treated as math.MaxInt64.
//
// Default: 0 (no refresh)
func WithRefreshInterval(ctx context.Context, refresh time.Duration) Option {
	return func(cfg *config) {
		cfg.refresh = refresh
		cfg.refreshCtx = ctx
	}
}

// Set sets GOMEMLIMIT with the given options.
//
// Options:
//   - WithRatio
//   - WithProvider
//   - WithLogger
//   - WithRefreshInterval
func Set(opts ...Option) (_ int64, _err error) {
	// init config
	cfg := &config{
		logger:   slog.New(discardHandler{}),
		ratio:    defaultAUTOMEMLIMIT,
		provider: FromCgroup,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// log error if any on return
	defer func() {
		if _err != nil {
			cfg.logger.Error("failed to set GOMEMLIMIT", slog.Any("error", _err))
		}
	}()

	// rollback to previous memory limit on panic
	snapshot := debug.SetMemoryLimit(-1)
	defer rollbackOnPanic(cfg.logger, snapshot, &_err)

	// check if GOMEMLIMIT is already set
	if val, ok := os.LookupEnv(envGOMEMLIMIT); ok {
		cfg.logger.Info("GOMEMLIMIT is already set, skipping", slog.String(envGOMEMLIMIT, val))
		return snapshot, nil
	}

	// parse AUTOMEMLIMIT
	ratio := cfg.ratio
	if val, ok := os.LookupEnv(envAUTOMEMLIMIT); ok {
		if val == "off" {
			cfg.logger.Info("AUTOMEMLIMIT is set to off, skipping")
			return snapshot, nil
		}
		r, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return snapshot, fmt.Errorf("cannot parse AUTOMEMLIMIT: %s", val)
		}
		ratio = r
	}

	// apply ratio to the provider
	provider := capProvider(ApplyRatio(cfg.provider, ratio))

	// set the memory limit and start refresh
	limit, err := updateGoMemLimit(uint64(snapshot), provider, cfg.logger)
	// keep the refresh loop running so it can notice when the provider starts returning limits, even if the first update fails
	if cfg.refresh > 0 && cfg.refreshCtx != nil {
		go refreshWithContext(cfg.refreshCtx, provider, cfg.logger, cfg.refresh)
	}
	if err != nil {
		if errors.Is(err, ErrNoLimit) {
			cfg.logger.Info("memory is not limited, skipping")
			return snapshot, nil
		}
		return 0, fmt.Errorf("failed to set GOMEMLIMIT: %w", err)
	}

	return int64(limit), nil
}

// updateGoMemLimit updates the Go's memory limit, if it has changed.
func updateGoMemLimit(currLimit uint64, provider Provider, logger *slog.Logger) (uint64, error) {
	newLimit, err := provider()
	if err != nil {
		return 0, err
	}

	if newLimit == currLimit {
		logger.Debug("GOMEMLIMIT is not changed, skipping", slog.Uint64(envGOMEMLIMIT, newLimit))
		return newLimit, nil
	}

	debug.SetMemoryLimit(int64(newLimit))
	logger.Info("GOMEMLIMIT is updated", slog.Uint64(envGOMEMLIMIT, newLimit), slog.Uint64("previous", currLimit))

	return newLimit, nil
}

// refreshWithContext periodically fetches the memory limit from the provider and reapplies it if it has changed.
// The context is used to control the lifecycle of the refresh goroutine.
func refreshWithContext(ctx context.Context, provider Provider, logger *slog.Logger, refresh time.Duration) {
	if refresh == 0 {
		return
	}

	provider = noErrNoLimitProvider(provider)

	ticker := time.NewTicker(refresh)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := func() (_err error) {
				snapshot := debug.SetMemoryLimit(-1)
				defer rollbackOnPanic(logger, snapshot, &_err)

				_, err := updateGoMemLimit(uint64(snapshot), provider, logger)
				if err != nil {
					return err
				}

				return nil
			}()
			if err != nil {
				logger.Error("failed to refresh GOMEMLIMIT", slog.Any("error", err))
			}
		}
	}
}

// rollbackOnPanic rollbacks to the snapshot on panic.
// Since it uses recover, it should be called in a deferred function.
func rollbackOnPanic(logger *slog.Logger, snapshot int64, err *error) {
	panicErr := recover()
	if panicErr != nil {
		if *err != nil {
			logger.Error("failed to set GOMEMLIMIT", slog.Any("error", *err))
		}
		*err = fmt.Errorf("panic during setting the Go's memory limit, rolling back to previous limit %d: %v",
			snapshot, panicErr,
		)
		debug.SetMemoryLimit(snapshot)
	}
}

func noErrNoLimitProvider(provider Provider) Provider {
	return func() (uint64, error) {
		limit, err := provider()
		if errors.Is(err, ErrNoLimit) {
			return math.MaxInt64, nil
		}
		return limit, err
	}
}

func capProvider(provider Provider) Provider {
	return func() (uint64, error) {
		limit, err := provider()
		if err != nil {
			return 0, err
		} else if limit > math.MaxInt64 {
			return math.MaxInt64, nil
		}
		return limit, nil
	}
}
