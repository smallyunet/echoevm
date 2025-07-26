// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract Assembly {
    uint256 public value;
    
    // Basic assembly operation
    function addAssembly(uint256 a, uint256 b) public pure returns (uint256) {
        assembly {
            let result := add(a, b)
            mstore(0x0, result)
            return(0x0, 32)
        }
    }
    
    // Assembly with storage access
    function setValueAssembly(uint256 _value) public {
        assembly {
            sstore(value.slot, _value)
        }
    }
    
    // Assembly with memory operations
    function memoryOperations() public pure returns (bytes32) {
        assembly {
            // Store value in memory
            mstore(0x0, 0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef)
            // Return the value
            return(0x0, 32)
        }
    }
    
    // Assembly with conditional logic
    function conditionalAssembly(uint256 a, uint256 b) public pure returns (uint256) {
        assembly {
            let result := 0
            if gt(a, b) {
                result := a
            }
            if lt(a, b) {
                result := b
            }
            mstore(0x0, result)
            return(0x0, 32)
        }
    }
    
    // Assembly with loops
    function loopAssembly(uint256 count) public pure returns (uint256) {
        assembly {
            let sum := 0
            for { let i := 0 } lt(i, count) { i := add(i, 1) } {
                sum := add(sum, i)
            }
            mstore(0x0, sum)
            return(0x0, 32)
        }
    }
    
    // Assembly with function calls
    function callAssembly() public returns (uint256) {
        assembly {
            // Call a function and get its return value
            let result := call(gas(), address(), 0, 0, 0, 0, 0)
            mstore(0x0, result)
            return(0x0, 32)
        }
    }
    
    // Assembly with bitwise operations
    function bitwiseAssembly(uint256 a, uint256 b) public pure returns (uint256, uint256, uint256) {
        assembly {
            let andResult := and(a, b)
            let orResult := or(a, b)
            let xorResult := xor(a, b)
            
            mstore(0x0, andResult)
            mstore(0x20, orResult)
            mstore(0x40, xorResult)
            return(0x0, 96)
        }
    }
    
    // Assembly with stack operations
    function stackAssembly(uint256 a, uint256 b, uint256 /* c */) public pure returns (uint256) {
        assembly {
            // Simple arithmetic operation using assembly
            let result := add(a, b)
            mstore(0x0, result)
            return(0x0, 32)
        }
    }
} 