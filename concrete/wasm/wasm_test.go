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
	_ "embed"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	cc_api "github.com/ethereum/go-ethereum/concrete/api"
	"github.com/ethereum/go-ethereum/concrete/lib"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

//go:embed bin/typical.wasm
var typicalCode []byte

//go:embed bin/benchmark.wasm
var benchmarkCode []byte

func TestStatefulPrecompile(t *testing.T) {
	address := common.HexToAddress("0x01")
	pc := NewWasmPrecompile(typicalCode, address)

	require.IsType(t, &wasmPrecompile{}, pc)

	runCounterKey := crypto.Keccak256Hash([]byte("typical.counter.0"))

	var wg sync.WaitGroup
	routines := 50
	iterations := 20
	wg.Add(routines)

	for ii := 0; ii < routines; ii++ {
		go func(ii int) {
			defer wg.Done()
			statedb := newTestStateDB()
			evm := newTestEVM(statedb)
			api := cc_api.New(evm, address)
			counter := lib.NewCounter(api.Persistent().NewReference(runCounterKey))
			require.Equal(t, uint64(0), counter.Get().Uint64())
			for jj := 0; jj < iterations; jj++ {
				data := []byte{byte(ii), byte(jj)}
				_, err := pc.Run(api, data)
				require.NoError(t, err)
				time.Sleep(time.Duration(rand.Intn(10)) * time.Microsecond)
			}
			require.Equal(t, uint64(iterations), counter.Get().Uint64())
		}(ii)
	}

	wg.Wait()
}

func newBenchmarkAPI(address common.Address) cc_api.API {
	statedb := newTestStateDB()
	evm := newTestEVM(statedb)
	api := cc_api.New(evm, address)
	return api
}

func BenchmarkNativeTypicalPrecompile(b *testing.B) {
	address := common.HexToAddress("0x01")
	api := newBenchmarkAPI(address)
	pc := lib.TypicalPrecompile{}
	preimage := crypto.Keccak256([]byte("hello world"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pc.Run(api, preimage)
	}
}

func BenchmarkWasmTypicalPrecompile(b *testing.B) {
	address := common.HexToAddress("0x01")
	api := newBenchmarkAPI(address)
	pc := NewWasmPrecompile(typicalCode, address)
	preimage := crypto.Keccak256([]byte("hello world"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pc.Run(api, preimage)
	}
}

func TestNativeBenchmarkPrecompile(t *testing.T) {
	address := common.HexToAddress("0x01")
	api := newBenchmarkAPI(address)
	pc := lib.BenchmarkPrecompile{}
	start := time.Now()
	pc.Run(api, nil)
	end := time.Now()
	fmt.Println("[external] BenchmarkPrecompile.Run", end.Sub(start).Nanoseconds(), "ns")
}

func TestWasmBenchmarkPrecompile(t *testing.T) {
	address := common.HexToAddress("0x01")
	api := newBenchmarkAPI(address)
	pc := NewWasmPrecompile(benchmarkCode, address)
	start := time.Now()
	pc.Run(api, nil)
	end := time.Now()
	fmt.Println("[external] BenchmarkPrecompile.Run", end.Sub(start).Nanoseconds(), "ns")
}
