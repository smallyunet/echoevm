// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

contract SumTo {
    function sumTo(uint256 n) external pure returns (uint256 total) {
        require(n <= 1_000_000, "n too large");
        for (uint256 i = 0; i <= n; ++i) {
            total += i;
        }
    }
}
