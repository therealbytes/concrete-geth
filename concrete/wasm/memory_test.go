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

package wasm

import (
	"context"
	_ "embed"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/concrete/wasm/bridge"
	"github.com/ethereum/go-ethereum/concrete/wasm/bridge/host"
	"github.com/stretchr/testify/require"
)

type mockMemory []byte

func newMockMemory() bridge.Memory {
	return &mockMemory{}
}

func (memory *mockMemory) Read(pointer bridge.MemPointer) []byte {
	if pointer.IsNull() {
		return []byte{}
	}
	offset, size := pointer.Unpack()
	if offset+size > uint32(len(*memory)) {
		panic("out of memory")
	}
	return (*memory)[offset : offset+size]
}

func (memory *mockMemory) Write(data []byte) bridge.MemPointer {
	size := len(data)
	if size == 0 {
		return bridge.NullPointer
	}
	offset := len(*memory)
	*memory = append(*memory, data...)
	var pointer bridge.MemPointer
	pointer.Pack(uint32(offset), uint32(size))
	return pointer
}

var _ bridge.Memory = (*mockMemory)(nil)

type mockAllocator struct{}

func newMockAllocator() bridge.Allocator {
	return &mockAllocator{}
}

func (a *mockAllocator) Malloc(size uint32) bridge.MemPointer { return bridge.NullPointer }
func (a *mockAllocator) Free(pointer bridge.MemPointer)       {}
func (a *mockAllocator) Prune()                               {}

func testMemoryReadWrite(t *testing.T, memory bridge.Memory) {
	data := []byte{1, 2, 3, 4, 5}
	ptr := memory.Write(data)
	require.False(t, ptr.IsNull())
	readData := memory.Read(ptr)
	require.Equal(t, data, readData)
}

func testMemoryPutGetValues(t *testing.T, memory bridge.Memory) {
	// Test PutValue and GetValue
	value := []byte{0x01, 0x02, 0x03}
	pointer := bridge.PutValue(memory, value)
	require.NotEqual(t, bridge.NullPointer, pointer)
	result := bridge.GetValue(memory, pointer)
	require.Equal(t, value, result)

	// Test PutValues and GetValues
	values := [][]byte{{0x01, 0x02}, {0x03, 0x04}, {0x05, 0x06, 0x07}}
	pointer = bridge.PutValues(memory, values)
	require.NotEqual(t, bridge.NullPointer, pointer)
	resultValues := bridge.GetValues(memory, pointer)
	require.Equal(t, values, resultValues)

	// Test PutValues with empty slice
	pointer = bridge.PutValues(memory, [][]byte{})
	require.Equal(t, bridge.NullPointer, pointer)

	// Test GetValues with null pointer
	resultValues = bridge.GetValues(memory, bridge.NullPointer)
	require.Equal(t, [][]byte{}, resultValues)
}

func testMemoryPutGetArgs(t *testing.T, memory bridge.Memory) {
	// Test PutArgs and GetArgs
	args := [][]byte{{0x01, 0x02}, {0x03, 0x04}, {0x05, 0x06, 0x07}}
	pointer := bridge.PutArgs(memory, args)
	require.NotEqual(t, bridge.NullPointer, pointer)
	resultArgs := bridge.GetArgs(memory, pointer)
	require.Equal(t, args, resultArgs)

	// Test PutArgs with empty slice
	pointer = bridge.PutArgs(memory, [][]byte{})
	require.Equal(t, bridge.NullPointer, pointer)

	// Test GetArgs with null pointer
	resultArgs = bridge.GetArgs(memory, bridge.NullPointer)
	require.Equal(t, [][]byte{}, resultArgs)
}

func testMemoryPutGetReturn(t *testing.T, memory bridge.Memory) {
	// Test PutReturn and GetReturn
	retValues := [][]byte{{0x01, 0x02}, {0x03, 0x04}, {0x05, 0x06, 0x07}}
	pointer := bridge.PutReturn(memory, retValues)
	require.NotEqual(t, bridge.NullPointer, pointer)
	resultRetValues := bridge.GetReturn(memory, pointer)
	require.Equal(t, retValues, resultRetValues)

	// Test PutReturn with empty slice
	pointer = bridge.PutReturn(memory, [][]byte{})
	require.Equal(t, bridge.NullPointer, pointer)

	// Test GetReturn with null pointer
	resultRetValues = bridge.GetReturn(memory, bridge.NullPointer)
	require.Equal(t, [][]byte{}, resultRetValues)
}

func testMemoryPutGetReturnWithError(t *testing.T, memory bridge.Memory) {
	// Test with success
	retValues := [][]byte{{0x01, 0x02}, {0x03, 0x04}, {0x05, 0x06, 0x07}}
	retPointer := bridge.PutReturnWithError(memory, retValues, nil)
	retValuesGot, err := bridge.GetReturnWithError(memory, retPointer)
	require.NoError(t, err)
	require.Equal(t, retValues, retValuesGot)

	// Test with error
	retErr := errors.New("some error")
	retPointer = bridge.PutReturnWithError(memory, retValues, retErr)
	retValuesGot, err = bridge.GetReturnWithError(memory, retPointer)
	require.EqualError(t, err, retErr.Error())
	require.Equal(t, retValues, retValuesGot)
}

func TestMockMemoryReadWrite(t *testing.T) {
	memory := newMockMemory()
	testMemoryReadWrite(t, memory)
}

func TestPutGetValues(t *testing.T) {
	memory := newMockMemory()
	testMemoryPutGetValues(t, memory)
}

func TestPutGetArgs(t *testing.T) {
	memory := newMockMemory()
	testMemoryPutGetArgs(t, memory)
}

func TestPutGetReturn(t *testing.T) {
	memory := newMockMemory()
	testMemoryPutGetReturn(t, memory)
}

func TestPutGetReturnWithError(t *testing.T) {
	memory := newMockMemory()
	testMemoryPutGetReturnWithError(t, memory)
}

//go:embed bin/blank.wasm
var blankCode []byte

func newTestMemory() (bridge.Memory, bridge.Allocator) {
	hostConfig := newHostConfig()
	mod, _, err := newModule(hostConfig, blankCode)
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	return host.NewMemory(ctx, mod)
}

func TestWasmMemoryReadWrite(t *testing.T) {
	memory, _ := newTestMemory()
	testMemoryReadWrite(t, memory)
}

func TestWasmMemoryFree(t *testing.T) {
	memory, alloc := newTestMemory()
	data := []byte{1, 2, 3, 4, 5}
	ptr := memory.Write(data)
	alloc.Free(ptr)
	require.Panics(t, func() { alloc.Free(ptr) })
}

func TestWasmMemoryPrune(t *testing.T) {
	memory, alloc := newTestMemory()
	data := []byte{1, 2, 3, 4, 5}
	ptr1 := memory.Write(data)
	ptr2 := memory.Write(data)
	alloc.Prune()
	require.Panics(t, func() { alloc.Free(ptr1) })
	require.Panics(t, func() { alloc.Free(ptr2) })
}

func TestWasmMemoryPutGetValues(t *testing.T) {
	memory, _ := newTestMemory()
	testMemoryPutGetValues(t, memory)
}

func TestWasmMemoryPutGetArgs(t *testing.T) {
	memory, _ := newTestMemory()
	testMemoryPutGetArgs(t, memory)
}

func TestWasmMemoryPutGetReturn(t *testing.T) {
	memory, _ := newTestMemory()
	testMemoryPutGetReturn(t, memory)
}

func TestWasmMemoryPutGetReturnWithError(t *testing.T) {
	memory, _ := newTestMemory()
	testMemoryPutGetReturnWithError(t, memory)
}
