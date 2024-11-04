package memlimit

import (
	"fmt"
	"math"
	"runtime/debug"
	"testing"
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
