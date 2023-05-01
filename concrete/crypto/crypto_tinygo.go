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

//go:build tinygo

package crypto

import (
	"github.com/ethereum/go-ethereum/common"
	mem "github.com/ethereum/go-ethereum/tinygo/mem"
)

//go:wasm-module env
//export concrete_Keccak256Bridge
func _Keccak256Bridge(pointer uint64) uint64

func Keccak256(data ...[]byte) []byte {
	dataPtr := mem.PutValues(data)
	hashPtr := _Keccak256Bridge(dataPtr)
	return mem.GetValue(hashPtr)
}

func Keccak256Hash(data ...[]byte) (h common.Hash) {
	return common.BytesToHash(Keccak256(data...))
}
