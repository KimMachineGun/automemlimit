//go:build linux
// +build linux

package memlimit

import (
	"flag"
	"log"
	"os"
	"testing"

	"github.com/containerd/cgroups/v3"
)

var (
	cgVersion cgroups.CGMode
	expected  uint64
)

func TestMain(m *testing.M) {
	flag.Uint64Var(&expected, "expected", 0, "Expected cgroup's memory limit")
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
		name    string
		args    args
		want    int64
		wantErr error
		skip    bool
	}{
		{
			name: "0.5",
			args: args{
				ratio: 0.5,
			},
			want:    int64(float64(expected) * 0.5),
			wantErr: nil,
			skip:    cgVersion == cgroups.Unavailable,
		},
		{
			name: "0.9",
			args: args{
				ratio: 0.9,
			},
			want:    int64(float64(expected) * 0.9),
			wantErr: nil,
			skip:    cgVersion == cgroups.Unavailable,
		},
		{
			name: "Unavailable",
			args: args{
				ratio: 0.9,
			},
			want:    0,
			wantErr: ErrCgroupsNotSupported,
			skip:    cgVersion != cgroups.Unavailable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skip()
			}
			got, err := SetGoMemLimit(tt.args.ratio)
			if err != tt.wantErr {
				t.Errorf("SetGoMemLimit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SetGoMemLimit() got = %v, want %v", got, tt.want)
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
		skip    bool
	}{
		{
			name: "FromCgroup",
			args: args{
				provider: FromCgroup,
				ratio:    0.9,
			},
			want:    int64(float64(expected) * 0.9),
			wantErr: nil,
			skip:    cgVersion == cgroups.Unavailable,
		},
		{
			name: "FromCgroup_Unavaliable",
			args: args{
				provider: FromCgroup,
				ratio:    0.9,
			},
			want:    0,
			wantErr: ErrNoCgroup,
			skip:    cgVersion != cgroups.Unavailable,
		},
		{
			name: "FromCgroupV1",
			args: args{
				provider: FromCgroupV1,
				ratio:    0.9,
			},
			want:    int64(float64(expected) * 0.9),
			wantErr: nil,
			skip:    cgVersion != cgroups.Legacy,
		},
		{
			name: "FromCgroupV2",
			args: args{
				provider: FromCgroupV2,
				ratio:    0.9,
			},
			want:    int64(float64(expected) * 0.9),
			wantErr: nil,
			skip:    cgVersion != cgroups.Hybrid && cgVersion != cgroups.Unified,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skip()
			}
			got, err := SetGoMemLimitWithProvider(tt.args.provider, tt.args.ratio)
			if err != tt.wantErr {
				t.Errorf("SetGoMemLimitWithProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SetGoMemLimitWithProvider() got = %v, want %v", got, tt.want)
			}
		})
	}
}
