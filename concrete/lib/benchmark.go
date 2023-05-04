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

package lib

import (
	"time"

	cc_api "github.com/ethereum/go-ethereum/concrete/api"
	"github.com/ethereum/go-ethereum/concrete/crypto"
	"github.com/ethereum/go-ethereum/tinygo/std"
)

type BenchmarkPrecompile struct {
	BlankPrecompile
}

func (pc *BenchmarkPrecompile) Run(api cc_api.API, input []byte) ([]byte, error) {

	runStart := std.Now()

	var runs int
	var startTime time.Time
	var endTime time.Time
	var timeNs int64

	hash := std.Keccak256(input)

	// std.Keccak256Hash

	runs = 10000
	startTime = std.Now()

	for i := 0; i < runs; i++ {
		hash = std.Keccak256(hash)
	}

	endTime = std.Now()
	timeNs = endTime.Sub(startTime).Nanoseconds()
	std.Print("std.Keccak256Hash", runs, "op", timeNs, "ns", int(timeNs)/runs, "ns/op")

	// crypto.Keccak256Hash

	runs = 10000
	startTime = std.Now()

	for i := 0; i < runs; i++ {
		hash = crypto.Keccak256(hash)
	}

	endTime = std.Now()
	timeNs = endTime.Sub(startTime).Nanoseconds()
	std.Print("crypto.Keccak256Hash", runs, "op", timeNs, "ns", int(timeNs)/runs, "ns/op")

	runEnd := std.Now()

	std.Print("[internal] BenchmarkPrecompile.Run", runEnd.Sub(runStart).Nanoseconds(), "ns")

	return nil, nil
}
