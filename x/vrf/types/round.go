package types

import "time"

// RoundAt returns the drand round that is scheduled at or before the given
// time, using the period and genesis time from params.
func RoundAt(params VrfParams, t time.Time) uint64 {
	if params.PeriodSeconds == 0 {
		return 0
	}

	genesis := time.Unix(params.GenesisUnixSec, 0).UTC()
	if t.Before(genesis) {
		return 0
	}

	dt := t.Sub(genesis)
	period := time.Duration(params.PeriodSeconds) * time.Second

	return uint64(dt/period) + 1
}
