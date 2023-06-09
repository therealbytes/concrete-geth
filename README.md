# Concrete

Concrete is a framework for building application-specific rollups on the OP Stack.

Concrete blockchains are general-purpose EVM rollups that have been _enhanced_ at the protocol level to meet the needs of a specific use case, like expensive cryptography, spatial indexing, or complex digital physics.

With the Concrete API, you can build a rollup that fits your needs, without having to deal with the low-level complexity of forking an Ethereum client.

With Concrete, you can:

- Write app-specific code in any language that compiles to WASM.

- Add supercharged, stateful precompiles to the EVM using common structures like maps, arrays, and sets.

- Define precompile interfaces in Solidity and have all the ABI encoding handled for you.

- Extend the structure of storage beyond the MPT (Merkle Patricia Tree) to reap huge performance gains (see [below](#quadrosol)).

- Take advantage of chain introspection to make your app-specific contracts compatible with non-enhanced chains.

- Use existing EVM-compatible tools like MUD, Foundry, Otterscan, and more.

## Getting started

To get started, check out the [Concrete basic template](https://github.com/therealbytes/concrete-template).

[Join our Discord](https://discord.gg/xW4unzxbqT) to get support and connect with the community.

## Built on Concrete

### [Mudtendo](https://github.com/therealbytes/mudtendo)

> An Autonomous World where participants use a 1985 Nintendo console to create stories together.

Mudtendo is a MUD application running on a [Concrete app-chain](https://github.com/therealbytes/neschain) with a built-in emulator for the NES (an old Nintendo game console).

### [Quadrosol](https://github.com/therealbytes/quadrosol/tree/concrete)

The Quadrosol Concrete app-chain has 2D spatial indexing built-in, enabling fast spatial queries for games and other applications.

By storing indexing nodes outside the MPT and running natively in Go, it outperforms a Solidity implementation by over 6x in speed and 70% in database footprint!

## The codebase

This repo is built on top of [op-geth](https://github.com/ethereum-optimism/op-geth), the default execution-engine for the OP Stack, which itself is built on top of [geth](https://github.com/ethereum/go-ethereum).

Find the diff with op-geth [here](https://github.com/therealbytes/concrete-geth/compare/optimism-fork-point...develop), the most notable changes are in `core/state/statedb` and `core/vm/evm.go`.

The framework-specific code is under `/concrete` and the bindings to compile TinyGo precompiles to WASM are under `/tinygo`.
