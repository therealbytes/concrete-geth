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
	"errors"

	"github.com/ethereum/go-ethereum/concrete/wasm/bridge"
)

type memory map[uint64][]byte

func NewMockMemory() Memory {
	return &memory{}
}

func (m memory) Ref(data []byte) bridge.MemPointer {
	if len(data) == 0 {
		return bridge.NullPointer
	}
	offset := uint32(len(m))
	size := uint32(len(data))
	var pointer bridge.MemPointer
	pointer.Pack(offset, size)
	m[pointer.Uint64()] = data
	return pointer
}

func (m memory) Deref(pointer bridge.MemPointer) []byte {
	if pointer.IsNull() {
		return []byte{}
	}
	data, ok := m[pointer.Uint64()]
	if !ok {
		panic(errors.New("invalid pointer"))
	}
	return data
}

var _ Memory = make(memory)
