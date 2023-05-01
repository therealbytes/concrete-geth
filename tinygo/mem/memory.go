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

package mem

import (
	"reflect"
	"sync"
	"unsafe"

	"github.com/ethereum/go-ethereum/concrete/wasm/bridge"
	"github.com/ethereum/go-ethereum/concrete/wasm/bridge/wasm"
)

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
