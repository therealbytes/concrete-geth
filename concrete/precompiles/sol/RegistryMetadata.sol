// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity ^0.8.0;

import "./PrecompileRegistry.sol";

address constant PrecompileRegistryAddress = address(
    0xCc00000000000000000000000000000000000000
);
address constant PreimageRegistryAddress = address(
    0xcC00000000000000000000000000000000000001
);
address constant BigPreimageRegistryAddress = address(
    0xCc00000000000000000000000000000000000002
);

string constant PrecompileRegistryName = "PrecompileRegistry";
string constant PreimageRegistryName = "PreimageRegistry";
string constant BigPreimageRegistryName = "BigPreimageRegistry";
