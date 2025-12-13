package common

import (
	"bytes"
	"embed"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/vm"
)

// LoadABI reads an embedded ABI json file and parses it.
func LoadABI(fs embed.FS, path string) (abi.ABI, error) {
	abiBz, err := fs.ReadFile(path)
	if err != nil {
		return abi.ABI{}, fmt.Errorf("precompile: failed to read ABI %q: %w", path, err)
	}

	parsed, err := abi.JSON(bytes.NewReader(abiBz))
	if err != nil {
		return abi.ABI{}, fmt.Errorf("precompile: failed to parse ABI %q: %w", path, err)
	}

	return parsed, nil
}

// SetupABI decodes the ABI method and unpacked args from the contract input.
// It returns an error for unknown methods or malformed calldata.
func SetupABI(api abi.ABI, contract *vm.Contract) (*abi.Method, []any, error) {
	if len(contract.Input) < 4 {
		return nil, nil, vm.ErrExecutionReverted
	}

	methodID := contract.Input[:4]
	method, err := api.MethodById(methodID)
	if err != nil {
		return nil, nil, err
	}

	var args []any
	if len(contract.Input) > 4 {
		args, err = method.Inputs.Unpack(contract.Input[4:])
		if err != nil {
			return nil, nil, err
		}
	}

	return method, args, nil
}
