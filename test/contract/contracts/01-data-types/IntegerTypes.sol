// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract IntegerTypes {
    uint8 public uint8Value = 255;
    uint16 public uint16Value = 65535;
    uint256 public uint256Value = 123456789;
    
    int8 public int8Value = -128;
    int16 public int16Value = -32768;
    int256 public int256Value = -123456789;
    
    function add(uint256 a, uint256 b) public pure returns (uint256) {
        return a + b;
    }
    
    function subtract(uint256 a, uint256 b) public pure returns (uint256) {
        return a - b;
    }
    
    function multiply(uint256 a, uint256 b) public pure returns (uint256) {
        return a * b;
    }
    
    function divide(uint256 a, uint256 b) public pure returns (uint256) {
        return a / b;
    }
    
    function modulo(uint256 a, uint256 b) public pure returns (uint256) {
        return a % b;
    }
    
    function power(uint256 base, uint256 exponent) public pure returns (uint256) {
        return base ** exponent;
    }
    
    function increment(uint256 value) public pure returns (uint256) {
        return value + 1;
    }
    
    function decrement(uint256 value) public pure returns (uint256) {
        return value - 1;
    }
} 