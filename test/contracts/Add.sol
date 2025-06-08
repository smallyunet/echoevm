// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract Add {
    function add() public pure returns (uint) {
        return 5 + 6;
    }

    // add allows addition of two parameters to demonstrate
    // calldata usage when running runtime bytecode.
    function add(uint256 a, uint256 b) public pure returns (uint256) {
        return a + b;
    }
}
