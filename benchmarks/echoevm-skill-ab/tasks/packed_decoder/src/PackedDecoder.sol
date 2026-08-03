// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

contract PackedDecoder {
    function decode(bytes calldata payload) external pure returns (uint96 amount, address recipient) {
        require(payload.length == 32, "invalid payload");
        assembly {
            let word := calldataload(payload.offset)
            recipient := shr(96, word)
            amount := and(word, 0xffffffffffffffffffffffff)
        }
    }
}
