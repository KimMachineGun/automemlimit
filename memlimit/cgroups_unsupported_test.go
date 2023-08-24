//go:build !linux
// +build !linux

package memlimit

import (
	"testing"
)

func TestFromCgroup(t *testing.T) {
	limit, err := FromCgroup()
	if err != ErrCgroupsNotSupported {
		t.Fatalf("FromCgroup() error = %v, wantErr %v", err, ErrCgroupsNotSupported)
	}
	if limit != 0 {
		t.Fatalf("FromCgroup() got = %v, want %v", limit, 0)
	}
}

func TestFromCgroupV1(t *testing.T) {
	limit, err := FromCgroupV1()
	if err != ErrCgroupsNotSupported {
		t.Fatalf("FromCgroupV1() error = %v, wantErr %v", err, ErrCgroupsNotSupported)
	}
	if limit != 0 {
		t.Fatalf("FromCgroupV1() got = %v, want %v", limit, 0)
	}
}

func TestFromCgroupHybrid(t *testing.T) {
	limit, err := FromCgroupHybrid()
	if err != ErrCgroupsNotSupported {
		t.Fatalf("FromCgroupHybrid() error = %v, wantErr %v", err, ErrCgroupsNotSupported)
	}
	if limit != 0 {
		t.Fatalf("FromCgroupHybrid() got = %v, want %v", limit, 0)
	}
}

func TestFromCgroupV2(t *testing.T) {
	limit, err := FromCgroupV2()
	if err != ErrCgroupsNotSupported {
		t.Fatalf("FromCgroupV2() error = %v, wantErr %v", err, ErrCgroupsNotSupported)
	}
	if limit != 0 {
		t.Fatalf("FromCgroupV2() got = %v, want %v", limit, 0)
	}
}
