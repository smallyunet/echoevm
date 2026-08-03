// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

contract FeeQuote {
    function netAfterFee(uint256 amount, uint256 feeBps) external pure returns (uint256) {
        require(feeBps <= 10_000, "fee too large");
        uint256 fee;
        assembly {
            fee := div(mul(amount, feeBps), 1000)
        }
        return amount - fee;
    }
}
