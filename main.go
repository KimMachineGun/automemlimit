package automemlimit

import (
	"io"
	"log"
	"os"
	"strconv"

	"github.com/KimMachineGun/automemlimit/memlimit"
)

const (
	EnvGOMEMLIMIT   = "GOMEMLIMIT"
	EnvAUTOMEMLIMIT = "AUTOMEMLIMIT"

	DefaultAUTOMEMLIMIT = 0.9
)

var logger = log.New(io.Discard, "", log.LstdFlags)

func init() {
	_, ok := os.LookupEnv(EnvGOMEMLIMIT)
	if ok {
		return
	}

	ratio := DefaultAUTOMEMLIMIT
	envAutoMemLimit, ok := os.LookupEnv(EnvAUTOMEMLIMIT)
	if ok {
		_ratio, err := strconv.ParseFloat(envAutoMemLimit, 64)
		if err != nil {
			logger.Printf("cannot parse AUTOMEMLIMIT: %s", envAutoMemLimit)
			return
		}
		ratio = _ratio
	}

	limit, err := memlimit.SetGoMemLimit(ratio)
	if err != nil {
		logger.Printf("failed to set GOMEMLIMIT: %v", err)
		return
	}

	logger.Printf("GOMEMLIMIT=%d", limit)
}
