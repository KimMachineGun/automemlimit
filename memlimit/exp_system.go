package memlimit

import (
	"github.com/pbnjay/memory"
)

func fromSystem() (uint64, error) {
	limit := memory.TotalMemory()
	if limit == 0 {
		return 0, ErrNoLimit
	}
	return limit, nil
}
