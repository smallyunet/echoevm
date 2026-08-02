// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract Calculator {
    uint256 private stored;

    constructor(uint256 initialValue) {
        stored = initialValue;
    }

    function add(uint256 left, uint256 right) external pure returns (uint256) {
        return left + right;
    }

    function store(uint256 value) external {
        stored = value;
    }

    function read() external view returns (uint256) {
        return stored;
    }
}
