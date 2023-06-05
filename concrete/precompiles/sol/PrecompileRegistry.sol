// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity ^0.8.0;

struct Version {
    uint256 major;
    uint256 minor;
    uint256 patch;
}

struct FrameworkMetadata {
    string name;
    Version version;
    string source;
}

struct PrecompileMetadata {
    address addr;
    string name;
    Version version;
    string author;
    string description;
    string source;
    string ABI;
}

interface PrecompileRegistry {
    function getFramework()
        external
        view
        returns (FrameworkMetadata memory);

    function getPrecompile(
        address _addr
    ) external view returns (PrecompileMetadata memory);

    function getPrecompileByName(
        string memory _name
    ) external view returns (address);

    function getPrecompiledAddresses() external view returns (address[] memory);

    function getPrecompiles()
        external
        view
        returns (PrecompileMetadata[] memory);
}
