// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {FeeQuote} from "../src/FeeQuote.sol";

contract HiddenFeeQuoteTest {
    FeeQuote private quote = new FeeQuote();

    function testExamplesAndBoundaries() public view {
        require(quote.netAfterFee(250000, 250) == 243750, "example");
        require(quote.netAfterFee(777, 0) == 777, "zero fee");
        require(quote.netAfterFee(777, 10_000) == 0, "full fee");
        require(quote.netAfterFee(999, 333) == 966, "rounding");
    }

    function testRejectsTooLargeFee() public {
        try quote.netAfterFee(1, 10_001) returns (uint256) {
            revert("accepted invalid fee");
        } catch {}
    }
}
