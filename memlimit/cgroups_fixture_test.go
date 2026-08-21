package memlimit

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fixtureRootPlaceholder = "{{ROOT}}"

func escapeMountInfoPath(path string) string {
	return strings.NewReplacer(
		`\`, `\134`,
		" ", `\040`,
		"\t", `\011`,
		"\n", `\012`,
	).Replace(path)
}

func buildCgroupFixturePaths(t *testing.T, name string) (string, string) {
	t.Helper()

	root, err := filepath.Abs(filepath.Join("testdata", "cgroups", name))
	if err != nil {
		t.Fatalf("Abs() error = %v, want nil", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "mountinfo"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v, want nil", err)
	}

	mountInfoPath := filepath.Join(t.TempDir(), "mountinfo")
	data = []byte(strings.ReplaceAll(string(data), fixtureRootPlaceholder, escapeMountInfoPath(root)))
	if err := os.WriteFile(mountInfoPath, data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v, want nil", err)
	}

	return mountInfoPath, filepath.Join(root, "cgroup")
}

func TestEscapeMountInfoPath(t *testing.T) {
	path := "/root dir\ttab\nline\\040"
	want := `/root\040dir\011tab\012line\134040`
	if got := escapeMountInfoPath(path); got != want {
		t.Fatalf("escapeMountInfoPath() = %q, want %q", got, want)
	}
}

func TestFromCgroupWithFixtures(t *testing.T) {
	tests := []struct {
		fixture   string
		want      uint64
		wantErrIs error
		wantErr   string
	}{
		{
			fixture: "v2_simple_limit",
			want:    2097152,
		},
		{
			fixture: "v2_parent_limit",
			want:    536870912,
		},
		{
			fixture:   "v2_missing_memory_max",
			wantErrIs: ErrNoLimit,
		},
		{
			fixture: "v2_invalid_memory_max",
			wantErr: "failed to parse memory.max value: strconv.ParseUint: parsing \"invalid\": invalid syntax",
		},
		{
			fixture: "v1_conflicting_mounts",
			wantErr: "conflicting memory limits from cgroup mount candidates: 1024 and 2048",
		},
	}

	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			mountInfoPath, cgroupPath := buildCgroupFixturePaths(t, tt.fixture)

			got, err := fromCgroup(mountInfoPath, cgroupPath)
			if tt.wantErrIs == nil && tt.wantErr == "" {
				if err != nil {
					t.Fatalf("fromCgroup() error = %v, want nil", err)
				}
			} else if tt.wantErrIs != nil {
				if !errors.Is(err, tt.wantErrIs) {
					t.Fatalf("fromCgroup() error = %v, want %v", err, tt.wantErrIs)
				}
			} else if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("fromCgroup() error = %v, want error %q", err, tt.wantErr)
			}

			if got != tt.want {
				t.Fatalf("fromCgroup() = %v, want %v", got, tt.want)
			}
		})
	}
}
