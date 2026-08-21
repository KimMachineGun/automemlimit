//go:build !linux

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

	os.Unsetenv("GOMEMLIMIT")
	os.Unsetenv("AUTOMEMLIMIT")

	os.Exit(m.Run())
}

func TestSetUnsupported(t *testing.T) {
	t.Cleanup(func() {
		debug.SetMemoryLimit(math.MaxInt64)
	})
	debug.SetMemoryLimit(math.MaxInt64)

	got, err := Set(WithRatio(0.9))
	if !errors.Is(err, ErrCgroupsNotSupported) {
		t.Fatalf("Set() error = %v, want %v", err, ErrCgroupsNotSupported)
	}
	if got != math.MaxInt64 {
		t.Fatalf("Set() = %v, want %v", got, math.MaxInt64)
	}
	if actual := debug.SetMemoryLimit(-1); actual != math.MaxInt64 {
		t.Fatalf("GOMEMLIMIT = %v, want %v", actual, math.MaxInt64)
	}
}

func TestSetWithSystemProvider(t *testing.T) {
	if expected == 0 {
		t.Skip()
	}

	t.Cleanup(func() {
		debug.SetMemoryLimit(math.MaxInt64)
	})
	debug.SetMemoryLimit(math.MaxInt64)

	want := int64(float64(expected) * 0.9)
	got, err := Set(WithProvider(FromSystem), WithRatio(0.9))
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
