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

	"github.com/ethereum/go-ethereum/concrete/wasm/bridge"
)

type Memory interface {
	Ref(data []byte) bridge.MemPointer
	Deref(pointer bridge.MemPointer) []byte
}

func PutValue(memory Memory, value []byte) bridge.MemPointer {
	return memory.Ref(value)
}

func GetValue(memory Memory, pointer bridge.MemPointer) []byte {
	return memory.Deref(pointer)
}

func PutValues(memory Memory, values [][]byte) bridge.MemPointer {
	if len(values) == 0 {
		return bridge.NullPointer
	}
	var pointers []bridge.MemPointer
	for _, v := range values {
		pointers = append(pointers, PutValue(memory, v))
	}
	packedPointers := bridge.PackPointers(pointers)
	return PutValue(memory, packedPointers)
}

func GetValues(memory Memory, pointer bridge.MemPointer) [][]byte {
	if pointer.IsNull() {
		return [][]byte{}
	}
	var values [][]byte
	valPointers := bridge.UnpackPointers(GetValue(memory, pointer))
	for _, p := range valPointers {
		values = append(values, GetValue(memory, p))
	}
	return values
}

func PutArgs(memory Memory, args [][]byte) bridge.MemPointer {
	return PutValues(memory, args)
}

func GetArgs(memory Memory, pointer bridge.MemPointer) [][]byte {
	return GetValues(memory, pointer)
}

func PutReturn(memory Memory, retValues [][]byte) bridge.MemPointer {
	return PutValues(memory, retValues)
}

func GetReturn(memory Memory, retPointer bridge.MemPointer) [][]byte {
	return GetValues(memory, retPointer)
}

func PutReturnWithError(memory Memory, retValues [][]byte, retErr error) bridge.MemPointer {
	if retErr == nil {
		errFlag := []byte{bridge.Err_Success}
		retValues = append([][]byte{errFlag}, retValues...)
	} else {
		errFlag := []byte{bridge.Err_Error}
		errMsg := []byte(retErr.Error())
		retValues = append([][]byte{errFlag, errMsg}, retValues...)
	}
	return PutReturn(memory, retValues)
}

func GetReturnWithError(memory Memory, retPointer bridge.MemPointer) ([][]byte, error) {
	retValues := GetReturn(memory, retPointer)
	if len(retValues) == 0 {
		return nil, nil
	}
	if retValues[0][0] == bridge.Err_Success {
		return retValues[1:], nil
	} else {
		return retValues[2:], errors.New(string(retValues[1]))
	}
}
