package types

import (
	"bytes"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	for _, e := range gs.Committee {
		if e.Address == "" {
			return fmt.Errorf("committee address must not be empty")
		}
		if _, err := sdk.AccAddressFromBech32(e.Address); err != nil {
			return fmt.Errorf("committee address is invalid: %w", err)
		}
	}

	for _, i := range gs.Identities {
		if i.ValidatorAddress == "" {
			return fmt.Errorf("identities validator_address must not be empty")
		}
		if _, err := sdk.ValAddressFromBech32(i.ValidatorAddress); err != nil {
			return fmt.Errorf("identities validator_address is invalid: %w", err)
		}

		if len(i.DrandBlsPublicKey) == 0 {
			return fmt.Errorf("identities drand_bls_public_key must not be empty")
		}

		// When params.chain_hash is set, enforce consistency for pre-seeded identities.
		if len(gs.Params.ChainHash) > 0 && len(i.ChainHash) > 0 && !bytes.Equal(gs.Params.ChainHash, i.ChainHash) {
			return fmt.Errorf("identities chain_hash must match params.chain_hash")
		}
	}

	return nil
}

func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
	}
}
