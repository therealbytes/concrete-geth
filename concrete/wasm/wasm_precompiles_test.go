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
	"context"
	_ "embed"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	cc_api "github.com/ethereum/go-ethereum/concrete/api"
	"github.com/ethereum/go-ethereum/concrete/lib"
	"github.com/ethereum/go-ethereum/concrete/wasm/bridge"
	"github.com/ethereum/go-ethereum/concrete/wasm/bridge/native"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	wz_api "github.com/tetratelabs/wazero/api"
)

//go:embed bin/echo.wasm
var echoCode []byte

//go:embed bin/log.wasm
var logCode []byte

//go:embed bin/typical.wasm
var typicalCode []byte

func TestWasmLog(t *testing.T) {
	address := common.HexToAddress("0x01")
	statedb := NewTestStateDB()
	evm := NewTestEVM(statedb)
	api := cc_api.New(evm, address)
	hostConfig := newHostConfig()

	var lastLog string
	hostConfig.log = func(ctx context.Context, module wz_api.Module, pointer uint64) uint64 {
		mem, _ := native.NewMemory(ctx, module)
		_msg := bridge.GetValues(mem, bridge.MemPointer(pointer))
		lastLog = string(_msg[0])
		return bridge.NullPointer.Uint64()
	}
	hostConfig.address = native.NewAddressHostFunc(address)

	ctx := context.Background()
	mod, r, err := newModule(hostConfig, logCode)
	require.NoError(t, err)

	pc := &wasmPrecompile{}
	pc.r = r
	pc.mod = mod
	pc.memory, pc.allocator = native.NewMemory(ctx, mod)
	pc.expRun = mod.ExportedFunction(WASM_RUN)
	defer pc.close()

	str1 := "hello world"
	str2 := "bye world"
	pc.Run(api, []byte(str1))
	require.Equal(t, str1, lastLog)
	pc.Run(api, []byte(str2))
	require.Equal(t, str2, lastLog)
}

func TestStatelessPrecompile(t *testing.T) {
	address := common.HexToAddress("0x01")
	pc := NewWasmPrecompile(echoCode, address)
	var api cc_api.API

	require.IsType(t, &statelessWasmPrecompile{}, pc)

	testMutatesStorage := func(wg *sync.WaitGroup) {
		defer wg.Done()
		require.False(t, pc.MutatesStorage([]byte("hello world")))
	}
	testFinalise := func(wg *sync.WaitGroup) {
		defer wg.Done()
		require.NoError(t, pc.Finalise(api))
	}
	testCommit := func(wg *sync.WaitGroup) {
		defer wg.Done()
		require.NoError(t, pc.Commit(api))
	}
	testRun := func(data []byte, wg *sync.WaitGroup) {
		defer wg.Done()
		result, err := pc.Run(api, data)
		require.NoError(t, err)
		require.Equal(t, data, result)
	}
	testRequiredGas := func(data []byte, wg *sync.WaitGroup) {
		defer wg.Done()
		gas := pc.RequiredGas(data)
		require.Equal(t, uint64(data[0]), gas)
	}

	var wg sync.WaitGroup
	routines := 10 * 5
	wg.Add(routines)
	for ii := 0; ii < routines/5; ii++ {
		data := []byte{byte(ii)}
		go testMutatesStorage(&wg)
		go testFinalise(&wg)
		go testCommit(&wg)
		go testRun(data, &wg)
		go testRequiredGas(data, &wg)
	}
	wg.Wait()
}

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
			statedb := NewTestStateDB()
			evm := NewTestEVM(statedb)
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
	statedb := NewTestStateDB()
	evm := NewTestEVM(statedb)
	api := cc_api.New(evm, address)
	return api
}

func BenchmarkNativeEchoPrecompile(b *testing.B) {
	address := common.HexToAddress("0x01")
	api := newBenchmarkAPI(address)
	pc := lib.EchoPrecompile{}
	input := []byte("hello world")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pc.Run(api, input)
	}
}

func BenchmarkWasmEchoPrecompile(b *testing.B) {
	address := common.HexToAddress("0x01")
	api := newBenchmarkAPI(address)
	pc := NewWasmPrecompile(echoCode, address)
	input := []byte("hello world")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pc.Run(api, input)
	}
}

func BenchmarkNativeTypicalPrecompile(b *testing.B) {
	address := common.HexToAddress("0x01")
	api := newBenchmarkAPI(address)
	pc := lib.TypicalPrecompile{}
	preimage := []byte("hello world")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pc.Run(api, preimage)
	}
}

func BenchmarkWasmTypicalPrecompile(b *testing.B) {
	address := common.HexToAddress("0x01")
	api := newBenchmarkAPI(address)
	pc := NewWasmPrecompile(typicalCode, address)
	preimage := []byte("hello world")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pc.Run(api, preimage)
	}
}
