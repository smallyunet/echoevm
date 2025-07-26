// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract Loops {
    uint256[] public numbers;
    
    constructor() {
        numbers.push(1);
        numbers.push(2);
        numbers.push(3);
        numbers.push(4);
        numbers.push(5);
    }
    
    // For loop
    function forLoop(uint256 _count) public pure returns (uint256) {
        uint256 sum = 0;
        for (uint256 i = 0; i < _count; i++) {
            sum += i;
        }
        return sum;
    }
    
    // While loop
    function whileLoop(uint256 _target) public pure returns (uint256) {
        uint256 counter = 0;
        while (counter < _target) {
            counter++;
        }
        return counter;
    }
    
    // Do-while loop
    function doWhileLoop(uint256 _target) public pure returns (uint256) {
        uint256 counter = 0;
        do {
            counter++;
        } while (counter < _target);
        return counter;
    }
    
    // Loop with break
    function loopWithBreak(uint256 _target) public pure returns (uint256) {
        uint256 sum = 0;
        for (uint256 i = 0; i < 100; i++) {
            if (i >= _target) {
                break;
            }
            sum += i;
        }
        return sum;
    }
    
    // Loop with continue
    function loopWithContinue(uint256 _max) public pure returns (uint256) {
        uint256 sum = 0;
        for (uint256 i = 0; i <= _max; i++) {
            if (i % 2 == 0) {
                continue; // Skip even numbers
            }
            sum += i;
        }
        return sum;
    }
    
    // Nested loops
    function nestedLoops(uint256 _rows, uint256 _cols) public pure returns (uint256) {
        uint256 total = 0;
        for (uint256 i = 0; i < _rows; i++) {
            for (uint256 j = 0; j < _cols; j++) {
                total += i * j;
            }
        }
        return total;
    }
    
    // Loop through array
    function sumArray() public view returns (uint256) {
        uint256 sum = 0;
        for (uint256 i = 0; i < numbers.length; i++) {
            sum += numbers[i];
        }
        return sum;
    }
    
    // Loop with multiple variables
    function multipleVariables(uint256 _count) public pure returns (uint256, uint256) {
        uint256 sum = 0;
        uint256 product = 1;
        for (uint256 i = 1; i <= _count; i++) {
            sum += i;
            product *= i;
        }
        return (sum, product);
    }
    
    // Infinite loop prevention
    function safeLoop(uint256 _max) public pure returns (uint256) {
        uint256 counter = 0;
        uint256 maxIterations = 1000; // Safety limit
        
        while (counter < _max && counter < maxIterations) {
            counter++;
        }
        return counter;
    }
} 