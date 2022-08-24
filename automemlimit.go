package automemlimit

import (
	"io"
	"log"
	"os"
	"strconv"

	"github.com/KimMachineGun/automemlimit/memlimit"
)

const (
	EnvGOMEMLIMIT         = "GOMEMLIMIT"
	EnvAUTOMEMLIMIT       = "AUTOMEMLIMIT"
	EnvAUTOMEMLIMIT_DEBUG = "AUTOMEMLIMIT_DEBUG"

	DefaultAUTOMEMLIMIT = 0.9
)

var logger = log.New(io.Discard, "", log.LstdFlags)

func init() {
	if os.Getenv(EnvAUTOMEMLIMIT_DEBUG) == "true" {
		logger = log.Default()
	}

	if val, ok := os.LookupEnv(EnvGOMEMLIMIT); ok {
		logger.Printf("GOMEMLIMIT is set already, skipping: %s\n", val)
		return
	}

	ratio := DefaultAUTOMEMLIMIT
	if val, ok := os.LookupEnv(EnvAUTOMEMLIMIT); ok {
		if val == "off" {
			logger.Printf("AUTOMEMLIMIT is set to off, skipping\n")
			return
		}
		_ratio, err := strconv.ParseFloat(val, 64)
		if err != nil {
			logger.Printf("cannot parse AUTOMEMLIMIT: %s\n", val)
			return
		}
		ratio = _ratio
	}
	if ratio <= 0 || ratio > 1 {
		logger.Printf("invalid AUTOMEMLIMIT: %f\n", ratio)
		return
	}

	limit, err := memlimit.SetGoMemLimit(ratio)
	if err != nil {
		logger.Printf("failed to set GOMEMLIMIT: %v\n", err)
		return
	}

	logger.Printf("GOMEMLIMIT=%d\n", limit)
}
