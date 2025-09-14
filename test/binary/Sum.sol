// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract Sum {
    function sum(uint256 n) public pure returns (uint256) {
        uint256 result = 0;
        for (uint256 i = 1; i <= n; i++) {
            result += i;
        }
        return result;
    }
}
