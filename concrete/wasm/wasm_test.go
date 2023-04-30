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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	cc_api "github.com/ethereum/go-ethereum/concrete/api"
	cc_api_test "github.com/ethereum/go-ethereum/concrete/api/test"
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

	ctx := context.Background()
	mod, _, _ := newModule(ctx, &bridgeConfig{addressBridge: bridgeAddress, logBridge: bridgeLog}, logCode)
	db := cc_api_test.NewMockStateDB()
	evm := cc_api_test.NewMockEVM(db)
	api := cc_api.New(evm, address)
	pc := NewStatelessWasmPrecompile(mod)

	// Test Log
	pc.Run(api, []byte("hello world"))
	require.Equal(t, "hello world", lastLog)
	pc.Run(api, []byte("bye world"))
	require.Equal(t, "bye world", lastLog)

	// Test Address
}

func TestNativeBridges(t *testing.T) {
	address := common.HexToAddress("0x01")
	ctx := context.Background()
	mod, _, _ := newModule(ctx, &bridgeConfig{}, echoCode)
	db := cc_api_test.NewMockStateDB()
	evm := cc_api_test.NewMockEVM(db)
	api := cc_api.New(evm, address)
	pc := NewStatelessWasmPrecompile(mod)

	// Test Run
	data := []byte("hello world")
	result, err := pc.Run(api, data)
	require.NoError(t, err)
	require.Equal(t, data, result)

	// Test RequiredGas
	data = []byte{2}
	gas := pc.RequiredGas(data)
	require.Equal(t, uint64(2), gas)

	// Test Finalise
	// Test Commit
	// Test IsPure
}
