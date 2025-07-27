// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract AddressType {
    address public owner;
    address public contractAddress;
    
    constructor() {
        owner = msg.sender;
        contractAddress = address(this);
    }
    
    function getBalance(address _address) public view returns (uint256) {
        return _address.balance;
    }
    
    function transfer(address payable _to) public payable {
        _to.transfer(msg.value);
    }
    
    function send(address payable _to, uint256 _amount) public returns (bool) {
        return _to.send(_amount);
    }
    
    function call(address _target, bytes calldata _data) public returns (bool, bytes memory) {
        (bool success, bytes memory data) = _target.call(_data);
        return (success, data);
    }
    
    function delegateCall(address _target, bytes calldata _data) public returns (bool, bytes memory) {
        (bool success, bytes memory data) = _target.delegatecall(_data);
        return (success, data);
    }
    
    function staticCall(address _target, bytes calldata _data) public view returns (bool, bytes memory) {
        return _target.staticcall(_data);
    }
    
    function isContract(address _address) public view returns (bool) {
        return _address.code.length > 0;
    }
    
    function getCodeSize(address _address) public view returns (uint256) {
        return _address.code.length;
    }
    
    // Add receive function to accept ETH
    receive() external payable {}
} 