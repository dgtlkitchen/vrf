package common

import (
	"errors"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
)

var ErrNotRunInEVM = errors.New("precompile: not run in Cosmos EVM")

type stateDBWithCacheContext interface {
	GetCacheContext() (sdk.Context, error)
	MultiStoreSnapshot() int
	AddPrecompileFn(snapshot int, events sdk.Events) error
	CommitWithCacheCtx() error
}

type stateDBWithContext interface {
	GetContext() sdk.Context
}

type NativeAction func(ctx sdk.Context) ([]byte, error)

// Precompile is a small helper for precompiles that need Cosmos SDK context/state access.
type Precompile struct {
	KvGasConfig          storetypes.GasConfig
	TransientKVGasConfig storetypes.GasConfig
}

func (p Precompile) RunNativeAction(evm *vm.EVM, contract *vm.Contract, action NativeAction) ([]byte, error) {
	bz, err := p.runNativeAction(evm, contract, action)
	if err != nil {
		return ReturnRevertError(evm, err)
	}
	return bz, nil
}

func (p Precompile) runNativeAction(evm *vm.EVM, contract *vm.Contract, action NativeAction) (bz []byte, err error) {
	var (
		ctx sdk.Context
	)

	// Prefer a cache-aware stateDB (e.g. Evmos/Ethermint-style implementations) when available.
	if stateDB, ok := evm.StateDB.(stateDBWithCacheContext); ok {
		ctx, err = stateDB.GetCacheContext()
		if err != nil {
			return nil, err
		}

		snapshot := stateDB.MultiStoreSnapshot()
		events := ctx.EventManager().Events()
		if err := stateDB.AddPrecompileFn(snapshot, events); err != nil {
			return nil, err
		}

		if err := stateDB.CommitWithCacheCtx(); err != nil {
			return nil, err
		}
	} else if stateDB, ok := evm.StateDB.(stateDBWithContext); ok {
		// Fallback for stateDBs that at least expose a Cosmos SDK context.
		ctx = stateDB.GetContext()
	} else {
		return nil, ErrNotRunInEVM
	}

	initialGas := ctx.GasMeter().GasConsumed()
	defer handleGasError(ctx, contract, initialGas, &err)()

	ctx = ctx.WithGasMeter(storetypes.NewGasMeter(contract.Gas)).
		WithKVGasConfig(p.KvGasConfig).
		WithTransientKVGasConfig(p.TransientKVGasConfig)
	ctx.GasMeter().ConsumeGas(initialGas, "creating a new gas meter")

	bz, err = action(ctx)
	if err != nil {
		return nil, err
	}

	cost := ctx.GasMeter().GasConsumed() - initialGas
	if !contract.UseGas(cost, nil, tracing.GasChangeCallPrecompiledContract) {
		return nil, vm.ErrOutOfGas
	}

	return bz, nil
}

// handleGasError converts Cosmos SDK out-of-gas panics into EVM errors.
func handleGasError(ctx sdk.Context, contract *vm.Contract, initialGas storetypes.Gas, err *error) func() {
	return func() {
		if r := recover(); r != nil {
			switch r.(type) {
			case storetypes.ErrorOutOfGas:
				usedGas := ctx.GasMeter().GasConsumed() - initialGas
				_ = contract.UseGas(usedGas, nil, tracing.GasChangeCallFailedExecution)
				*err = vm.ErrOutOfGas
				ctx = ctx.WithKVGasConfig(storetypes.GasConfig{}).
					WithTransientKVGasConfig(storetypes.GasConfig{})
			default:
				panic(r)
			}
		}
	}
}
