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
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/concrete/wasm/bridge"
	"github.com/stretchr/testify/require"
)

func TestPutGetValues(t *testing.T) {
	mem := NewMockMemory()

	// Test PutValue and GetValue
	value := []byte{0x01, 0x02, 0x03}
	pointer := PutValue(mem, value)
	require.NotEqual(t, bridge.NullPointer, pointer)
	result := GetValue(mem, pointer)
	require.Equal(t, value, result)

	// Test PutValues and GetValues
	values := [][]byte{{0x01, 0x02}, {0x03, 0x04}, {0x05, 0x06, 0x07}}
	pointer = PutValues(mem, values)
	require.NotEqual(t, bridge.NullPointer, pointer)
	resultValues := GetValues(mem, pointer)
	require.Equal(t, values, resultValues)

	// Test PutValues with empty slice
	pointer = PutValues(mem, [][]byte{})
	require.Equal(t, bridge.NullPointer, pointer)

	// Test GetValues with null pointer
	resultValues = GetValues(mem, bridge.NullPointer)
	require.Equal(t, [][]byte{}, resultValues)
}

func TestPutGetArgs(t *testing.T) {
	mem := NewMockMemory()

	// Test PutArgs and GetArgs
	args := [][]byte{{0x01, 0x02}, {0x03, 0x04}, {0x05, 0x06, 0x07}}
	pointer := PutArgs(mem, args)
	require.NotEqual(t, bridge.NullPointer, pointer)
	resultArgs := GetArgs(mem, pointer)
	require.Equal(t, args, resultArgs)

	// Test PutArgs with empty slice
	pointer = PutArgs(mem, [][]byte{})
	require.Equal(t, bridge.NullPointer, pointer)

	// Test GetArgs with null pointer
	resultArgs = GetArgs(mem, bridge.NullPointer)
	require.Equal(t, [][]byte{}, resultArgs)
}

func TestPutGetReturn(t *testing.T) {
	mem := NewMockMemory()

	// Test PutReturn and GetReturn
	retValues := [][]byte{{0x01, 0x02}, {0x03, 0x04}, {0x05, 0x06, 0x07}}
	pointer := PutReturn(mem, retValues)
	require.NotEqual(t, bridge.NullPointer, pointer)
	resultRetValues := GetReturn(mem, pointer)
	require.Equal(t, retValues, resultRetValues)

	// Test PutReturn with empty slice
	pointer = PutReturn(mem, [][]byte{})
	require.Equal(t, bridge.NullPointer, pointer)

	// Test GetReturn with null pointer
	resultRetValues = GetReturn(mem, bridge.NullPointer)
	require.Equal(t, [][]byte{}, resultRetValues)
}

func TestPutGetReturnWithError(t *testing.T) {
	mem := NewMockMemory()

	// Test with success
	retValues := [][]byte{{0x01, 0x02}, {0x03, 0x04}, {0x05, 0x06, 0x07}}
	retPointer := PutReturnWithError(mem, retValues, nil)
	retValuesGot, err := GetReturnWithError(mem, retPointer)
	require.NoError(t, err)
	require.Equal(t, retValues, retValuesGot)

	// Test with error
	retErr := errors.New("some error")
	retPointer = PutReturnWithError(mem, retValues, retErr)
	retValuesGot, err = GetReturnWithError(mem, retPointer)
	require.EqualError(t, err, retErr.Error())
	require.Equal(t, retValues, retValuesGot)
}
