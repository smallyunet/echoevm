// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Create2Factory, Child} from "../src/Create2Factory.sol";

contract HiddenCreate2FactoryTest {
    function testPredictionMatchesDeployment() public {
        Create2Factory factory = new Create2Factory();
        bytes32 salt = keccak256("hidden-one");
        address predicted = factory.predicted(salt, 42);
        address deployed = factory.deploy(salt, 42);
        require(predicted == deployed, "address mismatch");
        require(Child(deployed).value() == 42, "seed mismatch");
    }

    function testPredictionDependsOnConstructorArgs() public {
        Create2Factory factory = new Create2Factory();
        bytes32 salt = keccak256("hidden-two");
        require(factory.predicted(salt, 1) != factory.predicted(salt, 2), "seed omitted");
    }

    function testDeployAndCheck() public {
        Create2Factory factory = new Create2Factory();
        require(factory.deployAndCheck(bytes32(uint256(1234)), type(uint128).max), "check failed");
    }
}
