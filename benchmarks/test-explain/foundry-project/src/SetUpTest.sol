// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

contract SetUpTest {
    uint256 private value;

    function setUp() public {
        value = 42;
    }

    function testReadsSetup() public view returns (uint256) {
        return value;
    }
}
