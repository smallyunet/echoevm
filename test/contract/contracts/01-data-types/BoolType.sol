// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract BoolType {
    bool public isActive = true;
    bool public isPaused = false;
    
    function toggleActive() public {
        isActive = !isActive;
    }
    
    function setActive(bool _active) public {
        isActive = _active;
    }
    
    function getActiveStatus() public view returns (bool) {
        return isActive;
    }
    
    function logicalAnd(bool a, bool b) public pure returns (bool) {
        return a && b;
    }
    
    function logicalOr(bool a, bool b) public pure returns (bool) {
        return a || b;
    }
    
    function logicalNot(bool a) public pure returns (bool) {
        return !a;
    }
} 