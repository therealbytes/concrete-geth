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

package main

import (
	cc_api "github.com/ethereum/go-ethereum/concrete/api"
	"github.com/ethereum/go-ethereum/tinygo"
)

type stub struct{}

func (pc *stub) MutatesStorage(input []byte) bool {
	return true
}

func (pc *stub) RequiredGas(input []byte) uint64 {
	return 0
}

func (pc *stub) Finalise(api cc_api.API) error {
	return nil
}

func (pc *stub) Commit(api cc_api.API) error {
	return nil
}

func (pc *stub) Run(api cc_api.API, input []byte) ([]byte, error) {
	return nil, nil
}

func init() {
	tinygo.WasmWrap(&stub{}, true)
}

// main is REQUIRED for TinyGo to compile to Wasm
func main() {}
