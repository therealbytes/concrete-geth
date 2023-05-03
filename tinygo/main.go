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
	cc_api "github.com/ethereum/go-ethereum/concrete/api"
	"github.com/ethereum/go-ethereum/concrete/wasm/bridge"
	"github.com/ethereum/go-ethereum/concrete/wasm/bridge/wasm"
	"github.com/ethereum/go-ethereum/tinygo/mem"
)

// Note: This uses a uint64 instead of two result values for compatibility with
// WebAssembly 1.0.

var precompile cc_api.Precompile
var precompileIsPure bool
var precompileAddress common.Address

func WasmWrap(pc cc_api.Precompile, isPure bool) {
	precompile = pc
	precompileIsPure = isPure
}

//go:wasm-module env
//export concrete_EvmCaller
func _EvmCaller(pointer uint64) uint64

func EvmCaller(pointer uint64) uint64 {
	return _EvmCaller(pointer)
}

//go:wasm-module env
//export concrete_StateDBCaller
func _StateDBCaller(pointer uint64) uint64

func StateDBCaller(pointer uint64) uint64 {
	return _StateDBCaller(pointer)
}

//go:wasm-module env
//export concrete_AddressCaller
func _AddressCaller(pointer uint64) uint64

func AddressCaller() common.Address {
	address := wasm.Call_BytesArr_Bytes(mem.Memory, mem.Allocator, func(pointer uint64) uint64 { return _AddressCaller(pointer) }, nil)
	return common.BytesToAddress(address)
}

func GetAddress() common.Address {
	if precompileAddress == (common.Address{}) {
		precompileAddress = AddressCaller()
	}
	return precompileAddress
}

func NewAPI() cc_api.API {
	evm := wasm.NewCachedProxyEVM(mem.Memory, mem.Allocator, EvmCaller, StateDBCaller)
	address := GetAddress()
	return cc_api.New(evm, address)
}

func NewCommitSafeStateAPI() cc_api.API {
	statedb := wasm.NewCachedProxyStateDB(mem.Memory, mem.Allocator, StateDBCaller)
	address := GetAddress()
	return cc_api.NewStateAPI(cc_api.NewCommitSafeStateDB(statedb), address)
}

func CommitProxyCache(api cc_api.API) {
	evm := api.EVM()
	if proxy, ok := evm.(*wasm.CachedProxyEVM); ok {
		proxy.Commit()
	}
}

//export concrete_IsPure
func IsPure() uint64 {
	if precompileIsPure {
		return 1
	} else {
		return 0
	}
}

//export concrete_MutatesStorage
func MutatesStorage(pointer uint64) uint64 {
	input := bridge.GetValue(mem.Memory, bridge.MemPointer(pointer))
	if precompile.MutatesStorage(input) {
		return 1
	} else {
		return 0
	}
}

//export concrete_RequiredGas
func RequiredGas(pointer uint64) uint64 {
	input := bridge.GetValue(mem.Memory, bridge.MemPointer(pointer))
	gas := precompile.RequiredGas(input)
	return uint64(gas)
}

//export concrete_Finalise
func Finalise() uint64 {
	api := NewCommitSafeStateAPI()
	defer CommitProxyCache(api)
	precompile.Finalise(api)
	return bridge.NullPointer.Uint64()
}

//export concrete_Commit
func Commit() uint64 {
	api := NewCommitSafeStateAPI()
	defer CommitProxyCache(api)
	precompile.Commit(api)
	return bridge.NullPointer.Uint64()
}

//export concrete_Run
func Run(pointer uint64) uint64 {
	input := bridge.GetValue(mem.Memory, bridge.MemPointer(pointer))
	api := NewAPI()
	defer CommitProxyCache(api)
	output, err := precompile.Run(api, input)
	return bridge.PutReturnWithError(mem.Memory, [][]byte{output}, err).Uint64()
}
