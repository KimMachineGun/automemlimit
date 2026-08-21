package memlimit

import (
	"context"
	"fmt"
	"math"
	"runtime/debug"
	"sync/atomic"
	"testing"
	"time"
)

func TestSet(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T)
		opts    []Option
		want    int64
		wantErr string
	}{
		{
			name: "with_provider_1gib_0.9",
			opts: []Option{WithProvider(Limit(1073741824)), WithRatio(0.9)},
			want: 966367641,
		},
		{
			name: "with_provider_1gib_1.0",
			opts: []Option{WithProvider(Limit(1073741824)), WithRatio(1.0)},
			want: 1073741824,
		},
		{
			name: "with_provider_maxuint64_capped",
			opts: []Option{WithProvider(Limit(math.MaxUint64)), WithRatio(0.9)},
			want: math.MaxInt64,
		},
		{
			name: "with_min_after_ratio",
			opts: []Option{WithProvider(Limit(1000)), WithRatio(0.5), WithMin(700)},
			want: 700,
		},
		{
			name: "gomemlimit_env",
			setup: func(t *testing.T) {
				debug.SetMemoryLimit(104857600)
				t.Setenv("GOMEMLIMIT", "100MiB")
			},
			opts: []Option{
				WithProvider(Limit(1024 * 1024 * 1024)),
				WithRatio(0.9),
			},
			want: 104857600,
		},
		{
			name: "provider_error",
			setup: func(t *testing.T) {
				debug.SetMemoryLimit(math.MaxInt64)
			},
			opts: []Option{
				WithProvider(func() (uint64, error) {
					return 0, fmt.Errorf("provider failed")
				}),
			},
			want:    math.MaxInt64,
			wantErr: "failed to set GOMEMLIMIT: provider failed",
		},
		{
			name: "err_no_limit",
			setup: func(t *testing.T) {
				debug.SetMemoryLimit(500 * 1024 * 1024)
			},
			opts: []Option{
				WithProvider(func() (uint64, error) {
					return 0, ErrNoLimit
				}),
			},
			want: math.MaxInt64,
		},
		{
			name: "wrapped_err_no_limit",
			setup: func(t *testing.T) {
				debug.SetMemoryLimit(500 * 1024 * 1024)
			},
			opts: []Option{
				WithProvider(func() (uint64, error) {
					return 0, fmt.Errorf("wrapped: %w", ErrNoLimit)
				}),
			},
			want: math.MaxInt64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				debug.SetMemoryLimit(math.MaxInt64)
			})
			if tt.setup != nil {
				tt.setup(t)
			}

			got, err := Set(tt.opts...)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Set() error = %v, want nil", err)
				}
			} else if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("Set() error = %v, want error %q", err, tt.wantErr)
			}

			if got != tt.want {
				t.Fatalf("Set() = %v, want %v", got, tt.want)
			}
			actual := debug.SetMemoryLimit(-1)
			if actual != tt.want {
				t.Fatalf("GOMEMLIMIT = %v, want %v", actual, tt.want)
			}
		})
	}
}

func TestSetAUTOMEMLIMIT(t *testing.T) {
	tests := []struct {
		name    string
		env     string
		limit   uint64
		want    int64
		wantErr string
	}{
		{
			name:  "0.9",
			env:   "0.9",
			limit: 1000,
			want:  900,
		},
		{
			name:  "1.0",
			env:   "1.0",
			limit: 1000,
			want:  1000,
		},
		{
			name:  "off",
			env:   "off",
			limit: 1000,
			want:  math.MaxInt64,
		},
		{
			name:    "invalid",
			env:     "invalid",
			limit:   1000,
			want:    math.MaxInt64,
			wantErr: "cannot parse AUTOMEMLIMIT: invalid",
		},
		{
			name:    "zero",
			env:     "0",
			limit:   1000,
			want:    math.MaxInt64,
			wantErr: "failed to set GOMEMLIMIT: invalid ratio: 0.000000, ratio should be in the range (0.0,1.0]",
		},
		{
			name:    "over_1",
			env:     "1.5",
			limit:   1000,
			want:    math.MaxInt64,
			wantErr: "failed to set GOMEMLIMIT: invalid ratio: 1.500000, ratio should be in the range (0.0,1.0]",
		},
		{
			name:    "nan",
			env:     "NaN",
			limit:   1000,
			want:    math.MaxInt64,
			wantErr: "failed to set GOMEMLIMIT: invalid ratio: NaN, ratio should be in the range (0.0,1.0]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				debug.SetMemoryLimit(math.MaxInt64)
			})
			t.Setenv("AUTOMEMLIMIT", tt.env)

			debug.SetMemoryLimit(math.MaxInt64)

			got, err := Set(WithProvider(Limit(tt.limit)))
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Set() error = %v, want nil", err)
				}
			} else if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("Set() error = %v, want error %q", err, tt.wantErr)
			}

			if got != tt.want {
				t.Fatalf("Set() = %v, want %v", got, tt.want)
			}
			actual := debug.SetMemoryLimit(-1)
			if actual != tt.want {
				t.Fatalf("GOMEMLIMIT = %v, want %v", actual, tt.want)
			}
		})
	}
}

func waitForGoMemLimit(t *testing.T, want int64) {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for {
		got := debug.SetMemoryLimit(-1)
		if got == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("GOMEMLIMIT = %v, want %v", got, want)
		}
		time.Sleep(time.Millisecond)
	}
}

func waitForRefresh(t *testing.T, done <-chan struct{}) {
	t.Helper()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("refresh did not stop after context cancellation")
	}
}

func TestSetRefresh(t *testing.T) {
	t.Cleanup(func() {
		debug.SetMemoryLimit(math.MaxInt64)
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	limits := make(chan uint64, 2)
	limits <- 1000
	provider := func() (uint64, error) {
		select {
		case limit := <-limits:
			return limit, nil
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}

	got, err := Set(
		WithProvider(provider),
		WithRatio(0.5),
		WithRefreshInterval(ctx, 10*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("Set() error = %v, want nil", err)
	}
	if got != 500 {
		t.Fatalf("Set() = %v, want %v", got, 500)
	}

	limits <- 4000
	waitForGoMemLimit(t, 2000)
}

func TestRefreshContinuesAfterProviderError(t *testing.T) {
	t.Cleanup(func() {
		debug.SetMemoryLimit(math.MaxInt64)
	})
	debug.SetMemoryLimit(500)

	ctx, cancel := context.WithCancel(context.Background())

	recoveryStarted := make(chan struct{})
	allowRecovery := make(chan struct{}, 1)
	var callCount int
	provider := func() (uint64, error) {
		callCount++
		switch callCount {
		case 1:
			return 0, fmt.Errorf("provider failed")
		case 2:
			close(recoveryStarted)
			<-allowRecovery
		}
		return 2000, nil
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		refresh(ctx, provider, memlimitLogger(nil), 10*time.Millisecond)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case allowRecovery <- struct{}{}:
		default:
		}
		waitForRefresh(t, done)
	})

	select {
	case <-recoveryStarted:
	case <-time.After(time.Second):
		t.Fatal("provider retry did not occur")
	}
	if got := debug.SetMemoryLimit(-1); got != 500 {
		t.Fatalf("GOMEMLIMIT after provider error = %v, want %v", got, 500)
	}

	allowRecovery <- struct{}{}
	waitForGoMemLimit(t, 2000)
	cancel()
	waitForRefresh(t, done)
}

func TestSetRefreshAfterInitialError(t *testing.T) {
	t.Cleanup(func() {
		debug.SetMemoryLimit(math.MaxInt64)
	})

	snapshot := int64(123456789)
	wantErr := "failed to set GOMEMLIMIT: provider failed"
	debug.SetMemoryLimit(snapshot)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	var callCount atomic.Int32
	got, err := Set(
		WithProvider(func() (uint64, error) {
			switch callCount.Add(1) {
			case 1:
				return 0, fmt.Errorf("provider failed")
			case 2:
				return 1000, nil
			default:
				<-ctx.Done()
				return 0, ctx.Err()
			}
		}),
		WithRatio(1),
		WithRefreshInterval(ctx, 10*time.Millisecond),
	)
	if err == nil || err.Error() != wantErr {
		t.Fatalf("Set() error = %v, want error %q", err, wantErr)
	}
	if got != snapshot {
		t.Fatalf("Set() = %v, want %v", got, snapshot)
	}

	waitForGoMemLimit(t, 1000)
}

func TestSetRefreshDisabled(t *testing.T) {
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name     string
		ctx      context.Context
		interval time.Duration
	}{
		{name: "nil_context", interval: 10 * time.Millisecond},
		{name: "canceled_context", ctx: canceledCtx, interval: 10 * time.Millisecond},
		{name: "zero_interval", ctx: context.Background()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				debug.SetMemoryLimit(math.MaxInt64)
			})

			calls := make(chan struct{}, 2)
			_, err := Set(
				WithProvider(func() (uint64, error) {
					calls <- struct{}{}
					return 1000, nil
				}),
				WithRatio(1),
				WithRefreshInterval(tt.ctx, tt.interval),
			)
			if err != nil {
				t.Fatalf("Set() error = %v, want nil", err)
			}

			<-calls
			select {
			case <-calls:
				t.Fatal("provider was called after the initial update")
			case <-time.After(30 * time.Millisecond):
			}
		})
	}
}

func TestSetSkipDoesNotStartRefresh(t *testing.T) {
	tests := []struct {
		name string
		env  string
		val  string
	}{
		{name: "gomemlimit", env: envGOMEMLIMIT, val: "100MiB"},
		{name: "automemlimit_off", env: envAUTOMEMLIMIT, val: "off"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				debug.SetMemoryLimit(math.MaxInt64)
			})
			t.Setenv(tt.env, tt.val)

			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)

			called := make(chan struct{}, 1)
			_, err := Set(
				WithProvider(func() (uint64, error) {
					called <- struct{}{}
					return 1000, nil
				}),
				WithRefreshInterval(ctx, 10*time.Millisecond),
			)
			if err != nil {
				t.Fatalf("Set() error = %v, want nil", err)
			}

			select {
			case <-called:
				t.Fatal("provider was called")
			case <-time.After(30 * time.Millisecond):
			}
		})
	}
}
