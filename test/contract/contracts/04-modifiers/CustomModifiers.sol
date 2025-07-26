// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract CustomModifiers {
    address public owner;
    uint256 public value;
    bool public paused;
    
    constructor() {
        owner = msg.sender;
    }
    
    // Basic modifier - only owner can call
    modifier onlyOwner() {
        require(msg.sender == owner, "Only owner can call this function");
        _;
    }
    
    // Modifier with parameter
    modifier onlyValue(uint256 _minValue) {
        require(value >= _minValue, "Value too low");
        _;
    }
    
    // Modifier that checks if contract is not paused
    modifier whenNotPaused() {
        require(!paused, "Contract is paused");
        _;
    }
    
    // Modifier that checks if contract is paused
    modifier whenPaused() {
        require(paused, "Contract is not paused");
        _;
    }
    
    // Modifier with multiple conditions
    modifier onlyOwnerAndNotPaused() {
        require(msg.sender == owner, "Only owner can call this function");
        require(!paused, "Contract is paused");
        _;
    }
    
    // Modifier that executes code before and after
    modifier beforeAndAfter() {
        // Code before function execution
        value += 1;
        _;
        // Code after function execution
        value += 1;
    }
    
    // Functions using modifiers
    function setValue(uint256 _value) public onlyOwner whenNotPaused {
        value = _value;
    }
    
    function incrementValue() public onlyValue(10) beforeAndAfter {
        value += 5;
    }
    
    function pause() public onlyOwner {
        paused = true;
    }
    
    function unpause() public onlyOwner whenPaused {
        paused = false;
    }
    
    function emergencyFunction() public onlyOwnerAndNotPaused {
        value = 0;
    }
    
    function getValue() public view returns (uint256) {
        return value;
    }
    
    function transferOwnership(address _newOwner) public onlyOwner {
        owner = _newOwner;
    }
} 