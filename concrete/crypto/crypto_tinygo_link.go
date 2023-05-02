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

//go:build tinygo && link_crypto

package crypto

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/concrete/wasm/bridge"
	tinygo_infra "github.com/ethereum/go-ethereum/tinygo/mem"
)

//go:wasm-module env
//export concrete_Keccak256Caller
func _Keccak256Caller(pointer uint64) uint64

func Keccak256(data ...[]byte) []byte {
	argsPointer := bridge.PutArgs(tinygo_infra.Memory, data)
	retPointer := bridge.MemPointer(_Keccak256Caller(argsPointer.Uint64()))
	retValue := bridge.GetValue(tinygo_infra.Memory, retPointer)
	tinygo_infra.Allocator.Free(retPointer)
	return retValue
}

func Keccak256Hash(data ...[]byte) (h common.Hash) {
	return common.BytesToHash(Keccak256(data...))
}
