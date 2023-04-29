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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/concrete/api"
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

func NewWasmPrecompile(code []byte) api.Precompile {
	ctx := context.Background()

	// Create a stateless module for pure functions
	mod, _, err := newModule(ctx, native.DisabledBridge, native.DisabledBridge, native.DisabledBridge, code)
	if err != nil {
		panic(err)
	}

	_isPure, err := mod.ExportedFunction(WASM_IS_PURE).Call(ctx)
	if err != nil {
		panic(err)
	}
	isPure := _isPure[0] == 1

	if isPure {
		return NewStatelessWasmPrecompile(mod)
	} else {
		return NewStatefulWasmPrecompile(mod, code)
	}
}

func newModule(ctx context.Context, evmBridge native.NativeBridgeFunc, stateDBBridge native.NativeBridgeFunc, addressBridge native.NativeBridgeFunc, code []byte) (wz_api.Module, wazero.Runtime, error) {
	r := wazero.NewRuntime(ctx)
	_, err := r.NewHostModuleBuilder("env").
		NewFunctionBuilder().WithFunc(evmBridge).Export(WASM_EVM_BRIDGE).
		NewFunctionBuilder().WithFunc(stateDBBridge).Export(WASM_STATEDB_BRIDGE).
		NewFunctionBuilder().WithFunc(addressBridge).Export(WASM_ADDRESS_BRIDGE).
		NewFunctionBuilder().WithFunc(native.BridgeLog).Export(WASM_LOG_BRIDGE).
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

type StatelessWasmPrecompile struct {
	mod wz_api.Module
}

func NewStatelessWasmPrecompile(mod wz_api.Module) *StatelessWasmPrecompile {
	return &StatelessWasmPrecompile{mod}
}

func (p *StatelessWasmPrecompile) statelessCall_Bytes_Uint64(funcName *string, input []byte) uint64 {
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

func (p *StatelessWasmPrecompile) statelessCall_Bytes_BytesErr(funcName *string, input []byte) ([]byte, error) {
	ctx := context.Background()
	_retPointer := p.statelessCall_Bytes_Uint64(funcName, input)
	retPointer := bridge.MemPointer(_retPointer)
	retValues, retErr := native.GetReturnWithError(ctx, p.mod, retPointer)
	return retValues[0], retErr
}

func (p *StatelessWasmPrecompile) MutatesStorage(input []byte) bool {
	return false
}

func (p *StatelessWasmPrecompile) RequiredGas(input []byte) uint64 {
	return p.statelessCall_Bytes_Uint64(&WASM_REQUIRED_GAS, input)
}

func (p *StatelessWasmPrecompile) Finalise(api api.API) error {
	return nil
}

func (p *StatelessWasmPrecompile) Commit(api api.API) error {
	return nil
}

func (p *StatelessWasmPrecompile) Run(api api.API, input []byte) ([]byte, error) {
	return p.statelessCall_Bytes_BytesErr(&WASM_RUN, input)
}

type workerPayload struct {
	funcName *string
	api      api.API
}

type workerPayload__Err struct {
	workerPayload
	out chan error
}

type workerResponse_BytesErr struct {
	data []byte
	err  error
}

type workerPayload_Bytes_BytesErr struct {
	workerPayload
	input []byte
	out   chan *workerResponse_BytesErr
}

type StatefulWasmPrecompile struct {
	StatelessWasmPrecompile
	workerIn__Err           chan *workerPayload__Err
	workerIn_Bytes_BytesErr chan *workerPayload_Bytes_BytesErr
}

func statefulPrecompileWorker(ctx context.Context, code []byte, workerIn__Err chan *workerPayload__Err, workerIn_Bytes_BytesErr chan *workerPayload_Bytes_BytesErr, ready chan struct{}) {
	var evm api.EVM
	var statedb api.StateDB
	var address common.Address

	evmBridge := func(ctx context.Context, mod wz_api.Module, pointer uint64) uint64 {
		return native.BridgeCallEVM(ctx, mod, pointer, evm)
	}
	stateDBBridge := func(ctx context.Context, mod wz_api.Module, pointer uint64) uint64 {
		return native.BridgeCallStateDB(ctx, mod, pointer, statedb)
	}
	addressBridge := func(ctx context.Context, mod wz_api.Module, pointer uint64) uint64 {
		return native.BridgeAddress(ctx, mod, pointer, address)
	}

	mod, r, err := newModule(ctx, evmBridge, stateDBBridge, addressBridge, code)
	if err != nil {
		panic(err)
	}
	defer r.Close(ctx)

	ready <- struct{}{}

	for {
		select {
		case <-ctx.Done():
			return

		case payload := <-workerIn__Err:
			evm = payload.api.EVM()
			statedb = payload.api.StateDB()
			_retPointer, err := mod.ExportedFunction(*payload.funcName).Call(ctx)
			if err != nil {
				panic(err)
			}
			retPointer := bridge.MemPointer(_retPointer[0])
			_, retErr := native.GetReturnWithError(ctx, mod, retPointer)
			payload.out <- retErr

		case payload := <-workerIn_Bytes_BytesErr:
			evm = payload.api.EVM()
			statedb = payload.api.StateDB()
			pointer, err := native.WriteMemory(ctx, mod, payload.input)
			if err != nil {
				panic(err)
			}
			_retPointer, err := mod.ExportedFunction(*payload.funcName).Call(ctx, pointer.Uint64())
			if err != nil {
				panic(err)
			}
			retPointer := bridge.MemPointer(_retPointer[0])
			retValues, retErr := native.GetReturnWithError(ctx, mod, retPointer)
			payload.out <- &workerResponse_BytesErr{retValues[0], retErr}
		}

		native.PruneMemory(ctx, mod)
	}
}

func NewStatefulWasmPrecompile(mod wz_api.Module, code []byte) *StatefulWasmPrecompile {
	ctx := context.Background()
	workerIn__Err := make(chan *workerPayload__Err, 8)
	workerIn_Bytes_BytesErr := make(chan *workerPayload_Bytes_BytesErr, 8)
	ready := make(chan struct{})
	go statefulPrecompileWorker(ctx, code, workerIn__Err, workerIn_Bytes_BytesErr, ready)
	<-ready
	return &StatefulWasmPrecompile{
		StatelessWasmPrecompile: StatelessWasmPrecompile{mod},
		workerIn__Err:           workerIn__Err,
		workerIn_Bytes_BytesErr: workerIn_Bytes_BytesErr,
	}
}

func (p *StatefulWasmPrecompile) statefulCall__Err(api api.API, funcName *string) error {
	out := make(chan error)
	p.workerIn__Err <- &workerPayload__Err{workerPayload{funcName, api}, out}
	return <-out
}

func (p *StatefulWasmPrecompile) statefulCall_Bytes_BytesErr(api api.API, funcName *string, input []byte) ([]byte, error) {
	out := make(chan *workerResponse_BytesErr)
	p.workerIn_Bytes_BytesErr <- &workerPayload_Bytes_BytesErr{workerPayload{funcName, api}, input, out}
	resp := <-out
	return resp.data, resp.err
}

func (p *StatefulWasmPrecompile) MutatesStorage(input []byte) bool {
	_mutates := p.statelessCall_Bytes_Uint64(&WASM_MUTATES_STORAGE, input)
	return _mutates != 0
}

func (p *StatefulWasmPrecompile) Finalise(api api.API) error {
	return p.statefulCall__Err(api, &WASM_FINALISE)
}

func (p *StatefulWasmPrecompile) Commit(api api.API) error {
	return p.statefulCall__Err(api, &WASM_COMMIT)
}

func (p *StatefulWasmPrecompile) Run(api api.API, input []byte) ([]byte, error) {
	return p.statefulCall_Bytes_BytesErr(api, &WASM_RUN, input)
}
