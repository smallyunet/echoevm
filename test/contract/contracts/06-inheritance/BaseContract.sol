// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract BaseContract {
    address public owner;
    uint256 public value;
    bool public initialized;
    
    event ValueChanged(uint256 oldValue, uint256 newValue);
    event OwnerChanged(address indexed oldOwner, address indexed newOwner);
    
    constructor() {
        owner = msg.sender;
    }
    
    modifier onlyOwner() {
        require(msg.sender == owner, "Only owner can call this function");
        _;
    }
    
    modifier onlyInitialized() {
        require(initialized, "Contract not initialized");
        _;
    }
    
    function setValue(uint256 _value) public virtual onlyOwner {
        uint256 oldValue = value;
        value = _value;
        emit ValueChanged(oldValue, _value);
    }
    
    function getValue() public view returns (uint256) {
        return value;
    }
    
    function transferOwnership(address _newOwner) public onlyOwner {
        require(_newOwner != address(0), "Invalid new owner");
        address oldOwner = owner;
        owner = _newOwner;
        emit OwnerChanged(oldOwner, _newOwner);
    }
    
    function initialize() public virtual {
        require(!initialized, "Already initialized");
        initialized = true;
    }
    
    // Function that can be overridden
    function virtualFunction() public virtual returns (string memory) {
        return "Base implementation";
    }
    
    // Function that cannot be overridden
    function finalFunction() public pure returns (string memory) {
        return "This cannot be overridden";
    }
} 