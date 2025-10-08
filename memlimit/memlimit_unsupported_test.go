//go:build !linux
// +build !linux

package memlimit

import (
	"errors"
	"flag"
	"math"
	"os"
	"runtime/debug"
	"testing"
)

var expected uint64

func TestMain(m *testing.M) {
	flag.Uint64Var(&expected, "expected", 0, "Expected memory limit")
	flag.Parse()

	os.Exit(m.Run())
}

func TestSetGoMemLimit(t *testing.T) {
	type args struct {
		ratio float64
	}
	tests := []struct {
		name       string
		args       args
		want       int64
		wantErr    error
		gomemlimit int64
	}{
		{
			name: "0.5",
			args: args{
				ratio: 0.5,
			},
			want:       0,
			wantErr:    ErrCgroupsNotSupported,
			gomemlimit: math.MaxInt64,
		},
		{
			name: "0.9",
			args: args{
				ratio: 0.9,
			},
			want:       0,
			wantErr:    ErrCgroupsNotSupported,
			gomemlimit: math.MaxInt64,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				debug.SetMemoryLimit(math.MaxInt64)
			})
			got, err := Set(WithRatio(tt.args.ratio))
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Set() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetGoMemLimitWithProvider_WithCgroupProvider(t *testing.T) {
	type args struct {
		provider Provider
		ratio    float64
	}
	tests := []struct {
		name    string
		args    args
		want    int64
		wantErr error
	}{
		{
			name: "FromCgroup",
			args: args{
				provider: FromCgroup,
				ratio:    0.9,
			},
			want:    0,
			wantErr: ErrCgroupsNotSupported,
		},
		{
			name: "FromCgroupV1",
			args: args{
				provider: FromCgroupV1,
				ratio:    0.9,
			},
			want:    0,
			wantErr: ErrCgroupsNotSupported,
		},
		{
			name: "FromCgroupHybrid",
			args: args{
				provider: FromCgroupHybrid,
				ratio:    0.9,
			},
			want:    0,
			wantErr: ErrCgroupsNotSupported,
		},
		{
			name: "FromCgroupV2",
			args: args{
				provider: FromCgroupV2,
				ratio:    0.9,
			},
			want:    0,
			wantErr: ErrCgroupsNotSupported,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Set(WithProvider(tt.args.provider), WithRatio(tt.args.ratio))
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Set() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetGoMemLimitWithProvider_WithSystemProvider(t *testing.T) {
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
		skip       bool
	}{
		{
			name: "FromSystem",
			args: args{
				provider: FromSystem,
				ratio:    0.9,
			},
			want:       int64(float64(expected) * 0.9),
			wantErr:    nil,
			gomemlimit: int64(float64(expected) * 0.9),
			skip:       expected == 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skip()
			}
			t.Cleanup(func() {
				debug.SetMemoryLimit(math.MaxInt64)
			})
			got, err := Set(WithProvider(tt.args.provider), WithRatio(tt.args.ratio))
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Set() got = %v, want %v", got, tt.want)
			}
		})
	}
}
