// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity ^0.8.0;

interface PreimageRegistry {
    function addPreimage(bytes memory preimage) external returns (bytes32);

    function hasPreimage(bytes32 _hash) external view returns (bool);

    function getPreimageSize(bytes32 _hash) external view returns (uint256);

    function getPreimage(
        uint256 size,
        bytes32 _hash
    ) external view returns (bytes memory);
}
