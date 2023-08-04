// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity ^0.8.0;

/*
This contract tests two things:
- The setup function is called before each test function in a clean environment
- `^test` functions are expected to succeed and `^testFail` functions are expected to fail
*/

contract TestFail {
    function setUp() external {}

    function testSuccess() external pure {
        revert();
    }

    function testFailure() external pure {
        return;
    }
}
