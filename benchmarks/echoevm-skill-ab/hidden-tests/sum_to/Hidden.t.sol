// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {SumTo} from "../src/SumTo.sol";

contract HiddenSumToTest {
    SumTo private sum = new SumTo();

    function testCorrectValues() public view {
        require(sum.sumTo(0) == 0, "zero");
        require(sum.sumTo(1) == 1, "one");
        require(sum.sumTo(100) == 5050, "hundred");
        require(sum.sumTo(1_000_000) == 500000500000, "million");
    }

    function testLargeCallUsesBoundedGas() public {
        uint256 beforeGas = gasleft();
        uint256 result = sum.sumTo(1_000_000);
        uint256 used = beforeGas - gasleft();
        require(result == 500000500000, "result");
        require(used < 20_000, "gas regression");
    }

    function testRejectsTooLargeInput() public {
        try sum.sumTo(1_000_001) returns (uint256) {
            revert("accepted invalid n");
        } catch {}
    }
}
