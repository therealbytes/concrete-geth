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

func WasmWrap(pc cc_api.Precompile, isPure bool) {
	precompile = pc
	precompileIsPure = isPure
}

//go:wasm-module env
//export concrete_EvmBridge
func _EvmBridge(pointer uint64) uint64

func EvmBridge(pointer uint64) uint64 {
	return _EvmBridge(pointer)
}

//go:wasm-module env
//export concrete_StateDBBridge
func _StateDBBridge(pointer uint64) uint64

func StateDBBridge(pointer uint64) uint64 {
	return _StateDBBridge(pointer)
}

//go:wasm-module env
//export concrete_AddressBridge
func _AddressBridge(pointer uint64) uint64

func AddressBridge() common.Address {
	pointer := _AddressBridge(0)
	address := mem.GetValue(pointer)
	return common.BytesToAddress(address)
}

func NewAPI() cc_api.API {
	evm := wasm.NewProxyEVM(mem.Memory, EvmBridge, StateDBBridge)
	return cc_api.New(evm, AddressBridge())
}

func NewStateAPI() cc_api.API {
	statedb := wasm.NewProxyStateDB(mem.Memory, StateDBBridge)
	address := AddressBridge()
	return cc_api.NewStateAPI(cc_api.NewCommitSafeStateDB(statedb), address)
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
	input := mem.GetValue(pointer)
	if precompile.MutatesStorage(input) {
		return 1
	} else {
		return 0
	}
}

//export concrete_RequiredGas
func RequiredGas(pointer uint64) uint64 {
	input := mem.GetValue(pointer)
	gas := precompile.RequiredGas(input)
	return uint64(gas)
}

//export concrete_Finalise
func Finalise() uint64 {
	precompile.Finalise(NewStateAPI())
	return bridge.NullPointer.Uint64()
}

//export concrete_Commit
func Commit() uint64 {
	precompile.Commit(NewStateAPI())
	return bridge.NullPointer.Uint64()
}

//export concrete_Run
func Run(pointer uint64) uint64 {
	input := mem.GetValue(pointer)
	api := NewAPI()
	output, err := precompile.Run(api, input)
	return mem.PutReturnWithError([][]byte{output}, err)
}
