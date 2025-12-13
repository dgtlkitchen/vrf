package vrf

const (
	// Address is the canonical address for the VRF precompile.
	Address = "0x0000000000000000000000000000000000000807"

	LatestRandomnessMethod = "latestRandomness"
	RandomWordsMethod      = "randomWords"

	MaxRandomWords = 256
)

const (
	// GasLatestRandomness defines the base gas cost for latestRandomness().
	GasLatestRandomness uint64 = 2_000

	// GasRandomWords defines the base gas cost for randomWords().
	GasRandomWords uint64 = 2_000

	// GasRandomWord defines the additional gas cost per extra word beyond the first.
	GasRandomWord uint64 = 500
)
