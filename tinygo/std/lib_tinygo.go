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

	"github.com/ethereum/go-ethereum/tinygo"
)

//go:wasm-module env
//export concrete_LogBridge
func _LogBridge(pointer uint64) uint64

func Log(a ...any) uint64 {
	msg := fmt.Sprintln(a...)
	pointer := tinygo.PutValue([]byte(msg[:len(msg)-1]))
	return _LogBridge(pointer)
}
