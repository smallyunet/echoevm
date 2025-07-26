// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract ArrayTypes {
    // Fixed-size arrays
    uint256[5] public fixedArray = [1, 2, 3, 4, 5];
    
    // Dynamic arrays
    uint256[] public dynamicArray;
    string[] public stringArray;
    
    // Multi-dimensional arrays
    uint256[2][3] public multiArray = [[1, 2], [3, 4], [5, 6]];
    
    constructor() {
        dynamicArray.push(10);
        dynamicArray.push(20);
        dynamicArray.push(30);
        
        stringArray.push("Hello");
        stringArray.push("World");
    }
    
    function getFixedArrayElement(uint256 index) public view returns (uint256) {
        require(index < 5, "Index out of bounds");
        return fixedArray[index];
    }
    
    function setFixedArrayElement(uint256 index, uint256 value) public {
        require(index < 5, "Index out of bounds");
        fixedArray[index] = value;
    }
    
    function getDynamicArrayLength() public view returns (uint256) {
        return dynamicArray.length;
    }
    
    function getDynamicArrayElement(uint256 index) public view returns (uint256) {
        require(index < dynamicArray.length, "Index out of bounds");
        return dynamicArray[index];
    }
    
    function addToDynamicArray(uint256 value) public {
        dynamicArray.push(value);
    }
    
    function removeFromDynamicArray() public {
        require(dynamicArray.length > 0, "Array is empty");
        dynamicArray.pop();
    }
    
    function setDynamicArrayElement(uint256 index, uint256 value) public {
        require(index < dynamicArray.length, "Index out of bounds");
        dynamicArray[index] = value;
    }
    
    function getMultiArrayElement(uint256 row, uint256 col) public view returns (uint256) {
        require(row < 3 && col < 2, "Index out of bounds");
        return multiArray[row][col];
    }
    
    function setMultiArrayElement(uint256 row, uint256 col, uint256 value) public {
        require(row < 3 && col < 2, "Index out of bounds");
        multiArray[row][col] = value;
    }
    
    function getStringArrayElement(uint256 index) public view returns (string memory) {
        require(index < stringArray.length, "Index out of bounds");
        return stringArray[index];
    }
    
    function addToStringArray(string memory value) public {
        stringArray.push(value);
    }
    
    function getStringArrayLength() public view returns (uint256) {
        return stringArray.length;
    }
    
    function deleteArrayElement(uint256 index) public {
        require(index < dynamicArray.length, "Index out of bounds");
        delete dynamicArray[index];
    }
    
    function getArraySlice(uint256 start, uint256 end) public view returns (uint256[] memory) {
        require(start < dynamicArray.length && end <= dynamicArray.length && start < end, "Invalid slice");
        uint256[] memory slice = new uint256[](end - start);
        for (uint256 i = start; i < end; i++) {
            slice[i - start] = dynamicArray[i];
        }
        return slice;
    }
} 