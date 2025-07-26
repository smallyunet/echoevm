// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

library MathLibrary {
    // Find maximum of two numbers
    function max(uint256 a, uint256 b) internal pure returns (uint256) {
        return a > b ? a : b;
    }
    
    // Find minimum of two numbers
    function min(uint256 a, uint256 b) internal pure returns (uint256) {
        return a < b ? a : b;
    }
    
    // Calculate average of two numbers
    function average(uint256 a, uint256 b) internal pure returns (uint256) {
        return (a + b) / 2;
    }
    
    // Calculate percentage
    function percentage(uint256 amount, uint256 percent) internal pure returns (uint256) {
        return (amount * percent) / 100;
    }
    
    // Check if number is even
    function isEven(uint256 number) internal pure returns (bool) {
        return number % 2 == 0;
    }
    
    // Check if number is odd
    function isOdd(uint256 number) internal pure returns (bool) {
        return number % 2 != 0;
    }
    
    // Calculate factorial (limited to small numbers)
    function factorial(uint256 n) internal pure returns (uint256) {
        if (n == 0 || n == 1) {
            return 1;
        }
        uint256 result = 1;
        for (uint256 i = 2; i <= n; i++) {
            result *= i;
        }
        return result;
    }
    
    // Calculate power
    function power(uint256 base, uint256 exponent) internal pure returns (uint256) {
        if (exponent == 0) {
            return 1;
        }
        uint256 result = base;
        for (uint256 i = 1; i < exponent; i++) {
            result *= base;
        }
        return result;
    }
    
    // Calculate square root (integer approximation)
    function sqrt(uint256 x) internal pure returns (uint256) {
        if (x == 0) return 0;
        if (x == 1) return 1;
        
        uint256 z = (x + 1) / 2;
        uint256 y = x;
        while (z < y) {
            y = z;
            z = (x / z + z) / 2;
        }
        return y;
    }
} 