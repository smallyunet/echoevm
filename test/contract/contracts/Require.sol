// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract Require {
    function test(uint256 a) public pure {
        require(a > 0, "a must be greater than 0");
    }
}
