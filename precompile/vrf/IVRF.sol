// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.18;

/// @dev The IVRF contract's address.
address constant IVRF_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000807;

/// @dev The IVRF contract's instance.
IVRF constant IVRF_CONTRACT = IVRF(IVRF_PRECOMPILE_ADDRESS);

/// @title VRF Precompile Interface
/// @dev Interface for fetching the current block randomness and expanding it into random words.
/// @custom:address 0x0000000000000000000000000000000000000807
interface IVRF {
    /// @notice Returns the canonical randomness for the current block.
    function latestRandomness() external view returns (
        uint64 drandRound,
        bytes32 randomness
    );

    /// @notice Expands the current block's VRF seed into `count` random words.
    /// @dev words[i] = keccak256(abi.encode(chainHash, drandRound, randomness, userSeed, i)).
    function randomWords(uint256 count, bytes32 userSeed)
        external
        view
        returns (bytes32[] memory words);
}

