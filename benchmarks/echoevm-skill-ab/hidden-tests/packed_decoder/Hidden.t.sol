// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {PackedDecoder} from "../src/PackedDecoder.sol";

contract HiddenPackedDecoderTest {
    PackedDecoder private decoder = new PackedDecoder();

    function testDecodesPackedValues() public view {
        address recipient = address(0x1234567890AbcdEF1234567890aBcdef12345678);
        uint96 amount = 0x0102030405060708090a0b0c;
        (uint96 actualAmount, address actualRecipient) = decoder.decode(abi.encodePacked(amount, recipient));
        require(actualAmount == amount, "amount");
        require(actualRecipient == recipient, "recipient");
    }

    function testDecodesZerosAndMax() public view {
        (uint96 zeroAmount, address zeroRecipient) = decoder.decode(abi.encodePacked(uint96(0), address(0)));
        require(zeroAmount == 0 && zeroRecipient == address(0), "zero");
        (uint96 maxAmount, address maxRecipient) = decoder.decode(
            abi.encodePacked(type(uint96).max, address(type(uint160).max))
        );
        require(maxAmount == type(uint96).max, "max amount");
        require(maxRecipient == address(type(uint160).max), "max recipient");
    }

    function testRejectsWrongLengths() public {
        try decoder.decode(new bytes(31)) returns (uint96, address) {
            revert("accepted short input");
        } catch {}
        try decoder.decode(new bytes(33)) returns (uint96, address) {
            revert("accepted long input");
        } catch {}
    }
}
