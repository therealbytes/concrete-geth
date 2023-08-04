// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity ^0.8.0;

/*
This contract tests two things:
- The setup function is called before each test function in a clean environment
- `^test` functions are expected to succeed and `^testFail` functions are expected to fail
*/

contract Test {
    uint256 public value;

    function setUp() external {
        if (value != 0) {
            revert("value is not 0");
        }
        value = 1;
        return;
    }

    function testSuccess() external {
        if (value != 1) {
            revert("value is not 1");
        }
        value = 2;
        return;
    }

    function testFailure() external {
        if (value != 1) {
            return;
        }
        value = 2;
        revert("test failure");
    }
}
