package keeper

import (
	"fmt"
	"math/big"

	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/ethereum/go-ethereum/accounts/abi"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/vexxvakan/vrf/x/vrf/types"
)

// GetBeacon returns the VrfBeacon for the current context height
//
//   - If VrfParams.enabled == false, returns an error.
//   - If there is no VrfBeacon for the current height, returns an error.
func (k Keeper) GetBeacon(ctx sdk.Context) (types.VrfBeacon, error) {
	params, err := k.GetParams(ctx.Context())
	if err != nil {
		return types.VrfBeacon{}, err
	}

	if !params.Enabled {
		return types.VrfBeacon{}, fmt.Errorf("vrf: GetBeacon called while VRF is disabled")
	}

	beacon, err := k.GetLatestBeacon(ctx.Context())
	if err != nil {
		return types.VrfBeacon{}, err
	}

	return beacon, nil
}

// ExpandRandomness derives `count` random words from the beacon in the current
// context, mirroring the derivation used by the EVM precompile:
//
//	keccak256(abi.encode(chainHash, drandRound, randomness, userSeed, i))
//
// It returns an error if VRF is disabled or if no beacon exists for the
// current height.
func (k Keeper) ExpandRandomness(
	ctx sdk.Context,
	count uint32,
	userSeed []byte,
) (types.VrfBeacon, [][]byte, error) {
	if count == 0 {
		return types.VrfBeacon{}, nil, fmt.Errorf("vrf: ExpandRandomness requires count > 0")
	}

	beacon, err := k.GetBeacon(ctx)
	if err != nil {
		return types.VrfBeacon{}, nil, err
	}

	params, err := k.GetParams(ctx.Context())
	if err != nil {
		return types.VrfBeacon{}, nil, err
	}

	words, err := deriveRandomWords(params, beacon, count, userSeed)
	if err != nil {
		return types.VrfBeacon{}, nil, err
	}

	return beacon, words, nil
}

// deriveRandomWords implements the Solidity-side derivation using go-ethereum's
// ABI encoder to match:
//
//	abi.encode(chainHash, drandRound, randomness, userSeed, i)
//
// and keccak256 hashing.
func deriveRandomWords(
	params types.VrfParams,
	beacon types.VrfBeacon,
	count uint32,
	userSeed []byte,
) ([][]byte, error) {
	if count == 0 {
		return nil, fmt.Errorf("vrf: deriveRandomWords requires count > 0")
	}

	// Define ABI argument types corresponding to:
	//   bytes chainHash, uint64 drandRound, bytes32 randomness,
	//   bytes32 userSeed, uint256 i
	bytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "vrf: failed to construct abi bytes type")
	}
	uint64Type, err := abi.NewType("uint64", "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "vrf: failed to construct abi uint64 type")
	}
	bytes32Type, err := abi.NewType("bytes32", "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "vrf: failed to construct abi bytes32 type")
	}
	uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "vrf: failed to construct abi uint256 type")
	}

	args := abi.Arguments{
		{Type: bytesType},
		{Type: uint64Type},
		{Type: bytes32Type},
		{Type: bytes32Type},
		{Type: uint256Type},
	}

	seedBytes := toBytes32(beacon.Randomness)
	userSeedBytes := toBytes32(userSeed)

	out := make([][]byte, 0, count)

	for i := uint32(0); i < count; i++ {
		packed, err := args.Pack(
			params.ChainHash,
			beacon.DrandRound,
			seedBytes,
			userSeedBytes,
			new(big.Int).SetUint64(uint64(i)),
		)
		if err != nil {
			return nil, errors.Wrap(err, "vrf: failed to ABI-pack randomness inputs")
		}

		hash := gethcrypto.Keccak256(packed)
		out = append(out, hash)
	}

	return out, nil
}

// toBytes32 converts an arbitrary-length byte slice into a 32-byte array,
// zero-padding or truncating as needed.
func toBytes32(bz []byte) [32]byte {
	var out [32]byte
	if len(bz) == 0 {
		return out
	}

	// If longer than 32, truncate from the left (keep the rightmost bytes).
	if len(bz) > 32 {
		bz = bz[len(bz)-32:]
	}

	copy(out[:], bz)
	return out
}
