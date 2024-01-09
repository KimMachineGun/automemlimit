//go:build !linux
// +build !linux

package memlimit

import (
	"errors"
	"flag"
	"os"
	"testing"
)

var (
	expected uint64
)

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
		name    string
		args    args
		want    int64
		wantErr error
	}{
		{
			name: "0.5",
			args: args{
				ratio: 0.5,
			},
			want:    0,
			wantErr: ErrCgroupsNotSupported,
		},
		{
			name: "0.9",
			args: args{
				ratio: 0.9,
			},
			want:    0,
			wantErr: ErrCgroupsNotSupported,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SetGoMemLimit(tt.args.ratio)
			if !errors.Is(err, tt.wantErr) {
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
			got, err := SetGoMemLimitWithProvider(tt.args.provider, tt.args.ratio)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("SetGoMemLimitWithProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SetGoMemLimitWithProvider() got = %v, want %v", got, tt.want)
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
		name    string
		args    args
		want    int64
		wantErr error
		skip    bool
	}{
		{
			name: "FromSystem",
			args: args{
				provider: FromSystem,
				ratio:    0.9,
			},
			want:    int64(float64(expected) * 0.9),
			wantErr: nil,
			skip:    expected == 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skip()
			}
			got, err := SetGoMemLimitWithProvider(tt.args.provider, tt.args.ratio)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("SetGoMemLimitWithProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SetGoMemLimitWithProvider() got = %v, want %v", got, tt.want)
			}
		})
	}
}
