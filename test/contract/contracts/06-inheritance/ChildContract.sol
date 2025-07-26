// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "./BaseContract.sol";

contract ChildContract is BaseContract {
    string public childName;
    uint256 public childValue;
    
    event ChildValueSet(string name, uint256 value);
    
    constructor(string memory _name) {
        childName = _name;
    }
    
    // Override virtual function
    function setValue(uint256 _value) public virtual override onlyOwner {
        super.setValue(_value);
        childValue = _value * 2;
        emit ChildValueSet(childName, childValue);
    }
    
    // Override virtual function completely
    function virtualFunction() public view override returns (string memory) {
        return string(abi.encodePacked("Child implementation for ", childName));
    }
    
    // Override initialize function
    function initialize() public override {
        super.initialize();
        childValue = 100;
    }
    
    // New function specific to child
    function setChildName(string memory _name) public onlyOwner {
        childName = _name;
    }
    
    function getChildInfo() public view returns (string memory, uint256, uint256) {
        return (childName, childValue, value);
    }
    
    // Function that uses parent's modifier
    function childOnlyOwnerFunction() public view onlyOwner returns (string memory) {
        return "Child owner function called";
    }
    
    // Function that uses parent's modifier
    function childOnlyInitializedFunction() public view onlyInitialized returns (string memory) {
        return "Child initialized function called";
    }
} 