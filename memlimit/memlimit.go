package memlimit

import (
	"errors"
	"runtime/debug"
)

var (
	ErrNoLimit             = errors.New("memory is not limited")
	ErrNoCgroup            = errors.New("process is not in cgroup")
	ErrCgroupsNotSupported = errors.New("cgroups is not supported on this system")
)

type Provider func() (uint64, error)

func SetGoMemLimit(ratio float64) (int64, error) {
	return SetGoMemLimitWithProvider(FromCgroup, ratio)
}

func SetGoMemLimitWithProvider(provider Provider, ratio float64) (int64, error) {
	limit, err := provider()
	if err != nil {
		return 0, err
	}
	goMemLimit := int64(ratio * float64(limit))
	debug.SetMemoryLimit(goMemLimit)
	return goMemLimit, nil
}

func Limit(limit uint64) func() (uint64, error) {
	return func() (uint64, error) {
		return limit, nil
	}
}
