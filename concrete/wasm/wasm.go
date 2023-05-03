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
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	cc_api "github.com/ethereum/go-ethereum/concrete/api"
	"github.com/ethereum/go-ethereum/concrete/wasm/bridge"
	"github.com/ethereum/go-ethereum/concrete/wasm/bridge/host"
	"github.com/tetratelabs/wazero"
	wz_api "github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

var (
	// WASM functions
	WASM_IS_PURE         = "concrete_IsPure"
	WASM_MUTATES_STORAGE = "concrete_MutatesStorage"
	WASM_REQUIRED_GAS    = "concrete_RequiredGas"
	WASM_FINALISE        = "concrete_Finalise"
	WASM_COMMIT          = "concrete_Commit"
	WASM_RUN             = "concrete_Run"
	// Host functions
	WASM_EVM_CALLER       = "concrete_EvmCaller"
	WASM_STATEDB_CALLER   = "concrete_StateDBCaller"
	WASM_ADDRESS_CALLER   = "concrete_AddressCaller"
	WASM_LOG_CALLER       = "concrete_LogCaller"
	WASM_KECCAK256_CALLER = "concrete_Keccak256Caller"
)

func NewWasmPrecompile(code []byte, address common.Address) cc_api.Precompile {
	pc := newWasmPrecompile(code, address)
	if pc.isPure() {
		return &statelessWasmPrecompile{pc}
	}
	return pc
}

type hostConfig struct {
	evm       host.HostFunc
	statedb   host.HostFunc
	address   host.HostFunc
	log       host.HostFunc
	keccak256 host.HostFunc
}

func newHostConfig() *hostConfig {
	return &hostConfig{
		evm:       host.DisabledHostFunc,
		statedb:   host.DisabledHostFunc,
		address:   host.DisabledHostFunc,
		log:       host.LogHostFunc,
		keccak256: host.Keccak256HostFunc,
	}
}

func newModule(config *hostConfig, code []byte) (wz_api.Module, wazero.Runtime, error) {
	ctx := context.Background()
	r := wazero.NewRuntime(ctx)
	_, err := r.NewHostModuleBuilder("env").
		NewFunctionBuilder().WithFunc(config.evm).Export(WASM_EVM_CALLER).
		NewFunctionBuilder().WithFunc(config.statedb).Export(WASM_STATEDB_CALLER).
		NewFunctionBuilder().WithFunc(config.address).Export(WASM_ADDRESS_CALLER).
		NewFunctionBuilder().WithFunc(config.log).Export(WASM_LOG_CALLER).
		NewFunctionBuilder().WithFunc(config.keccak256).Export(WASM_KECCAK256_CALLER).
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

type wasmPrecompile struct {
	r                 wazero.Runtime
	mod               wz_api.Module
	mutex             sync.Mutex
	memory            bridge.Memory
	allocator         bridge.Allocator
	api               cc_api.API
	expIsPure         wz_api.Function
	expMutatesStorage wz_api.Function
	expRequiredGas    wz_api.Function
	expFinalise       wz_api.Function
	expCommit         wz_api.Function
	expRun            wz_api.Function
}

func newWasmPrecompile(code []byte, address common.Address) *wasmPrecompile {
	pc := &wasmPrecompile{}

	hostConfig := newHostConfig()
	apiGetter := func() cc_api.API { return pc.api }
	hostConfig.evm = host.NewEVMHostFunc(apiGetter)
	hostConfig.statedb = host.NewStateDBHostFunc(apiGetter)
	hostConfig.address = host.NewAddressHostFunc(address)

	mod, r, err := newModule(hostConfig, code)
	if err != nil {
		panic(err)
	}

	pc.r = r
	pc.mod = mod
	pc.memory, pc.allocator = host.NewMemory(context.Background(), mod)

	pc.expIsPure = mod.ExportedFunction(WASM_IS_PURE)
	pc.expMutatesStorage = mod.ExportedFunction(WASM_MUTATES_STORAGE)
	pc.expRequiredGas = mod.ExportedFunction(WASM_REQUIRED_GAS)
	pc.expFinalise = mod.ExportedFunction(WASM_FINALISE)
	pc.expCommit = mod.ExportedFunction(WASM_COMMIT)
	pc.expRun = mod.ExportedFunction(WASM_RUN)

	return pc
}

func (p *wasmPrecompile) close() {
	ctx := context.Background()
	p.r.Close(ctx)
}

func (p *wasmPrecompile) call__Uint64(expFunc wz_api.Function) uint64 {
	ctx := context.Background()
	_ret, err := expFunc.Call(ctx)
	if err != nil {
		panic(err)
	}
	return _ret[0]
}

func (p *wasmPrecompile) call__Err(expFunc wz_api.Function) error {
	_retPointer := p.call__Uint64(expFunc)
	retPointer := bridge.MemPointer(_retPointer)
	_, retErr := bridge.GetReturnWithError(p.memory, retPointer)
	return retErr
}

func (p *wasmPrecompile) call_Bytes_Uint64(expFunc wz_api.Function, input []byte) uint64 {
	ctx := context.Background()
	pointer := bridge.PutValue(p.memory, input)
	defer p.allocator.Free(pointer)
	_ret, err := expFunc.Call(ctx, pointer.Uint64())
	if err != nil {
		fmt.Println("err", err)
		panic(err)
	}
	return _ret[0]
}

func (p *wasmPrecompile) call_Bytes_BytesErr(expFunc wz_api.Function, input []byte) ([]byte, error) {
	_retPointer := p.call_Bytes_Uint64(expFunc, input)
	retPointer := bridge.MemPointer(_retPointer)
	retValues, retErr := bridge.GetReturnWithError(p.memory, retPointer)
	return retValues[0], retErr
}

func (p *wasmPrecompile) before(api cc_api.API) {
	p.mutex.Lock()
	p.api = api
}

func (p *wasmPrecompile) after(api cc_api.API) {
	p.api = nil
	p.allocator.Prune()
	p.mutex.Unlock()
}

func (p *wasmPrecompile) isPure() bool {
	p.before(nil)
	defer p.after(nil)
	return p.call__Uint64(p.expIsPure) != 0
}

func (p *wasmPrecompile) RequiredGas(input []byte) uint64 {
	p.before(nil)
	defer p.after(nil)
	return p.call_Bytes_Uint64(p.expRequiredGas, input)
}

func (p *wasmPrecompile) MutatesStorage(input []byte) bool {
	p.before(nil)
	defer p.after(nil)
	return p.call_Bytes_Uint64(p.expMutatesStorage, input) != 0
}

func (p *wasmPrecompile) Finalise(api cc_api.API) error {
	p.before(api)
	defer p.after(api)
	return p.call__Err(p.expFinalise)
}

func (p *wasmPrecompile) Commit(api cc_api.API) error {
	p.before(api)
	defer p.after(api)
	return p.call__Err(p.expCommit)
}

func (p *wasmPrecompile) Run(api cc_api.API, input []byte) ([]byte, error) {
	p.before(api)
	defer p.after(api)
	return p.call_Bytes_BytesErr(p.expRun, input)
}

var _ cc_api.Precompile = (*wasmPrecompile)(nil)

type statelessWasmPrecompile struct {
	*wasmPrecompile
}

func (p *statelessWasmPrecompile) MutatesStorage(input []byte) bool {
	return false
}

func (p *statelessWasmPrecompile) Finalise(api cc_api.API) error {
	return nil
}

func (p *statelessWasmPrecompile) Commit(api cc_api.API) error {
	return nil
}
