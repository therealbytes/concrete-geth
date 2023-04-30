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
	"github.com/ethereum/go-ethereum/concrete/wasm/bridge/native"
	"github.com/stretchr/testify/require"
)

//go:embed bin/blank.wasm
var blankCode []byte

//go:embed bin/log.wasm
var logCode []byte

func TestReadWriteMemory(t *testing.T) {
	ctx := context.Background()
	mod, _, _ := newModule(ctx, &bridgeConfig{}, blankCode)

	data := []byte{1, 2, 3, 4, 5}
	ptr, err := native.WriteMemory(ctx, mod, data)
	require.NoError(t, err)
	require.False(t, ptr.IsNull())

	readData, err := native.ReadMemory(ctx, mod, ptr)
	require.NoError(t, err)
	require.Equal(t, data, readData)
}

func TestFreeMemory(t *testing.T) {
	ctx := context.Background()
	mod, _, _ := newModule(ctx, &bridgeConfig{}, blankCode)

	data := []byte{1, 2, 3, 4, 5}

	ptr, _ := native.WriteMemory(ctx, mod, data)

	err := native.FreeMemory(ctx, mod, ptr)
	require.NoError(t, err)

	err = native.FreeMemory(ctx, mod, ptr)
	require.Error(t, err)
}

func TestPruneMemory(t *testing.T) {
	ctx := context.Background()
	mod, _, _ := newModule(ctx, &bridgeConfig{}, blankCode)

	data := []byte{1, 2, 3, 4, 5}

	ptr1, _ := native.WriteMemory(ctx, mod, data)
	ptr2, _ := native.WriteMemory(ctx, mod, data)

	err := native.PruneMemory(ctx, mod)
	require.NoError(t, err)

	err = native.FreeMemory(ctx, mod, ptr1)
	require.Error(t, err)

	err = native.FreeMemory(ctx, mod, ptr2)
	require.Error(t, err)
}

func TestPutGetValues(t *testing.T) {
	ctx := context.Background()
	mod, _, _ := newModule(ctx, &bridgeConfig{}, blankCode)

	// Test PutValue and GetValue
	value := []byte{0x01, 0x02, 0x03}
	pointer := native.PutValue(ctx, mod, value)
	require.NotEqual(t, bridge.NullPointer, pointer)
	result := native.GetValue(ctx, mod, pointer)
	require.Equal(t, value, result)

	// Test PutValues and GetValues
	values := [][]byte{{0x01, 0x02}, {0x03, 0x04}, {0x05, 0x06, 0x07}}
	pointer = native.PutValues(ctx, mod, values)
	require.NotEqual(t, bridge.NullPointer, pointer)
	resultValues := native.GetValues(ctx, mod, pointer)
	require.Equal(t, values, resultValues)

	// Test PutValues with empty slice
	pointer = native.PutValues(ctx, mod, [][]byte{})
	require.Equal(t, bridge.NullPointer, pointer)

	// Test GetValues with null pointer
	resultValues = native.GetValues(ctx, mod, bridge.NullPointer)
	require.Equal(t, [][]byte{}, resultValues)
}

func TestPutGetArgs(t *testing.T) {
	ctx := context.Background()
	mod, _, _ := newModule(ctx, &bridgeConfig{}, blankCode)

	// Test PutArgs and GetArgs
	args := [][]byte{{0x01, 0x02}, {0x03, 0x04}, {0x05, 0x06, 0x07}}
	pointer := native.PutArgs(ctx, mod, args)
	require.NotEqual(t, bridge.NullPointer, pointer)
	resultArgs := native.GetArgs(ctx, mod, pointer)
	require.Equal(t, args, resultArgs)

	// Test PutArgs with empty slice
	pointer = native.PutArgs(ctx, mod, [][]byte{})
	require.Equal(t, bridge.NullPointer, pointer)

	// Test GetArgs with null pointer
	resultArgs = native.GetArgs(ctx, mod, bridge.NullPointer)
	require.Equal(t, [][]byte{}, resultArgs)
}

func TestPutGetReturn(t *testing.T) {
	ctx := context.Background()
	mod, _, _ := newModule(ctx, &bridgeConfig{}, blankCode)

	// Test PutReturn and GetReturn
	retValues := [][]byte{{0x01, 0x02}, {0x03, 0x04}, {0x05, 0x06, 0x07}}
	pointer := native.PutReturn(ctx, mod, retValues)
	require.NotEqual(t, bridge.NullPointer, pointer)
	resultRetValues := native.GetReturn(ctx, mod, pointer)
	require.Equal(t, retValues, resultRetValues)

	// Test PutReturn with empty slice
	pointer = native.PutReturn(ctx, mod, [][]byte{})
	require.Equal(t, bridge.NullPointer, pointer)

	// Test GetReturn with null pointer
	resultRetValues = native.GetReturn(ctx, mod, bridge.NullPointer)
	require.Equal(t, [][]byte{}, resultRetValues)
}

func TestPutGetReturnWithError(t *testing.T) {
	ctx := context.Background()
	mod, _, _ := newModule(ctx, &bridgeConfig{}, blankCode)

	// Test with success
	retValues := [][]byte{{0x01, 0x02}, {0x03, 0x04}, {0x05, 0x06, 0x07}}
	retPointer := native.PutReturnWithError(ctx, mod, retValues, nil)
	retValuesGot, err := native.GetReturnWithError(ctx, mod, retPointer)
	require.NoError(t, err)
	require.Equal(t, retValues, retValuesGot)

	// Test with error
	retErr := errors.New("some error")
	retPointer = native.PutReturnWithError(ctx, mod, retValues, retErr)
	retValuesGot, err = native.GetReturnWithError(ctx, mod, retPointer)
	require.EqualError(t, err, retErr.Error())
	require.Equal(t, retValues, retValuesGot)
}
