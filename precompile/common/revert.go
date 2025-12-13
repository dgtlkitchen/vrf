package common

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/vm"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

var errSelector = gethcrypto.Keccak256([]byte("Error(string)"))[:4] // 0x08c379a0

// RevertReasonBytes ABI-encodes a Solidity `Error(string)` revert reason.
func RevertReasonBytes(reason string) ([]byte, error) {
	stringType, err := abi.NewType("string", "", nil)
	if err != nil {
		return nil, fmt.Errorf("precompile: failed to construct abi string type: %w", err)
	}

	args := abi.Arguments{{Type: stringType}}
	encoded, err := args.Pack(reason)
	if err != nil {
		return nil, fmt.Errorf("precompile: failed to ABI-pack revert reason: %w", err)
	}

	out := make([]byte, 0, len(errSelector)+len(encoded))
	out = append(out, errSelector...)
	out = append(out, encoded...)
	return out, nil
}

// ReturnRevertError aligns precompile errors with go-ethereum revert behavior
// by ABI-encoding the reason and setting it as return data.
func ReturnRevertError(evm *vm.EVM, err error) ([]byte, error) {
	revertReason, encErr := RevertReasonBytes(err.Error())
	if encErr != nil {
		return nil, vm.ErrExecutionReverted
	}

	evm.Interpreter().SetReturnData(revertReason)
	return revertReason, vm.ErrExecutionReverted
}
