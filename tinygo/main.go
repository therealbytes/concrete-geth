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
	"reflect"
	"sync"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	cc_api "github.com/ethereum/go-ethereum/concrete/api"
	"github.com/ethereum/go-ethereum/concrete/wasm/bridge"
	"github.com/ethereum/go-ethereum/concrete/wasm/bridge/wasm"
)

// Note: This uses a uint64 instead of two result values for compatibility with
// WebAssembly 1.0.

var precompile cc_api.Precompile
var precompileIsPure bool

func WasmWrap(pc cc_api.Precompile, isPure bool) {
	precompile = pc
	precompileIsPure = isPure
}

type memory struct{}

func (m *memory) Ref(data []byte) bridge.MemPointer {
	if len(data) == 0 {
		return bridge.NullPointer
	}
	offset := uint32(uintptr(unsafe.Pointer(&data[0])))
	size := uint32(len(data))
	var pointer bridge.MemPointer
	pointer.Pack(offset, size)
	return pointer
}

func (m *memory) Deref(pointer bridge.MemPointer) []byte {
	if pointer.IsNull() {
		return []byte{}
	}
	offset, size := pointer.Unpack()
	return *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Data: uintptr(offset),
		//nolint:typecheck
		Len: uintptr(size),
		//nolint:typecheck
		Cap: uintptr(size),
	}))
}

var Memory wasm.Memory = &memory{}

func PutValue(value []byte) uint64 {
	return uint64(wasm.PutValue(Memory, value))
}

func GetValue(pointer uint64) []byte {
	return wasm.GetValue(Memory, bridge.MemPointer(pointer))
}

var allocs = sync.Map{}

//export concrete_Malloc
func Malloc(size uintptr) unsafe.Pointer {
	if size == 0 {
		return nil
	}
	buf := make([]byte, size)
	ptr := unsafe.Pointer(&buf[0])
	allocs.Store(uintptr(ptr), buf)
	return ptr
}

//export concrete_Free
func Free(ptr unsafe.Pointer) {
	if ptr == nil {
		return
	}
	if _, ok := allocs.Load(uintptr(ptr)); ok {
		allocs.Delete(uintptr(ptr))
	} else {
		panic("free: invalid pointer")
	}
}

//export concrete_Prune
func Prune() {
	allocs = sync.Map{}
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
	return common.BytesToAddress(Memory.Deref(bridge.MemPointer(pointer)))
}

func NewAPI() cc_api.API {
	evm := wasm.NewProxyEVM(Memory, EvmBridge, StateDBBridge)
	return cc_api.New(evm, AddressBridge())
}

func NewStateAPI() cc_api.API {
	statedb := wasm.NewProxyStateDB(Memory, StateDBBridge)
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
	input := GetValue(pointer)
	if precompile.MutatesStorage(input) {
		return 1
	} else {
		return 0
	}
}

//export concrete_RequiredGas
func RequiredGas(pointer uint64) uint64 {
	input := GetValue(pointer)
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
	input := GetValue(pointer)
	api := NewAPI()
	output, err := precompile.Run(api, input)
	return wasm.PutReturnWithError(Memory, [][]byte{output}, err).Uint64()
}
