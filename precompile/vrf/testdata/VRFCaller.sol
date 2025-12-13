// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.18;

import "../IVRF.sol";

contract VRFCaller {
    function callLatestRandomness() external view returns (uint64 drandRound, bytes32 randomness) {
        return IVRF_CONTRACT.latestRandomness();
    }

    function callRandomWords(uint256 count, bytes32 userSeed) external view returns (bytes32[] memory words) {
        return IVRF_CONTRACT.randomWords(count, userSeed);
    }
}

