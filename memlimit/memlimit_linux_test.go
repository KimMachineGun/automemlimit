//go:build linux

package memlimit

import (
	"flag"
	"math"
	"os"
	"runtime/debug"
	"testing"
)

var (
	expected       uint64
	expectedSystem uint64
)

func TestMain(m *testing.M) {
	flag.Uint64Var(&expected, "expected", 0, "Expected cgroup memory limit")
	flag.Uint64Var(&expectedSystem, "expected-system", 0, "Expected system memory limit")
	flag.Parse()

	os.Unsetenv("GOMEMLIMIT")
	os.Unsetenv("AUTOMEMLIMIT")

	os.Exit(m.Run())
}

func TestSetLinux(t *testing.T) {
	if expected == 0 {
		t.Skip()
	}

	t.Cleanup(func() {
		debug.SetMemoryLimit(math.MaxInt64)
	})
	debug.SetMemoryLimit(math.MaxInt64)

	want := int64(float64(expected) * 0.9)
	got, err := Set(WithRatio(0.9))
	if err != nil {
		t.Fatalf("Set() error = %v, want nil", err)
	}
	if got != want {
		t.Fatalf("Set() = %v, want %v", got, want)
	}
	if actual := debug.SetMemoryLimit(-1); actual != want {
		t.Fatalf("GOMEMLIMIT = %v, want %v", actual, want)
	}
}

func TestSetWithSystemFallbackLinux(t *testing.T) {
	if expectedSystem == 0 || expected != 0 {
		t.Skip()
	}

	t.Cleanup(func() {
		debug.SetMemoryLimit(math.MaxInt64)
	})
	debug.SetMemoryLimit(math.MaxInt64)

	want := int64(float64(expectedSystem) * 0.9)

	got, err := Set(
		WithProvider(
			ApplyFallback(
				FromCgroup,
				FromSystem,
			),
		),
		WithRatio(0.9),
	)
	if err != nil {
		t.Fatalf("Set() error = %v, want nil", err)
	}

	if got != want {
		t.Fatalf("Set() = %v, want %v", got, want)
	}
	if actual := debug.SetMemoryLimit(-1); actual != want {
		t.Fatalf("GOMEMLIMIT = %v, want %v", actual, want)
	}
}
