// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "./MathLibrary.sol";

contract LibraryUser {
    using MathLibrary for uint256;
    
    uint256 public value1;
    uint256 public value2;
    
    function setValues(uint256 _value1, uint256 _value2) public {
        value1 = _value1;
        value2 = _value2;
    }
    
    function getMax() public view returns (uint256) {
        return MathLibrary.max(value1, value2);
    }
    
    function getMin() public view returns (uint256) {
        return MathLibrary.min(value1, value2);
    }
    
    function getAverage() public view returns (uint256) {
        return MathLibrary.average(value1, value2);
    }
    
    function getPercentage(uint256 percent) public view returns (uint256) {
        return MathLibrary.percentage(value1, percent);
    }
    
    function checkEven() public view returns (bool) {
        return MathLibrary.isEven(value1);
    }
    
    function checkOdd() public view returns (bool) {
        return MathLibrary.isOdd(value1);
    }
    
    function getFactorial(uint256 n) public pure returns (uint256) {
        return MathLibrary.factorial(n);
    }
    
    function getPower(uint256 base, uint256 exponent) public pure returns (uint256) {
        return MathLibrary.power(base, exponent);
    }
    
    function getSquareRoot(uint256 x) public pure returns (uint256) {
        return MathLibrary.sqrt(x);
    }
    
    // Using the library with the 'using' directive
    function usingMax(uint256 a, uint256 b) public pure returns (uint256) {
        return a.max(b);
    }
    
    function usingMin(uint256 a, uint256 b) public pure returns (uint256) {
        return a.min(b);
    }
    
    function usingAverage(uint256 a, uint256 b) public pure returns (uint256) {
        return a.average(b);
    }
    
    function usingPercentage(uint256 amount, uint256 percent) public pure returns (uint256) {
        return amount.percentage(percent);
    }
    
    function usingIsEven(uint256 number) public pure returns (bool) {
        return number.isEven();
    }
    
    function usingIsOdd(uint256 number) public pure returns (bool) {
        return number.isOdd();
    }
} 