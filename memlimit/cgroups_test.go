//go:build linux
// +build linux

package memlimit

import (
	"flag"
	"log"
	"os"
	"testing"

	"github.com/containerd/cgroups"
)

func TestMain(m *testing.M) {
	flag.Uint64Var(&expected, "expected", 0, "Expected cgroup's memory limit")
	flag.Parse()

	cgVersion = cgroups.Mode()
	log.Println("Cgroups version:", cgVersion)

	os.Exit(m.Run())
}

var (
	cgVersion cgroups.CGMode
	expected  uint64
)

func TestFromCgroup(t *testing.T) {
	limit, err := FromCgroup()
	if cgVersion == cgroups.Unavailable && err != ErrNoCgroup {
		t.Fatalf("FromCgroup() error = %v, wantErr %v", err, ErrNoCgroup)
	}

	if err != nil {
		t.Fatalf("FromCgroup() error = %v, wantErr %v", err, nil)
	}
	if limit != expected {
		t.Fatalf("FromCgroup() got = %v, want %v", limit, expected)
	}
}

func TestFromCgroupV1(t *testing.T) {
	if cgVersion != cgroups.Legacy {
		t.Skip("cgroups v1 is not supported")
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
	if cgVersion != cgroups.Hybrid && cgVersion != cgroups.Unified {
		t.Skip("cgroups v2 is not supported")
	}
	limit, err := FromCgroupV2()
	if err != nil {
		t.Fatalf("FromCgroupV2() error = %v, wantErr %v", err, nil)
	}
	if limit != expected {
		t.Fatalf("FromCgroupV2() got = %v, want %v", limit, expected)
	}
}
