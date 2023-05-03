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

package std

import (
	"fmt"

	"github.com/ethereum/go-ethereum/concrete/crypto"
	"github.com/ethereum/go-ethereum/concrete/wasm/bridge/wasm"
	"github.com/ethereum/go-ethereum/tinygo/mem"
)

//go:wasm-module env
//export concrete_LogCaller
func _logCaller(pointer uint64) uint64

func Log(a ...any) {
	msg := []byte(fmt.Sprintln(a...))
	data := [][]byte{msg[:len(msg)-1]}
	wasm.Call_BytesArr_Bytes(mem.Memory, mem.Allocator, func(pointer uint64) uint64 { return _logCaller(pointer) }, data...)
}

var Keccak256 = crypto.ReimplementedKeccak256
var Keccak256Hash = crypto.ReimplementedKeccak256Hash
