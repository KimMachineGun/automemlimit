//go:build linux
// +build linux

package memlimit

import (
	"testing"
)

func TestFromCgroup(t *testing.T) {
	if expected == 0 {
		t.Skip()
	}

	limit, err := FromCgroup()
	if cgVersion == 0 && err != ErrNoCgroup {
		t.Fatalf("FromCgroup() error = %v, wantErr %v", err, ErrNoCgroup)
	}

	if err != nil {
		t.Fatalf("FromCgroup() error = %v, wantErr %v", err, nil)
	}
	if limit != expected {
		t.Fatalf("FromCgroup() got = %v, want %v", limit, expected)
	}
}

func TestFromCgroupHybrid(t *testing.T) {
	if expected == 0 {
		t.Skip()
	}

	limit, err := FromCgroupHybrid()
	if cgVersion == 0 && err != ErrNoCgroup {
		t.Fatalf("FromCgroupHybrid() error = %v, wantErr %v", err, ErrNoCgroup)
	}

	if err != nil {
		t.Fatalf("FromCgroupHybrid() error = %v, wantErr %v", err, nil)
	}
	if limit != expected {
		t.Fatalf("FromCgroupHybrid() got = %v, want %v", limit, expected)
	}
}

func TestFromCgroupV1(t *testing.T) {
	if expected == 0 || cgVersion != 1 {
		t.Skip()
	}
	limit, err := FromCgroupV1()
	if err != nil {
		t.Fatalf("FromCgroupV1() error = %v, wantErr %v", err, nil)
	}
	if limit != expected {
		t.Fatalf("FromCgroupV1() got = %v, want %v", limit, expected)
	}
}

func TestFromCgroupV2(t *testing.T) {
	if expected == 0 || cgVersion != 2 {
		t.Skip()
	}
	limit, err := FromCgroupV2()
	if err != nil {
		t.Fatalf("FromCgroupV2() error = %v, wantErr %v", err, nil)
	}
	if limit != expected {
		t.Fatalf("FromCgroupV2() got = %v, want %v", limit, expected)
	}
}
