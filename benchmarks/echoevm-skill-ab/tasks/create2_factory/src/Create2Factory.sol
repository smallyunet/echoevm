// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

contract Child {
    uint256 public immutable value;

    constructor(uint256 seed) {
        value = seed;
    }
}

contract Create2Factory {
    function predicted(bytes32 salt, uint256 seed) public view returns (address) {
        seed;
        bytes32 initCodeHash = keccak256(type(Child).creationCode);
        return address(uint160(uint256(keccak256(abi.encodePacked(bytes1(0xff), address(this), salt, initCodeHash)))));
    }

    function deploy(bytes32 salt, uint256 seed) public returns (address child) {
        child = address(new Child{salt: salt}(seed));
    }

    function deployAndCheck(bytes32 salt, uint256 seed) external returns (bool) {
        address expected = predicted(salt, seed);
        address child = deploy(salt, seed);
        return child == expected && Child(child).value() == seed;
    }
}
