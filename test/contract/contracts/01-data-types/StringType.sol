// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract StringType {
    string public message = "Hello World";
    string public emptyString = "";
    
    function setMessage(string memory _message) public {
        message = _message;
    }
    
    function getMessage() public view returns (string memory) {
        return message;
    }
    
    function concatenate(string memory _a, string memory _b) public pure returns (string memory) {
        return string(abi.encodePacked(_a, _b));
    }
    
    function concatenateWithBytes(string memory _str, bytes memory _bytes) public pure returns (string memory) {
        return string(abi.encodePacked(_str, _bytes));
    }
    
    function stringToBytes(string memory _str) public pure returns (bytes memory) {
        return bytes(_str);
    }
    
    function bytesToString(bytes memory _bytes) public pure returns (string memory) {
        return string(_bytes);
    }
    
    function getStringLength(string memory _str) public pure returns (uint256) {
        return bytes(_str).length;
    }
    
    function isEmpty(string memory _str) public pure returns (bool) {
        return bytes(_str).length == 0;
    }
    
    function compareStrings(string memory _a, string memory _b) public pure returns (bool) {
        return keccak256(abi.encodePacked(_a)) == keccak256(abi.encodePacked(_b));
    }
} 