package vrf

import (
	"embed"
	"fmt"
	"math/big"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	pcommon "github.com/vexxvakan/vrf/precompile/common"
	vrfkeeper "github.com/vexxvakan/vrf/x/vrf/keeper"
)

var _ vm.PrecompiledContract = (*Precompile)(nil)

var (
	//go:embed abi.json
	abiFS embed.FS

	ABI abi.ABI
)

func init() {
	var err error
	ABI, err = pcommon.LoadABI(abiFS, "abi.json")
	if err != nil {
		panic(err)
	}
}

// Precompile implements the VRF EVM precompile.
type Precompile struct {
	pcommon.Precompile

	abi.ABI
	keeper vrfkeeper.Keeper
}

func NewPrecompile(k vrfkeeper.Keeper) *Precompile {
	return &Precompile{
		Precompile: pcommon.Precompile{
			// Avoid extra SDK KV gas costs; the precompile charges its own gas schedule.
			KvGasConfig:          storetypes.GasConfig{},
			TransientKVGasConfig: storetypes.GasConfig{},
		},
		ABI:    ABI,
		keeper: k,
	}
}

// Address returns the canonical VRF precompile address.
func (p Precompile) Address() common.Address {
	return common.HexToAddress(Address)
}

func (p Precompile) RequiredGas(input []byte) uint64 {
	if len(input) < 4 {
		return 0
	}

	methodID := input[:4]
	method, err := p.MethodById(methodID)
	if err != nil {
		return 0
	}

	switch method.Name {
	case LatestRandomnessMethod:
		return GasLatestRandomness
	case RandomWordsMethod:
		return GasRandomWords
	default:
		return 0
	}
}

func (p Precompile) Run(evm *vm.EVM, contract *vm.Contract, readonly bool) ([]byte, error) {
	return p.RunNativeAction(evm, contract, func(ctx sdk.Context) ([]byte, error) {
		return p.Execute(ctx, contract, readonly)
	})
}

func (p Precompile) Execute(ctx sdk.Context, contract *vm.Contract, readOnly bool) ([]byte, error) {
	method, args, err := pcommon.SetupABI(p.ABI, contract)
	if err != nil {
		return nil, err
	}

	_ = readOnly // all methods are view

	switch method.Name {
	case LatestRandomnessMethod:
		return p.latestRandomness(ctx, method)
	case RandomWordsMethod:
		return p.randomWords(ctx, method, args)
	default:
		return nil, fmt.Errorf("vrf precompile: unknown method %q", method.Name)
	}
}

func (p Precompile) latestRandomness(ctx sdk.Context, method *abi.Method) ([]byte, error) {
	beacon, err := p.keeper.GetBeacon(ctx)
	if err != nil {
		return nil, err
	}

	var randomness [32]byte
	copy(randomness[:], beacon.Randomness)

	return method.Outputs.Pack(beacon.DrandRound, randomness)
}

func (p Precompile) randomWords(ctx sdk.Context, method *abi.Method, args []any) ([]byte, error) {
	count, userSeed, err := parseRandomWordsArgs(args)
	if err != nil {
		return nil, err
	}

	_, words, err := p.keeper.ExpandRandomness(ctx, count, userSeed[:])
	if err != nil {
		return nil, err
	}

	// Charge proportional to count (base covers the first word).
	if count > 1 {
		ctx.GasMeter().ConsumeGas(GasRandomWord*uint64(count-1), "vrf precompile randomWords")
	}

	out := make([][32]byte, 0, count)
	for _, w := range words {
		var bz [32]byte
		copy(bz[:], w)
		out = append(out, bz)
	}

	return method.Outputs.Pack(out)
}

func parseRandomWordsArgs(args []any) (uint32, [32]byte, error) {
	if len(args) != 2 {
		return 0, [32]byte{}, fmt.Errorf("vrf precompile: expected 2 args; got %d", len(args))
	}

	countBig, ok := args[0].(*big.Int)
	if !ok {
		return 0, [32]byte{}, fmt.Errorf("vrf precompile: invalid type for count: %T", args[0])
	}
	if countBig.Sign() <= 0 {
		return 0, [32]byte{}, fmt.Errorf("vrf precompile: count must be > 0")
	}
	if countBig.BitLen() > 32 {
		return 0, [32]byte{}, fmt.Errorf("vrf precompile: count overflows uint32")
	}

	countU64 := countBig.Uint64()
	if countU64 > MaxRandomWords {
		return 0, [32]byte{}, fmt.Errorf("vrf precompile: count %d exceeds max %d", countU64, MaxRandomWords)
	}

	var userSeed [32]byte
	switch v := args[1].(type) {
	case [32]byte:
		userSeed = v
	default:
		return 0, [32]byte{}, fmt.Errorf("vrf precompile: invalid type for userSeed: %T", args[1])
	}

	return uint32(countU64), userSeed, nil
}
