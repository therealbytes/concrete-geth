// Copyright 2023 The concrete-geth Authors
//
// The concrete-geth library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The concrete library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the concrete library. If not, see <http://www.gnu.org/licenses/>.

package tinygo

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/concrete/api"
	"github.com/ethereum/go-ethereum/concrete/wasm/bridge"
	"github.com/ethereum/go-ethereum/concrete/wasm/bridge/wasm"
	"github.com/ethereum/go-ethereum/tinygo/infra"
)

var precompile api.Precompile
var precompileConfig WasmConfig
var precompileAddress common.Address

type WasmConfig struct {
	IsPure       bool
	CacheProxies bool
}

func (c WasmConfig) cacheProxies() bool {
	return c.CacheProxies && !c.IsPure
}

var DefaultConfig = WasmConfig{
	IsPure:       false,
	CacheProxies: false,
}

func WasmWrap(pc api.Precompile) {
	precompile = pc
	precompileConfig = DefaultConfig
}

func WasmWrapWithConfig(pc api.Precompile, config WasmConfig) {
	precompile = pc
	precompileConfig = config
}

// Note: This uses a uint64 instead of two result values for compatibility with
// WebAssembly 1.0.

//go:wasm-module env
//export concrete_EvmCaller
func _evmCaller(pointer uint64) uint64

func evmCaller(pointer uint64) uint64 {
	return _evmCaller(pointer)
}

//go:wasm-module env
//export concrete_StateDBCaller
func _stateDBCaller(pointer uint64) uint64

func stateDBCaller(pointer uint64) uint64 {
	return _stateDBCaller(pointer)
}

//go:wasm-module env
//export concrete_AddressCaller
func _addressCaller(pointer uint64) uint64

func addressCaller() common.Address {
	address := wasm.Call_BytesArr_Bytes(infra.Memory, infra.Allocator, func(pointer uint64) uint64 { return _addressCaller(pointer) }, nil)
	return common.BytesToAddress(address)
}

func getAddress() common.Address {
	if precompileAddress == (common.Address{}) {
		precompileAddress = addressCaller()
	}
	return precompileAddress
}

func newAPI() api.API {
	var statedb api.StateDB
	if precompileConfig.cacheProxies() {
		statedb = wasm.NewCachedProxyStateDB(infra.Memory, infra.Allocator, stateDBCaller)
	} else {
		statedb = wasm.NewProxyStateDB(infra.Memory, infra.Allocator, stateDBCaller)
	}
	evm := wasm.NewProxyEVMWithStateDB(infra.Memory, infra.Allocator, evmCaller, statedb)
	address := getAddress()
	return api.New(evm, address)
}

func newCommitSafeStateAPI() api.API {
	var statedb api.StateDB
	if precompileConfig.cacheProxies() {
		statedb = wasm.NewCachedProxyStateDB(infra.Memory, infra.Allocator, stateDBCaller)
	} else {
		statedb = wasm.NewProxyStateDB(infra.Memory, infra.Allocator, stateDBCaller)
	}
	address := getAddress()
	return api.NewStateAPI(api.NewCommitSafeStateDB(statedb), address)
}

func commitProxyCache(API api.API) {
	if !precompileConfig.cacheProxies() {
		return
	}
	statedb := API.StateDB()
	if _statedb, ok := statedb.(*api.CommitSafeStateDB); ok {
		statedb = _statedb.StateDB
	}
	if proxy, ok := statedb.(*wasm.CachedProxyStateDB); ok {
		proxy.Commit()
	}
}

//export concrete_IsPure
func isPure() uint64 {
	if precompileConfig.IsPure {
		return 1
	} else {
		return 0
	}
}

//export concrete_MutatesStorage
func mutatesStorage(pointer uint64) uint64 {
	input := bridge.GetValue(infra.Memory, bridge.MemPointer(pointer))
	if precompile.MutatesStorage(input) {
		return 1
	} else {
		return 0
	}
}

//export concrete_RequiredGas
func requiredGas(pointer uint64) uint64 {
	input := bridge.GetValue(infra.Memory, bridge.MemPointer(pointer))
	gas := precompile.RequiredGas(input)
	return uint64(gas)
}

//export concrete_Finalise
func finalise() uint64 {
	API := newCommitSafeStateAPI()
	precompile.Finalise(API)
	commitProxyCache(API)
	return bridge.NullPointer.Uint64()
}

//export concrete_Commit
func commit() uint64 {
	API := newCommitSafeStateAPI()
	precompile.Commit(API)
	commitProxyCache(API)
	return bridge.NullPointer.Uint64()
}

//export concrete_Run
func run(pointer uint64) uint64 {
	input := bridge.GetValue(infra.Memory, bridge.MemPointer(pointer))
	API := newAPI()
	output, err := precompile.Run(API, input)
	commitProxyCache(API)
	return bridge.PutReturnWithError(infra.Memory, [][]byte{output}, err).Uint64()
}
