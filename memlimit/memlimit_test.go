package memlimit

import (
	"fmt"
	"math"
	"runtime/debug"
	"sync/atomic"
	"testing"
	"time"
)

func TestLimit(t *testing.T) {
	type args struct {
		limit uint64
	}
	tests := []struct {
		name    string
		args    args
		want    uint64
		wantErr error
	}{
		{
			name: "0bytes",
			args: args{
				limit: 0,
			},
			want:    0,
			wantErr: nil,
		},
		{
			name: "1kib",
			args: args{
				limit: 1024,
			},
			want:    1024,
			wantErr: nil,
		},
		{
			name: "1mib",
			args: args{
				limit: 1024 * 1024,
			},
			want:    1024 * 1024,
			wantErr: nil,
		},
		{
			name: "1gib",
			args: args{
				limit: 1024 * 1024 * 1024,
			},
			want:    1024 * 1024 * 1024,
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Limit(tt.args.limit)()
			if err != tt.wantErr {
				t.Errorf("Limit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Limit() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetGoMemLimitWithProvider(t *testing.T) {
	type args struct {
		provider Provider
		ratio    float64
	}
	tests := []struct {
		name       string
		args       args
		want       int64
		wantErr    error
		gomemlimit int64
	}{
		{
			name: "Limit_0.5",
			args: args{
				provider: Limit(1024 * 1024 * 1024),
				ratio:    0.5,
			},
			want:       536870912,
			wantErr:    nil,
			gomemlimit: 536870912,
		},
		{
			name: "Limit_0.9",
			args: args{
				provider: Limit(1024 * 1024 * 1024),
				ratio:    0.9,
			},
			want:       966367641,
			wantErr:    nil,
			gomemlimit: 966367641,
		},
		{
			name: "Limit_0.9_math.MaxUint64",
			args: args{
				provider: Limit(math.MaxUint64),
				ratio:    0.9,
			},
			want:       math.MaxInt64,
			wantErr:    nil,
			gomemlimit: math.MaxInt64,
		},
		{
			name: "Limit_0.9_math.MaxUint64",
			args: args{
				provider: Limit(math.MaxUint64),
				ratio:    0.9,
			},
			want:       math.MaxInt64,
			wantErr:    nil,
			gomemlimit: math.MaxInt64,
		},
		{
			name: "Limit_0.45_math.MaxUint64",
			args: args{
				provider: Limit(math.MaxUint64),
				ratio:    0.45,
			},
			want:       8301034833169298432,
			wantErr:    nil,
			gomemlimit: 8301034833169298432,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				debug.SetMemoryLimit(math.MaxInt64)
			})
			got, err := SetGoMemLimitWithProvider(tt.args.provider, tt.args.ratio)
			if err != tt.wantErr {
				t.Errorf("SetGoMemLimitWithProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SetGoMemLimitWithProvider() got = %v, want %v", got, tt.want)
			}
			if debug.SetMemoryLimit(-1) != tt.gomemlimit {
				t.Errorf("debug.SetMemoryLimit(-1) got = %v, want %v", debug.SetMemoryLimit(-1), tt.gomemlimit)
			}
		})
	}
}

func TestSetGoMemLimitWithOpts(t *testing.T) {
	tests := []struct {
		name       string
		opts       []Option
		want       int64
		wantErr    error
		gomemlimit int64
	}{
		{
			name: "unknown error",
			opts: []Option{
				WithProvider(func() (uint64, error) {
					return 0, fmt.Errorf("unknown error")
				}),
			},
			want:       0,
			wantErr:    fmt.Errorf("failed to set GOMEMLIMIT: unknown error"),
			gomemlimit: math.MaxInt64,
		},
		{
			name: "ErrNoLimit",
			opts: []Option{
				WithProvider(func() (uint64, error) {
					return 0, ErrNoLimit
				}),
			},
			want:       0,
			wantErr:    nil,
			gomemlimit: math.MaxInt64,
		},
		{
			name: "wrapped ErrNoLimit",
			opts: []Option{
				WithProvider(func() (uint64, error) {
					return 0, fmt.Errorf("wrapped: %w", ErrNoLimit)
				}),
			},
			want:       0,
			wantErr:    nil,
			gomemlimit: math.MaxInt64,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SetGoMemLimitWithOpts(tt.opts...)
			if tt.wantErr != nil && err.Error() != tt.wantErr.Error() {
				t.Errorf("SetGoMemLimitWithOpts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SetGoMemLimitWithOpts() got = %v, want %v", got, tt.want)
			}
			if debug.SetMemoryLimit(-1) != tt.gomemlimit {
				t.Errorf("debug.SetMemoryLimit(-1) got = %v, want %v", debug.SetMemoryLimit(-1), tt.gomemlimit)
			}
		})
	}
}

func TestSetGoMemLimitWithOpts_rollbackOnPanic(t *testing.T) {
	t.Cleanup(func() {
		debug.SetMemoryLimit(math.MaxInt64)
	})

	limit := int64(987654321)
	_ = debug.SetMemoryLimit(987654321)
	_, err := SetGoMemLimitWithOpts(
		WithProvider(func() (uint64, error) {
			debug.SetMemoryLimit(123456789)
			panic("panic")
		}),
		WithRatio(1),
	)
	if err == nil {
		t.Error("SetGoMemLimitWithOpts() error = nil, want panic")
	}

	curr := debug.SetMemoryLimit(-1)
	if curr != limit {
		t.Errorf("debug.SetMemoryLimit(-1) got = %v, want %v", curr, limit)
	}
}

func TestSetGoMemLimitWithOpts_WithRefreshInterval(t *testing.T) {
	t.Cleanup(func() {
		debug.SetMemoryLimit(math.MaxInt64)
	})

	var limit atomic.Int64
	output, err := SetGoMemLimitWithOpts(
		WithProvider(func() (uint64, error) {
			l := limit.Load()
			if l == 0 {
				return 0, ErrNoLimit
			}
			return uint64(l), nil
		}),
		WithRatio(1),
		WithRefreshInterval(10*time.Millisecond),
	)
	if err != nil {
		t.Errorf("SetGoMemLimitWithOpts() error = %v", err)
	} else if output != limit.Load() {
		t.Errorf("SetGoMemLimitWithOpts() got = %v, want %v", output, limit.Load())
	}

	// 1. no limit
	curr := debug.SetMemoryLimit(-1)
	if curr != math.MaxInt64 {
		t.Errorf("debug.SetMemoryLimit(-1) got = %v, want %v", curr, limit.Load())
	}

	// 2. max limit
	limit.Add(math.MaxInt64)
	time.Sleep(100 * time.Millisecond)

	curr = debug.SetMemoryLimit(-1)
	if curr != math.MaxInt64 {
		t.Errorf("debug.SetMemoryLimit(-1) got = %v, want %v", curr, math.MaxInt64)
	}

	// 3. adjust limit
	limit.Add(-1024)
	time.Sleep(100 * time.Millisecond)

	curr = debug.SetMemoryLimit(-1)
	if curr != math.MaxInt64-1024 {
		t.Errorf("debug.SetMemoryLimit(-1) got = %v, want %v", curr, math.MaxInt64-1024)
	}

	// 4. no limit again (don't change the limit)
	limit.Store(0)
	time.Sleep(100 * time.Millisecond)

	curr = debug.SetMemoryLimit(-1)
	if curr != math.MaxInt64-1024 {
		t.Errorf("debug.SetMemoryLimit(-1) got = %v, want %v", curr, math.MaxInt64-1024)
	}

	// 5. new limit
	limit.Store(math.MaxInt32)
	time.Sleep(100 * time.Millisecond)

	curr = debug.SetMemoryLimit(-1)
	if curr != math.MaxInt32 {
		t.Errorf("debug.SetMemoryLimit(-1) got = %v, want %v", curr, math.MaxInt32)
	}
}
