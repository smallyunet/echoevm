// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract IfElse {
    uint256 public value = 50;
    
    function simpleIf(uint256 _input) public pure returns (string memory) {
        if (_input > 100) {
            return "Greater than 100";
        }
        return "Less than or equal to 100";
    }
    
    function ifElse(uint256 _input) public pure returns (string memory) {
        if (_input > 100) {
            return "Greater than 100";
        } else {
            return "Less than or equal to 100";
        }
    }
    
    function ifElseIf(uint256 _input) public pure returns (string memory) {
        if (_input > 100) {
            return "Greater than 100";
        } else if (_input > 50) {
            return "Greater than 50 but less than or equal to 100";
        } else if (_input > 0) {
            return "Greater than 0 but less than or equal to 50";
        } else {
            return "Zero or negative";
        }
    }
    
    function nestedIf(uint256 _input) public pure returns (string memory) {
        if (_input > 0) {
            if (_input > 100) {
                return "Positive and greater than 100";
            } else {
                return "Positive but less than or equal to 100";
            }
        } else {
            return "Zero or negative";
        }
    }
    
    function conditionalAssignment(uint256 _input) public pure returns (uint256) {
        uint256 result;
        if (_input > 100) {
            result = _input * 2;
        } else {
            result = _input / 2;
        }
        return result;
    }
    
    function ternaryOperator(uint256 _input) public pure returns (string memory) {
        return _input > 100 ? "Large" : "Small";
    }
    
    function complexConditional(uint256 _input) public pure returns (string memory) {
        if (_input > 100 && _input % 2 == 0) {
            return "Large even number";
        } else if (_input > 100 && _input % 2 != 0) {
            return "Large odd number";
        } else if (_input <= 100 && _input % 2 == 0) {
            return "Small even number";
        } else {
            return "Small odd number";
        }
    }
    
    function stateBasedConditional() public view returns (string memory) {
        if (value > 100) {
            return "State value is large";
        } else if (value > 50) {
            return "State value is medium";
        } else {
            return "State value is small";
        }
    }
} 