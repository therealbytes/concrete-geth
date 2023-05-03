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

func (mem *mockMemory) Read(pointer bridge.MemPointer) []byte {
	if pointer.IsNull() {
		return []byte{}
	}
	offset, size := pointer.Unpack()
	if offset+size > uint32(len(*mem)) {
		panic("out of memory")
	}
	return (*mem)[offset : offset+size]
}

func (mem *mockMemory) Write(data []byte) bridge.MemPointer {
	size := len(data)
	if size == 0 {
		return bridge.NullPointer
	}
	offset := len(*mem)
	*mem = append(*mem, data...)
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

func testMemoryReadWrite(t *testing.T, mem bridge.Memory) {
	data := []byte{1, 2, 3, 4, 5}
	ptr := mem.Write(data)
	require.False(t, ptr.IsNull())
	readData := mem.Read(ptr)
	require.Equal(t, data, readData)
}

func testMemoryPutGetValues(t *testing.T, mem bridge.Memory) {
	// Test PutValue and GetValue
	value := []byte{0x01, 0x02, 0x03}
	pointer := bridge.PutValue(mem, value)
	require.NotEqual(t, bridge.NullPointer, pointer)
	result := bridge.GetValue(mem, pointer)
	require.Equal(t, value, result)

	// Test PutValues and GetValues
	values := [][]byte{{0x01, 0x02}, {0x03, 0x04}, {0x05, 0x06, 0x07}}
	pointer = bridge.PutValues(mem, values)
	require.NotEqual(t, bridge.NullPointer, pointer)
	resultValues := bridge.GetValues(mem, pointer)
	require.Equal(t, values, resultValues)

	// Test PutValues with empty slice
	pointer = bridge.PutValues(mem, [][]byte{})
	require.Equal(t, bridge.NullPointer, pointer)

	// Test GetValues with null pointer
	resultValues = bridge.GetValues(mem, bridge.NullPointer)
	require.Equal(t, [][]byte{}, resultValues)
}

func testMemoryPutGetArgs(t *testing.T, mem bridge.Memory) {
	// Test PutArgs and GetArgs
	args := [][]byte{{0x01, 0x02}, {0x03, 0x04}, {0x05, 0x06, 0x07}}
	pointer := bridge.PutArgs(mem, args)
	require.NotEqual(t, bridge.NullPointer, pointer)
	resultArgs := bridge.GetArgs(mem, pointer)
	require.Equal(t, args, resultArgs)

	// Test PutArgs with empty slice
	pointer = bridge.PutArgs(mem, [][]byte{})
	require.Equal(t, bridge.NullPointer, pointer)

	// Test GetArgs with null pointer
	resultArgs = bridge.GetArgs(mem, bridge.NullPointer)
	require.Equal(t, [][]byte{}, resultArgs)
}

func testMemoryPutGetReturn(t *testing.T, mem bridge.Memory) {
	// Test PutReturn and GetReturn
	retValues := [][]byte{{0x01, 0x02}, {0x03, 0x04}, {0x05, 0x06, 0x07}}
	pointer := bridge.PutReturn(mem, retValues)
	require.NotEqual(t, bridge.NullPointer, pointer)
	resultRetValues := bridge.GetReturn(mem, pointer)
	require.Equal(t, retValues, resultRetValues)

	// Test PutReturn with empty slice
	pointer = bridge.PutReturn(mem, [][]byte{})
	require.Equal(t, bridge.NullPointer, pointer)

	// Test GetReturn with null pointer
	resultRetValues = bridge.GetReturn(mem, bridge.NullPointer)
	require.Equal(t, [][]byte{}, resultRetValues)
}

func testMemoryPutGetReturnWithError(t *testing.T, mem bridge.Memory) {
	// Test with success
	retValues := [][]byte{{0x01, 0x02}, {0x03, 0x04}, {0x05, 0x06, 0x07}}
	retPointer := bridge.PutReturnWithError(mem, retValues, nil)
	retValuesGot, err := bridge.GetReturnWithError(mem, retPointer)
	require.NoError(t, err)
	require.Equal(t, retValues, retValuesGot)

	// Test with error
	retErr := errors.New("some error")
	retPointer = bridge.PutReturnWithError(mem, retValues, retErr)
	retValuesGot, err = bridge.GetReturnWithError(mem, retPointer)
	require.EqualError(t, err, retErr.Error())
	require.Equal(t, retValues, retValuesGot)
}

func TestMockMemoryReadWrite(t *testing.T) {
	mem := newMockMemory()
	testMemoryReadWrite(t, mem)
}

func TestPutGetValues(t *testing.T) {
	mem := newMockMemory()
	testMemoryPutGetValues(t, mem)
}

func TestPutGetArgs(t *testing.T) {
	mem := newMockMemory()
	testMemoryPutGetArgs(t, mem)
}

func TestPutGetReturn(t *testing.T) {
	mem := newMockMemory()
	testMemoryPutGetReturn(t, mem)
}

func TestPutGetReturnWithError(t *testing.T) {
	mem := newMockMemory()
	testMemoryPutGetReturnWithError(t, mem)
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
	mem, _ := newTestMemory()
	testMemoryReadWrite(t, mem)
}

func TestWasmMemoryFree(t *testing.T) {
	mem, alloc := newTestMemory()
	data := []byte{1, 2, 3, 4, 5}
	ptr := mem.Write(data)
	alloc.Free(ptr)
	require.Panics(t, func() { alloc.Free(ptr) })
}

func TestWasmMemoryPrune(t *testing.T) {
	mem, alloc := newTestMemory()
	data := []byte{1, 2, 3, 4, 5}
	ptr1 := mem.Write(data)
	ptr2 := mem.Write(data)
	alloc.Prune()
	require.Panics(t, func() { alloc.Free(ptr1) })
	require.Panics(t, func() { alloc.Free(ptr2) })
}

func TestWasmMemoryPutGetValues(t *testing.T) {
	mem, _ := newTestMemory()
	testMemoryPutGetValues(t, mem)
}

func TestWasmMemoryPutGetArgs(t *testing.T) {
	mem, _ := newTestMemory()
	testMemoryPutGetArgs(t, mem)
}

func TestWasmMemoryPutGetReturn(t *testing.T) {
	mem, _ := newTestMemory()
	testMemoryPutGetReturn(t, mem)
}

func TestWasmMemoryPutGetReturnWithError(t *testing.T) {
	mem, _ := newTestMemory()
	testMemoryPutGetReturnWithError(t, mem)
}
