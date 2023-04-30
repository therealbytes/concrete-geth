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
	"sync"

	"github.com/ethereum/go-ethereum/common"
	cc_api "github.com/ethereum/go-ethereum/concrete/api"
	"github.com/ethereum/go-ethereum/concrete/wasm/bridge"
	"github.com/ethereum/go-ethereum/concrete/wasm/bridge/native"
	"github.com/tetratelabs/wazero"
	wz_api "github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

var (
	WASM_IS_PURE         = "concrete_IsPure"
	WASM_MUTATES_STORAGE = "concrete_MutatesStorage"
	WASM_REQUIRED_GAS    = "concrete_RequiredGas"
	WASM_FINALISE        = "concrete_Finalise"
	WASM_COMMIT          = "concrete_Commit"
	WASM_RUN             = "concrete_Run"
	WASM_EVM_BRIDGE      = "concrete_EvmBridge"
	WASM_STATEDB_BRIDGE  = "concrete_StateDBBridge"
	WASM_ADDRESS_BRIDGE  = "concrete_AddressBridge"
	WASM_LOG_BRIDGE      = "concrete_LogBridge"
)

func NewWasmPrecompile(code []byte, address common.Address) cc_api.Precompile {
	pc := newStatelessWasmPrecompile(code, address)
	if pc.isPure() {
		return pc
	}
	pc.close()
	return newStatefulWasmPrecompile(code)
}

type bridgeConfig struct {
	evmBridge     native.NativeBridgeFunc
	stateDBBridge native.NativeBridgeFunc
	addressBridge native.NativeBridgeFunc
	logBridge     native.NativeBridgeFunc
}

func newModule(bridges *bridgeConfig, code []byte) (wz_api.Module, wazero.Runtime, error) {

	ctx := context.Background()

	evmBridge := bridges.evmBridge
	if evmBridge == nil {
		evmBridge = native.DisabledBridge
	}

	stateDBBridge := bridges.stateDBBridge
	if stateDBBridge == nil {
		stateDBBridge = native.DisabledBridge
	}

	addressBridge := bridges.addressBridge
	if addressBridge == nil {
		addressBridge = native.BridgeAddress0
	}

	logBridge := bridges.logBridge
	if logBridge == nil {
		logBridge = native.BridgeLog
	}

	r := wazero.NewRuntime(ctx)
	_, err := r.NewHostModuleBuilder("env").
		NewFunctionBuilder().WithFunc(evmBridge).Export(WASM_EVM_BRIDGE).
		NewFunctionBuilder().WithFunc(stateDBBridge).Export(WASM_STATEDB_BRIDGE).
		NewFunctionBuilder().WithFunc(addressBridge).Export(WASM_ADDRESS_BRIDGE).
		NewFunctionBuilder().WithFunc(logBridge).Export(WASM_LOG_BRIDGE).
		Instantiate(ctx)
	if err != nil {
		return nil, nil, err
	}
	wasi_snapshot_preview1.MustInstantiate(ctx, r)
	mod, err := r.Instantiate(ctx, code)
	if err != nil {
		return nil, nil, err
	}

	return mod, r, nil
}

type mutexQueue struct {
	mutex sync.Mutex
	queue chan struct{}
}

func newMutexQueue(capacity int) *mutexQueue {
	return &mutexQueue{
		queue: make(chan struct{}, capacity),
	}
}

func (m *mutexQueue) Lock() {
	m.queue <- struct{}{}
	m.mutex.Lock()
}

func (m *mutexQueue) Unlock() {
	m.mutex.Unlock()
	<-m.queue
}

type wasmPrecompile struct {
	r     wazero.Runtime
	mod   wz_api.Module
	mutex *mutexQueue
}

func (p *wasmPrecompile) close() {
	ctx := context.Background()
	p.r.Close(ctx)
}

func (p *wasmPrecompile) call__Uint64(funcName *string) uint64 {
	ctx := context.Background()
	_ret, err := p.mod.ExportedFunction(*funcName).Call(ctx)
	if err != nil {
		panic(err)
	}
	return _ret[0]
}

func (p *wasmPrecompile) call_Bytes_Uint64(funcName *string, input []byte) uint64 {
	ctx := context.Background()
	pointer, err := native.WriteMemory(ctx, p.mod, input)
	if err != nil {
		panic(err)
	}
	defer native.FreeMemory(ctx, p.mod, pointer)
	_ret, err := p.mod.ExportedFunction(*funcName).Call(ctx, pointer.Uint64())
	if err != nil {
		panic(err)
	}
	return _ret[0]
}

func (p *wasmPrecompile) call_Bytes_BytesErr(funcName *string, input []byte) ([]byte, error) {
	ctx := context.Background()
	_retPointer := p.call_Bytes_Uint64(funcName, input)
	retPointer := bridge.MemPointer(_retPointer)
	retValues, retErr := native.GetReturnWithError(ctx, p.mod, retPointer)
	return retValues[0], retErr
}

func (p *wasmPrecompile) call__Err(funcName *string) error {
	ctx := context.Background()
	_retPointer, err := p.mod.ExportedFunction(*funcName).Call(ctx)
	if err != nil {
		panic(err)
	}
	retPointer := bridge.MemPointer(_retPointer[0])
	_, retErr := native.GetReturnWithError(ctx, p.mod, retPointer)
	return retErr
}

func (p *wasmPrecompile) before(api cc_api.API) {
	p.mutex.Lock()
}

func (p *wasmPrecompile) after(api cc_api.API) {
	p.mutex.Unlock()
}

type statelessWasmPrecompile struct {
	wasmPrecompile
}

func newStatelessWasmPrecompile(code []byte, address common.Address) *statelessWasmPrecompile {
	addressBridge := func(ctx context.Context, mod wz_api.Module, pointer uint64) uint64 {
		return native.BridgeAddress(ctx, mod, pointer, address)
	}
	mod, r, err := newModule(&bridgeConfig{addressBridge: addressBridge}, code)
	if err != nil {
		panic(err)
	}
	mutex := newMutexQueue(16)
	return &statelessWasmPrecompile{wasmPrecompile{r: r, mod: mod, mutex: mutex}}
}

func (p *wasmPrecompile) isPure() bool {
	p.before(nil)
	defer p.after(nil)
	return p.call__Uint64(&WASM_IS_PURE) != 0
}

func (p *wasmPrecompile) MutatesStorage(input []byte) bool {
	return false
}

func (p *wasmPrecompile) RequiredGas(input []byte) uint64 {
	p.before(nil)
	defer p.after(nil)
	return p.call_Bytes_Uint64(&WASM_REQUIRED_GAS, input)
}

func (p *wasmPrecompile) Finalise(api cc_api.API) error {
	return nil
}

func (p *wasmPrecompile) Commit(api cc_api.API) error {
	return nil
}

func (p *wasmPrecompile) Run(api cc_api.API, input []byte) ([]byte, error) {
	p.before(api)
	defer p.after(api)
	return p.call_Bytes_BytesErr(&WASM_RUN, input)
}

var _ cc_api.Precompile = (*statelessWasmPrecompile)(nil)

type statefulWasmPrecompile struct {
	wasmPrecompile
	api cc_api.API
}

func newStatefulWasmPrecompile(code []byte) *statefulWasmPrecompile {
	pc := &statefulWasmPrecompile{}

	evmBridge := func(ctx context.Context, mod wz_api.Module, pointer uint64) uint64 {
		return native.BridgeCallEVM(ctx, mod, pointer, pc.api.EVM())
	}
	stateDBBridge := func(ctx context.Context, mod wz_api.Module, pointer uint64) uint64 {
		return native.BridgeCallStateDB(ctx, mod, pointer, pc.api.StateDB())
	}
	addressBridge := func(ctx context.Context, mod wz_api.Module, pointer uint64) uint64 {
		return native.BridgeAddress(ctx, mod, pointer, pc.api.Address())
	}

	bridges := &bridgeConfig{
		evmBridge:     evmBridge,
		stateDBBridge: stateDBBridge,
		addressBridge: addressBridge,
	}

	mod, r, err := newModule(bridges, code)
	if err != nil {
		panic(err)
	}

	pc.r = r
	pc.mod = mod
	pc.mutex = newMutexQueue(16)

	return pc
}

func (p *statefulWasmPrecompile) before(api cc_api.API) {
	p.mutex.Lock()
	p.api = api
}

func (p *statefulWasmPrecompile) after(api cc_api.API) {
	p.mutex.Unlock()
	p.api = nil
}

func (p *statefulWasmPrecompile) MutatesStorage(input []byte) bool {
	p.before(nil)
	defer p.after(nil)
	return p.call_Bytes_Uint64(&WASM_MUTATES_STORAGE, input) != 0
}

func (p *statefulWasmPrecompile) Finalise(api cc_api.API) error {
	p.before(api)
	defer p.after(api)
	return p.call__Err(&WASM_FINALISE)
}

func (p *statefulWasmPrecompile) Commit(api cc_api.API) error {
	p.before(api)
	defer p.after(api)
	return p.call__Err(&WASM_COMMIT)
}

func (p *statefulWasmPrecompile) Run(api cc_api.API, input []byte) ([]byte, error) {
	p.before(api)
	defer p.after(api)
	return p.call_Bytes_BytesErr(&WASM_RUN, input)
}

var _ cc_api.Precompile = (*statefulWasmPrecompile)(nil)
