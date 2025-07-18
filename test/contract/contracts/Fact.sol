// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract Fact {
    function fact(uint256 n) public pure returns (uint256) {
        if (n == 0)
        {
            return 1;
        }
        else
        {
            return n * fact(n - 1);
        }
    }
}
