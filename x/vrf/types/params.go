package types

import (
	"fmt"
)

// VrfParams mirrors the PRD definition and contains all cryptographic and timing
// context needed to verify drand beacons on-chain and map block time to drand
// rounds.
func DefaultParams() VrfParams {
	return VrfParams{
		PeriodSeconds:       30,
		SafetyMarginSeconds: 30,
	}
}

func (p VrfParams) Validate() error {
	if p.PeriodSeconds == 0 {
		return fmt.Errorf("period_seconds must be positive")
	}

	if p.SafetyMarginSeconds < p.PeriodSeconds {
		return fmt.Errorf("safety_margin_seconds (%d) must be >= period_seconds (%d)", p.SafetyMarginSeconds, p.PeriodSeconds)
	}

	if p.Enabled {
		if len(p.PublicKey) == 0 {
			return fmt.Errorf("public_key must not be empty when enabled")
		}

		if len(p.ChainHash) == 0 {
			return fmt.Errorf("chain_hash must not be empty when enabled")
		}
	}

	return nil
}
