//go:build !linux
// +build !linux

package memlimit

import (
	"testing"
)

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
				provider: fromCgroupHybrid,
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
