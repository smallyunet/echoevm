// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract FunctionMutability {
    uint256 public stateVariable = 100;
    
    // Pure function - doesn't read or modify state
    function pureFunction(uint256 a, uint256 b) public pure returns (uint256) {
        return a + b;
    }
    
    // View function - reads state but doesn't modify it
    function viewFunction() public view returns (uint256) {
        return stateVariable;
    }
    
    // Payable function - can receive ETH
    function payableFunction() public payable returns (uint256) {
        return msg.value;
    }
    
    // Regular function - can read and modify state
    function regularFunction(uint256 _value) public returns (uint256) {
        stateVariable = _value;
        return stateVariable;
    }
    
    // Function that combines view and pure operations
    function complexViewFunction(uint256 _multiplier) public view returns (uint256) {
        return stateVariable * _multiplier;
    }
    
    // Function that demonstrates pure operations
    function pureOperations(uint256 a, uint256 b) public pure returns (uint256, uint256, uint256) {
        uint256 sum = a + b;
        uint256 product = a * b;
        uint256 difference = a > b ? a - b : b - a;
        return (sum, product, difference);
    }
    
    // Function that demonstrates view operations
    function viewOperations() public view returns (uint256, uint256) {
        uint256 currentValue = stateVariable;
        uint256 doubledValue = currentValue * 2;
        return (currentValue, doubledValue);
    }
    
    // Function that demonstrates state modification
    function stateModification(uint256 _newValue) public returns (uint256, uint256) {
        uint256 oldValue = stateVariable;
        stateVariable = _newValue;
        return (oldValue, stateVariable);
    }
    
    // Function that demonstrates payable operations
    function payableOperations() public payable returns (uint256, uint256) {
        uint256 receivedValue = msg.value;
        uint256 totalValue = stateVariable + receivedValue;
        return (receivedValue, totalValue);
    }
} 