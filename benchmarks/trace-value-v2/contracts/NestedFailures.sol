// SPDX-License-Identifier: MIT
pragma solidity ^0.8.30;

contract Child {
    uint256 public value;

    function write(uint256 next) external returns (uint256) {
        value = next;
        return value;
    }

    function writeThenRevert(uint256 next) external {
        value = next;
        revert("child reverted");
    }
}

contract RevertingChild {
    constructor(uint256 marker) {
        assembly {
            sstore(0, marker)
        }
        revert("constructor reverted");
    }
}

contract Parent {
    Child public child;

    constructor() {
        child = new Child();
    }

    function ignoredChildRevert(uint256 next) external returns (bool ok) {
        (ok,) = address(child).call(abi.encodeCall(Child.writeThenRevert, (next)));
    }

    function swallowedCreateRevert(uint256 marker) external returns (address created) {
        try new RevertingChild(marker) returns (RevertingChild deployed) {
            return address(deployed);
        } catch {
            return address(0);
        }
    }

    function delegateWriteCorruptsParent(uint256 next) external returns (bool ok) {
        (ok,) = address(child).delegatecall(abi.encodeCall(Child.write, (next)));
    }
}

contract MathFaults {
    function wrongAverage(uint256[] calldata values) external pure returns (uint256) {
        uint256 total;
        for (uint256 index; index < values.length; ++index) {
            total += values[index];
        }
        return total / (values.length - 1);
    }
}
