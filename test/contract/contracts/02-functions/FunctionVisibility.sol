// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract FunctionVisibility {
    uint256 public publicValue = 100;
    uint256 private privateValue = 200;
    uint256 internal internalValue = 300;
    uint256 public externalValue = 400;
    
    // Public function - accessible from everywhere
    function publicFunction() public pure returns (string memory) {
        return "Public function called";
    }
    
    // Private function - only accessible within this contract
    function privateFunction() private pure returns (string memory) {
        return "Private function called";
    }
    
    // Internal function - accessible within this contract and derived contracts
    function internalFunction() internal pure returns (string memory) {
        return "Internal function called";
    }
    
    // External function - only accessible from outside the contract
    function externalFunction() external pure returns (string memory) {
        return "External function called";
    }
    
    // Function that calls private function
    function callPrivateFunction() public pure returns (string memory) {
        return privateFunction();
    }
    
    // Function that calls internal function
    function callInternalFunction() public pure returns (string memory) {
        return internalFunction();
    }
    
    // Function that accesses private state variable
    function getPrivateValue() public view returns (uint256) {
        return privateValue;
    }
    
    // Function that accesses internal state variable
    function getInternalValue() public view returns (uint256) {
        return internalValue;
    }
    
    // Function that modifies private state variable
    function setPrivateValue(uint256 _value) public {
        privateValue = _value;
    }
    
    // Function that modifies internal state variable
    function setInternalValue(uint256 _value) public {
        internalValue = _value;
    }
} 