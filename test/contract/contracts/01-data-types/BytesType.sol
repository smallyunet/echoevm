// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract BytesType {
    // Fixed-size bytes
    bytes1 public bytes1Value = 0x01;
    bytes2 public bytes2Value = 0x0102;
    bytes32 public bytes32Value = 0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef;
    
    // Dynamic bytes
    bytes public dynamicBytes;
    
    function setDynamicBytes(bytes memory _data) public {
        dynamicBytes = _data;
    }
    
    function getDynamicBytes() public view returns (bytes memory) {
        return dynamicBytes;
    }
    
    function getDynamicBytesLength() public view returns (uint256) {
        return dynamicBytes.length;
    }
    
    function pushByte(bytes1 _byte) public {
        dynamicBytes.push(_byte);
    }
    
    function popByte() public {
        dynamicBytes.pop();
    }
    
    function concatenate(bytes memory _a, bytes memory _b) public pure returns (bytes memory) {
        return abi.encodePacked(_a, _b);
    }
    
    function slice(bytes memory _data, uint256 _start, uint256 _length) public pure returns (bytes memory) {
        bytes memory result = new bytes(_length);
        for (uint256 i = 0; i < _length; i++) {
            result[i] = _data[_start + i];
        }
        return result;
    }
    
    function toHexString(bytes memory _data) public pure returns (string memory) {
        bytes memory alphabet = "0123456789abcdef";
        bytes memory str = new bytes(2 + _data.length * 2);
        str[0] = "0";
        str[1] = "x";
        for (uint256 i = 0; i < _data.length; i++) {
            str[2 + i * 2] = alphabet[uint8(_data[i] >> 4)];
            str[3 + i * 2] = alphabet[uint8(_data[i] & 0x0f)];
        }
        return string(str);
    }
} 