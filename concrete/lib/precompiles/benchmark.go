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

package precompiles

import (
	"time"

	cc_api "github.com/ethereum/go-ethereum/concrete/api"
	"github.com/ethereum/go-ethereum/concrete/crypto"
	"github.com/ethereum/go-ethereum/concrete/lib"
	tinygo_lib "github.com/ethereum/go-ethereum/tinygo/lib"
)

// A precompiled used for benchmarking TinyGo WASM.

type BenchmarkPrecompile struct {
	lib.BlankPrecompile
}

func (pc *BenchmarkPrecompile) Run(api cc_api.API, input []byte) ([]byte, error) {

	runStart := tinygo_lib.Now()

	var runs int
	var startTime time.Time
	var endTime time.Time
	var timeNs int64

	hash := tinygo_lib.Keccak256(input)

	// tinygo_lib.Keccak256Hash

	runs = 10000
	startTime = tinygo_lib.Now()

	for i := 0; i < runs; i++ {
		hash = tinygo_lib.Keccak256(hash)
	}

	endTime = tinygo_lib.Now()
	timeNs = endTime.Sub(startTime).Nanoseconds()
	tinygo_lib.Print("tinygo_lib.Keccak256Hash", runs, "op", timeNs, "ns", int(timeNs)/runs, "ns/op")

	// crypto.Keccak256Hash

	runs = 10000
	startTime = tinygo_lib.Now()

	for i := 0; i < runs; i++ {
		hash = crypto.Keccak256(hash)
	}

	endTime = tinygo_lib.Now()
	timeNs = endTime.Sub(startTime).Nanoseconds()
	tinygo_lib.Print("crypto.Keccak256Hash", runs, "op", timeNs, "ns", int(timeNs)/runs, "ns/op")

	runEnd := tinygo_lib.Now()

	tinygo_lib.Print("BenchmarkPrecompile.Run", runEnd.Sub(runStart).Microseconds(), "μs")

	return nil, nil
}
