package keeper

import (
	"context"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

var _ stakingtypes.StakingHooks = Hooks{}

// Hooks implements x/staking hooks for cleaning up vrf-related state.
type Hooks struct {
	k Keeper
}

func (k Keeper) Hooks() stakingtypes.StakingHooks {
	return Hooks{k: k}
}

func (Hooks) AfterValidatorCreated(context.Context, sdk.ValAddress) error { return nil }

func (Hooks) BeforeValidatorModified(context.Context, sdk.ValAddress) error { return nil }

func (h Hooks) AfterValidatorRemoved(ctx context.Context, _ sdk.ConsAddress, valAddr sdk.ValAddress) error {
	// Best-effort cleanup: remove any identity binding for the removed validator.
	return h.k.RemoveVrfIdentity(ctx, valAddr.String())
}

func (Hooks) AfterValidatorBonded(context.Context, sdk.ConsAddress, sdk.ValAddress) error { return nil }

func (Hooks) AfterValidatorBeginUnbonding(context.Context, sdk.ConsAddress, sdk.ValAddress) error {
	return nil
}

func (Hooks) BeforeDelegationCreated(context.Context, sdk.AccAddress, sdk.ValAddress) error {
	return nil
}

func (Hooks) BeforeDelegationSharesModified(context.Context, sdk.AccAddress, sdk.ValAddress) error {
	return nil
}

func (Hooks) BeforeDelegationRemoved(context.Context, sdk.AccAddress, sdk.ValAddress) error {
	return nil
}

func (Hooks) AfterDelegationModified(context.Context, sdk.AccAddress, sdk.ValAddress) error {
	return nil
}

func (Hooks) BeforeValidatorSlashed(context.Context, sdk.ValAddress, math.LegacyDec) error {
	return nil
}

func (Hooks) AfterUnbondingInitiated(context.Context, uint64) error { return nil }
