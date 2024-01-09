package memlimit

import (
	"fmt"
	"os"
	"reflect"
	"testing"
)

func TestParseExperiments(t *testing.T) {
	tests := []struct {
		name    string
		env     string
		want    Experiments
		wantErr error
	}{
		{
			name: "empty",
			env:  "",
			want: Experiments{},
		},
		{
			name:    "unknown",
			env:     "unknown",
			want:    Experiments{},
			wantErr: fmt.Errorf("unknown AUTOMEMLIMIT_EXPERIMENT unknown"),
		},
		{
			name: "none",
			env:  "none",
			want: Experiments{},
		},
		{
			name: "none - with other",
			env:  "system,none",
			want: Experiments{},
		},
		{
			name: "system",
			env:  "system",
			want: Experiments{
				System: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exp, ok := os.LookupEnv(envAUTOMEMLIMIT_EXPERIMENT)
			t.Cleanup(func() {
				if ok {
					os.Setenv(envAUTOMEMLIMIT_EXPERIMENT, exp)
				} else {
					os.Unsetenv(envAUTOMEMLIMIT_EXPERIMENT)
				}
			})

			os.Setenv("AUTOMEMLIMIT_EXPERIMENT", tt.env)
			exps, err := parseExperiments()
			if !reflect.DeepEqual(exps, tt.want) {
				t.Errorf("experiments= %#v, want %#v", exps, tt.want)
			}
			if !reflect.DeepEqual(err, tt.wantErr) {
				t.Errorf("err = %#v, want %#v", err, tt.wantErr)
			}
		})
	}
}
