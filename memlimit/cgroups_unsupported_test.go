//go:build !linux

package memlimit

import (
	"errors"
	"testing"
)

func TestFromCgroup(t *testing.T) {
	limit, err := FromCgroup()
	if !errors.Is(err, ErrCgroupsNotSupported) {
		t.Fatalf("FromCgroup() error = %v, want %v", err, ErrCgroupsNotSupported)
	}

	if limit != 0 {
		t.Fatalf("FromCgroup() = %v, want %v", limit, 0)
	}
}
