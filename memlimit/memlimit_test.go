//go:build linux
// +build linux

package memlimit

import (
	"flag"
	"log"
	"math"
	"os"
	"runtime/debug"
	"testing"

	"github.com/containerd/cgroups/v3"
)

var (
	cgVersion      cgroups.CGMode
	expected       uint64
	expectedSystem uint64
)

func TestMain(m *testing.M) {
	flag.Uint64Var(&expected, "expected", 0, "Expected cgroup's memory limit")
	flag.Uint64Var(&expectedSystem, "expected-system", 0, "Expected system memory limit")
	flag.Parse()

	cgVersion = cgroups.Mode()
	log.Println("Cgroups version:", cgVersion)

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
		skip       bool
	}{
		{
			name: "0.5",
			args: args{
				ratio: 0.5,
			},
			want:       int64(float64(expected) * 0.5),
			wantErr:    nil,
			gomemlimit: int64(float64(expected) * 0.5),
			skip:       expected == 0 || cgVersion == cgroups.Unavailable,
		},
		{
			name: "0.9",
			args: args{
				ratio: 0.9,
			},
			want:       int64(float64(expected) * 0.9),
			wantErr:    nil,
			gomemlimit: int64(float64(expected) * 0.9),
			skip:       expected == 0 || cgVersion == cgroups.Unavailable,
		},
		{
			name: "Unavailable",
			args: args{
				ratio: 0.9,
			},
			want:       0,
			wantErr:    ErrCgroupsNotSupported,
			gomemlimit: math.MaxInt64,
			skip:       cgVersion != cgroups.Unavailable,
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
			got, err := SetGoMemLimit(tt.args.ratio)
			if err != tt.wantErr {
				t.Errorf("SetGoMemLimit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SetGoMemLimit() got = %v, want %v", got, tt.want)
			}
			if debug.SetMemoryLimit(-1) != tt.gomemlimit {
				t.Errorf("debug.SetMemoryLimit(-1) got = %v, want %v", debug.SetMemoryLimit(-1), tt.gomemlimit)
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
		name       string
		args       args
		want       int64
		wantErr    error
		gomemlimit int64
		skip       bool
	}{
		{
			name: "FromCgroup",
			args: args{
				provider: FromCgroup,
				ratio:    0.9,
			},
			want:       int64(float64(expected) * 0.9),
			wantErr:    nil,
			gomemlimit: int64(float64(expected) * 0.9),
			skip:       expected == 0 || cgVersion == cgroups.Unavailable,
		},
		{
			name: "FromCgroup_Unavaliable",
			args: args{
				provider: FromCgroup,
				ratio:    0.9,
			},
			want:       0,
			wantErr:    ErrNoCgroup,
			gomemlimit: math.MaxInt64,
			skip:       expected == 0 || cgVersion != cgroups.Unavailable,
		},
		{
			name: "FromCgroupV1",
			args: args{
				provider: FromCgroupV1,
				ratio:    0.9,
			},
			want:       int64(float64(expected) * 0.9),
			wantErr:    nil,
			gomemlimit: int64(float64(expected) * 0.9),
			skip:       expected == 0 || cgVersion != cgroups.Legacy,
		},
		{
			name: "FromCgroupHybrid",
			args: args{
				provider: FromCgroupHybrid,
				ratio:    0.9,
			},
			want:       int64(float64(expected) * 0.9),
			wantErr:    nil,
			gomemlimit: int64(float64(expected) * 0.9),
			skip:       expected == 0 || cgVersion != cgroups.Hybrid,
		},
		{
			name: "FromCgroupV2",
			args: args{
				provider: FromCgroupV2,
				ratio:    0.9,
			},
			want:       int64(float64(expected) * 0.9),
			wantErr:    nil,
			gomemlimit: int64(float64(expected) * 0.9),
			skip:       expected == 0 || cgVersion != cgroups.Unified,
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
			want:       int64(float64(expectedSystem) * 0.9),
			wantErr:    nil,
			gomemlimit: int64(float64(expectedSystem) * 0.9),
			skip:       expectedSystem == 0,
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
