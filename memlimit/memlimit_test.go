package memlimit

import (
	"context"
	"fmt"
	"math"
	"os"
	"runtime/debug"
	"strings"
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
			got, err := Set(WithProvider(tt.args.provider), WithRatio(tt.args.ratio))
			if err != tt.wantErr {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Set() got = %v, want %v", got, tt.want)
			}
			if debug.SetMemoryLimit(-1) != tt.gomemlimit {
				t.Errorf("debug.SetMemoryLimit(-1) got = %v, want %v", debug.SetMemoryLimit(-1), tt.gomemlimit)
			}
		})
	}
}

func TestSet_WithSystemFallback(t *testing.T) {
	t.Cleanup(func() {
		debug.SetMemoryLimit(math.MaxInt64)
	})

	// Test that manual fallback to system memory works when provider fails
	got, err := Set(
		WithProvider(
			ApplyFallback(
				func() (uint64, error) {
					return 0, fmt.Errorf("provider error")
				},
				FromSystem,
			),
		),
		WithRatio(0.9),
	)
	if err != nil {
		t.Errorf("Set() error = %v, expected nil due to system fallback", err)
	}
	// Should have set some limit from system memory
	if got == 0 {
		t.Skip("System memory not available")
	}
	if debug.SetMemoryLimit(-1) <= 0 {
		t.Errorf("Expected memory limit to be set from system fallback")
	}
}

func TestSet_rollbackOnPanic(t *testing.T) {
	// Ensure GOMEMLIMIT is not set
	os.Unsetenv("GOMEMLIMIT")

	t.Cleanup(func() {
		debug.SetMemoryLimit(math.MaxInt64)
	})

	limit := int64(987654321)
	_ = debug.SetMemoryLimit(987654321)
	_, err := Set(
		WithProvider(func() (uint64, error) {
			t.Log("Provider called, about to panic")
			debug.SetMemoryLimit(123456789)
			panic("test panic")
		}),
		WithRatio(1),
	)
	if err == nil || !strings.Contains(err.Error(), "panic") {
		t.Errorf("Set() error = %v, want error containing 'panic'", err)
	}

	curr := debug.SetMemoryLimit(-1)
	if curr != limit {
		t.Errorf("debug.SetMemoryLimit(-1) got = %v, want %v", curr, limit)
	}
}

func TestSet_WithRefreshInterval(t *testing.T) {
	t.Cleanup(func() {
		debug.SetMemoryLimit(math.MaxInt64)
	})

	// Test that refresh context controls the goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var refreshCount atomic.Int32
	_, err := Set(
		WithProvider(func() (uint64, error) {
			refreshCount.Add(1)
			return 1024 * 1024 * 1024, nil
		}),
		WithRefreshInterval(ctx, 50*time.Millisecond),
	)
	if err != nil {
		t.Errorf("Set() error = %v", err)
	}

	// Wait for a few refresh cycles
	time.Sleep(200 * time.Millisecond)

	// Cancel the refresh context
	cancel()

	countBeforeCancel := refreshCount.Load()

	// Wait a bit more
	time.Sleep(200 * time.Millisecond)

	// Refresh count should not increase after cancel
	if refreshCount.Load() > countBeforeCancel {
		t.Errorf("Refresh continued after context cancellation")
	}
}
