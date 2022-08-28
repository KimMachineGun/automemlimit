package memlimit

import (
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
