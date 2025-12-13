package common

import (
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/stretchr/testify/require"
)

func TestRevertReasonBytes(t *testing.T) {
	t.Parallel()

	bz, err := RevertReasonBytes("boom")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(bz), 4)
	require.Equal(t, []byte{0x08, 0xC3, 0x79, 0xA0}, bz[:4])

	stringType, err := abi.NewType("string", "", nil)
	require.NoError(t, err)

	args := abi.Arguments{{Type: stringType}}
	out, err := args.Unpack(bz[4:])
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, "boom", out[0].(string))
}
