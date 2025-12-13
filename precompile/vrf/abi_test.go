package vrf

import (
	"testing"

	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestABISelectors(t *testing.T) {
	t.Parallel()

	latest := ABI.Methods[LatestRandomnessMethod]
	require.Equal(t, gethcrypto.Keccak256([]byte("latestRandomness()"))[:4], latest.ID)

	words := ABI.Methods[RandomWordsMethod]
	require.Equal(t, gethcrypto.Keccak256([]byte("randomWords(uint256,bytes32)"))[:4], words.ID)
}
