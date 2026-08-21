package memlimit

import (
	"errors"
	"math"
	"testing"
)

func TestApplyRatio(t *testing.T) {
	tests := []struct {
		name    string
		limit   uint64
		ratio   float64
		want    uint64
		wantErr string
	}{
		{
			name:  "0.9",
			limit: 1000,
			ratio: 0.9,
			want:  900,
		},
		{
			name:  "1.0",
			limit: 1000,
			ratio: 1.0,
			want:  1000,
		},
		{
			name:  "truncate_to_zero",
			limit: 1,
			ratio: 0.9,
			want:  0,
		},
		{
			name:    "zero",
			limit:   1000,
			ratio:   0,
			wantErr: "invalid ratio: 0.000000, ratio should be in the range (0.0,1.0]",
		},
		{
			name:    "over_1",
			limit:   1000,
			ratio:   1.5,
			wantErr: "invalid ratio: 1.500000, ratio should be in the range (0.0,1.0]",
		},
		{
			name:    "nan",
			limit:   1000,
			ratio:   math.NaN(),
			wantErr: "invalid ratio: NaN, ratio should be in the range (0.0,1.0]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ApplyRatio(Limit(tt.limit), tt.ratio)()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("ApplyRatio() error = %v, want nil", err)
				}
			} else if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("ApplyRatio() error = %v, want error %q", err, tt.wantErr)
			}

			if got != tt.want {
				t.Fatalf("ApplyRatio() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplyFallback(t *testing.T) {
	errPrimary := errors.New("primary error")
	errFallback := errors.New("fallback error")

	tests := []struct {
		name      string
		primary   Provider
		fallback  Provider
		want      uint64
		wantErrIs error
	}{
		{
			name: "primary_success",
			primary: func() (uint64, error) {
				return 1000, nil
			},
			fallback: func() (uint64, error) {
				t.Fatal("fallback should not be called when primary succeeds")
				return 2000, nil
			},
			want: 1000,
		},
		{
			name: "primary_fail_fallback_success",
			primary: func() (uint64, error) {
				return 0, errPrimary
			},
			fallback: func() (uint64, error) {
				return 2000, nil
			},
			want: 2000,
		},
		{
			name: "primary_err_no_limit_fallback_success",
			primary: func() (uint64, error) {
				return 0, ErrNoLimit
			},
			fallback: func() (uint64, error) {
				return 2000, nil
			},
			want: 2000,
		},
		{
			name: "both_fail",
			primary: func() (uint64, error) {
				return 0, errPrimary
			},
			fallback: func() (uint64, error) {
				return 0, errFallback
			},
			want:      0,
			wantErrIs: errFallback,
		},
		{
			name: "primary_zero_no_error",
			primary: func() (uint64, error) {
				return 0, nil
			},
			fallback: func() (uint64, error) {
				t.Fatal("fallback should not be called when primary returns zero without error")
				return 2000, nil
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ApplyFallback(tt.primary, tt.fallback)()
			if tt.wantErrIs == nil {
				if err != nil {
					t.Fatalf("ApplyFallback() error = %v, want nil", err)
				}
			} else if !errors.Is(err, tt.wantErrIs) {
				t.Fatalf("ApplyFallback() error = %v, want %v", err, tt.wantErrIs)
			}

			if got != tt.want {
				t.Fatalf("ApplyFallback() = %v, want %v", got, tt.want)
			}
		})
	}
}
