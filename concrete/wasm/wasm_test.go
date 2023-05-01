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
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	cc_api "github.com/ethereum/go-ethereum/concrete/api"
	"github.com/ethereum/go-ethereum/concrete/wasm/bridge"
	"github.com/ethereum/go-ethereum/concrete/wasm/bridge/native"
	"github.com/stretchr/testify/require"
	wz_api "github.com/tetratelabs/wazero/api"
)

//go:embed bin/echo.wasm
var echoCode []byte

//go:embed bin/log.wasm
var logCode []byte

func TestStatelessWasmBridges(t *testing.T) {
	address := common.HexToAddress("0x01")

	var lastLog string
	bridgeLog := func(ctx context.Context, module wz_api.Module, pointer uint64) uint64 {
		msg := native.GetValue(ctx, module, bridge.MemPointer(pointer))
		lastLog = string(msg)
		return bridge.NullPointer.Uint64()
	}
	bridgeAddress := func(ctx context.Context, module wz_api.Module, pointer uint64) uint64 {
		return native.PutValue(ctx, module, address.Bytes()).Uint64()
	}

	mod, r, _ := newModule(&bridgeConfig{addressBridge: bridgeAddress, logBridge: bridgeLog}, logCode)
	pc := &statelessWasmPrecompile{wasmPrecompile{r, mod, newMutexQueue(1)}}
	var api cc_api.API

	defer pc.close()

	// Test Log
	str1 := "hello world"
	str2 := "bye world"
	pc.Run(api, []byte(str1))
	require.Equal(t, str1, lastLog)
	pc.Run(api, []byte(str2))
	require.Equal(t, str2, lastLog)

	// Test Address
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
	routines := 6 * 5
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
