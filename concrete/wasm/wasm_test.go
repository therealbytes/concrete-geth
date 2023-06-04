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
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/concrete/api"
	"github.com/ethereum/go-ethereum/concrete/lib"
	"github.com/ethereum/go-ethereum/concrete/lib/precompiles"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/typical.wasm
var typicalCode []byte

//go:embed testdata/benchmark.wasm
var benchmarkCode []byte

var typicalImplementations = []struct {
	name string
	pc   api.Precompile
}{
	{"Native", &precompiles.TypicalPrecompile{}},
	{"Wasm", NewWasmPrecompile(typicalCode)},
}

func TestPrecompile(t *testing.T) {
	var (
		r             = require.New(t)
		runCounterKey = crypto.Keccak256Hash([]byte("typical.counter.0"))
	)
	for _, impl := range typicalImplementations {
		t.Run(impl.name, func(t *testing.T) {
			var wg sync.WaitGroup
			routines := 50
			iterations := 20
			wg.Add(routines)
			for ii := 0; ii < routines; ii++ {
				go func(ii int) {
					defer wg.Done()
					var (
						statedb = newTestStateDB()
						evm     = newTestEVM(statedb)
						API     = api.New(evm, common.Address{})
						counter = lib.NewCounter(API.Persistent().NewReference(runCounterKey))
					)
					r.Equal(uint64(0), counter.Get().Uint64())
					for jj := 0; jj < iterations; jj++ {
						_, err := impl.pc.Run(API, nil)
						r.NoError(err)
						time.Sleep(time.Duration(rand.Intn(10)) * time.Microsecond)
					}
					r.Equal(uint64(iterations), counter.Get().Uint64())
				}(ii)
			}
			wg.Wait()
		})
	}
}

func BenchmarkPrecompile(b *testing.B) {
	var (
		statedb  = newTestStateDB()
		evm      = newTestEVM(statedb)
		API      = api.New(evm, common.Address{})
		preimage = crypto.Keccak256([]byte("test.data"))
	)
	for _, impl := range typicalImplementations {
		b.Run(impl.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := impl.pc.Run(API, preimage)
				require.NoError(b, err)
			}
		})
	}
}

var benchmarkImplementations = []struct {
	name string
	pc   api.Precompile
}{
	{"Native", &precompiles.BenchmarkPrecompile{}},
	{"Wasm", NewWasmPrecompile(benchmarkCode)},
}

func TestRunBenchmarkPrecompile(t *testing.T) {
	var (
		statedb = newTestStateDB()
		evm     = newTestEVM(statedb)
		API     = api.New(evm, common.Address{})
	)
	for _, impl := range benchmarkImplementations {
		t.Run(impl.name, func(t *testing.T) {
			_, err := impl.pc.Run(API, nil)
			require.NoError(t, err)
		})
	}
}
