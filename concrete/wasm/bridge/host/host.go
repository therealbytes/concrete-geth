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

package host

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	cc_api "github.com/ethereum/go-ethereum/concrete/api"
	"github.com/ethereum/go-ethereum/concrete/wasm/bridge"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	wz_api "github.com/tetratelabs/wazero/api"
)

var (
	ErrMemoryReadOutOfRange = errors.New("go: memory read out of range of memory size")
)

var (
	WASM_MALLOC = "concrete_Malloc"
	WASM_FREE   = "concrete_Free"
	WASM_PRUNE  = "concrete_Prune"
)

type allocator struct {
	ctx       context.Context
	mod       wz_api.Module
	expMalloc wz_api.Function
	expFree   wz_api.Function
	expPrune  wz_api.Function
}

func NewAllocator(ctx context.Context, mod wz_api.Module) bridge.Allocator {
	return &allocator{ctx: ctx, mod: mod}
}

func (a *allocator) Malloc(size uint32) bridge.MemPointer {
	if size == 0 {
		return bridge.NullPointer
	}
	if a.expMalloc == nil {
		a.expMalloc = a.mod.ExportedFunction(WASM_MALLOC)
	}
	_offset, err := a.expMalloc.Call(a.ctx, uint64(size))
	if err != nil {
		panic(err)
	}
	var pointer bridge.MemPointer
	pointer.Pack(uint32(_offset[0]), size)
	return pointer
}

func (a *allocator) Free(pointer bridge.MemPointer) {
	if pointer.IsNull() {
		return
	}
	if a.expFree == nil {
		a.expFree = a.mod.ExportedFunction(WASM_FREE)
	}
	_, err := a.expFree.Call(a.ctx, uint64(pointer.Offset()))
	if err != nil {
		panic(err)
	}
}

func (a *allocator) Prune() {
	if a.expPrune == nil {
		a.expPrune = a.mod.ExportedFunction(WASM_PRUNE)
	}
	_, err := a.expPrune.Call(a.ctx)
	if err != nil {
		panic(err)
	}
}

type memory struct {
	allocator
}

func NewMemory(ctx context.Context, mod wz_api.Module) (bridge.Memory, bridge.Allocator) {
	alloc := &allocator{ctx: ctx, mod: mod}
	return &memory{allocator: *alloc}, alloc
}

func NewMemoryFromAlloc(alloc *allocator) bridge.Allocator {
	return &memory{*alloc}
}

func (m *memory) Write(data []byte) bridge.MemPointer {
	if len(data) == 0 {
		return bridge.NullPointer
	}
	pointer := m.Malloc(uint32(len(data)))
	ok := m.mod.Memory().Write(pointer.Offset(), data)
	if !ok {
		panic(ErrMemoryReadOutOfRange)
	}
	return pointer
}

func (m *memory) Read(pointer bridge.MemPointer) []byte {
	if pointer.IsNull() {
		return []byte{}
	}
	output, ok := m.mod.Memory().Read(pointer.Offset(), pointer.Size())
	if !ok {
		panic(ErrMemoryReadOutOfRange)
	}
	return output
}

type HostFunc func(ctx context.Context, module wz_api.Module, pointer uint64) uint64

func NewStateDBHostFunc(apiGetter func() cc_api.API) HostFunc {
	return func(ctx context.Context, module wz_api.Module, _pointer uint64) uint64 {
		statedb := apiGetter().StateDB()
		mem, _ := NewMemory(ctx, module)

		var handleCall func(pointer bridge.MemPointer) bridge.MemPointer
		handleCall = func(pointer bridge.MemPointer) bridge.MemPointer {
			args := bridge.GetArgs(mem, pointer)
			var opcode bridge.OpCode
			opcode.Decode(args[0])
			args = args[1:]

			if opcode == bridge.Op_StateDB_Many {
				for _, ptr := range bridge.UnpackPointers(args[0]) {
					handleCall(ptr)
				}
				return bridge.NullPointer
			} else {
				out := CallStateDB(statedb, opcode, args)
				ptr := bridge.PutValue(mem, out)
				return ptr
			}
		}

		return handleCall(bridge.MemPointer(_pointer)).Uint64()
	}
}

func NewEVMHostFunc(apiGetter func() cc_api.API) HostFunc {
	return func(ctx context.Context, module wz_api.Module, pointer uint64) uint64 {
		mem, _ := NewMemory(ctx, module)
		args := bridge.GetArgs(mem, bridge.MemPointer(pointer))
		var opcode bridge.OpCode
		opcode.Decode(args[0])
		args = args[1:]
		out := CallEVM(apiGetter().EVM(), opcode, args)
		return bridge.PutValue(mem, out).Uint64()
	}
}

func NewAddressHostFunc(address common.Address) HostFunc {
	return func(ctx context.Context, module wz_api.Module, pointer uint64) uint64 {
		mem, _ := NewMemory(ctx, module)
		return bridge.PutValue(mem, address.Bytes()).Uint64()
	}
}

func DisabledHostFunc(ctx context.Context, module wz_api.Module, pointer uint64) uint64 {
	panic("go: disabled host function -- this likely means you are trying to access the concrete API from a wasm precompile declared as pure")
}

func LogHostFunc(ctx context.Context, module wz_api.Module, pointer uint64) uint64 {
	mem, _ := NewMemory(ctx, module)
	data := bridge.GetArgs(mem, bridge.MemPointer(pointer))
	opcode := data[0][0]
	msg := data[1]
	if opcode == bridge.Op_Log_Log {
		log.Debug("wasm:", string(msg))
	} else {
		fmt.Println("wasm:", string(msg))
	}
	return bridge.NullPointer.Uint64()
}

func Keccak256HostFunc(ctx context.Context, module wz_api.Module, pointer uint64) uint64 {
	mem, _ := NewMemory(ctx, module)
	data := bridge.GetArgs(mem, bridge.MemPointer(pointer))
	hash := crypto.Keccak256(data...)
	return bridge.PutValue(mem, hash).Uint64()
}

func TimeHostFunc(ctx context.Context, module wz_api.Module, pointer uint64) uint64 {
	return uint64(time.Now().UnixNano())
}

func CallStateDB(db cc_api.StateDB, opcode bridge.OpCode, args [][]byte) []byte {
	switch opcode {
	case bridge.Op_StateDB_GetPersistentState:
		addr := common.BytesToAddress(args[0])
		hash := common.BytesToHash(args[1])
		state := db.GetPersistentState(addr, hash)
		return state.Bytes()

	case bridge.Op_StateDB_GetPersistentPreimage:
		hash := common.BytesToHash(args[0])
		preimage := db.GetPersistentPreimage(hash)
		return preimage

	case bridge.Op_StateDB_GetPersistentPreimageSize:
		hash := common.BytesToHash(args[0])
		size := db.GetPersistentPreimageSize(hash)
		return bridge.Uint64ToBytes(uint64(size))

	case bridge.Op_StateDB_GetEphemeralState:
		addr := common.BytesToAddress(args[0])
		hash := common.BytesToHash(args[1])
		state := db.GetEphemeralState(addr, hash)
		return state.Bytes()

	case bridge.Op_StateDB_GetEphemeralPreimage:
		hash := common.BytesToHash(args[0])
		preimage := db.GetEphemeralPreimage(hash)
		return preimage

	case bridge.Op_StateDB_GetEphemeralPreimageSize:
		hash := common.BytesToHash(args[0])
		size := db.GetEphemeralPreimageSize(hash)
		return bridge.Uint64ToBytes(uint64(size))

	case bridge.Op_StateDB_SetPersistentState:
		addr := common.BytesToAddress(args[0])
		key := common.BytesToHash(args[1])
		value := common.BytesToHash(args[2])
		db.SetPersistentState(addr, key, value)

	case bridge.Op_StateDB_AddPersistentPreimage:
		hash := common.BytesToHash(args[0])
		preimage := args[1]
		db.AddPersistentPreimage(hash, preimage)

	case bridge.Op_StateDB_SetEphemeralState:
		addr := common.BytesToAddress(args[0])
		key := common.BytesToHash(args[1])
		value := common.BytesToHash(args[2])
		db.SetEphemeralState(addr, key, value)

	case bridge.Op_StateDB_AddEphemeralPreimage:
		hash := common.BytesToHash(args[0])
		preimage := args[1]
		db.AddEphemeralPreimage(hash, preimage)
	}

	return nil
}

func CallEVM(evm cc_api.EVM, opcode bridge.OpCode, args [][]byte) []byte {
	switch opcode {
	case bridge.Op_EVM_BlockHash:
		block := new(big.Int).SetBytes(args[0])
		hash := evm.BlockHash(block)
		return hash.Bytes()

	case bridge.Op_EVM_BlockTimestamp:
		timestamp := evm.BlockTimestamp()
		return timestamp.Bytes()

	case bridge.Op_EVM_BlockNumber:
		number := evm.BlockNumber()
		return number.Bytes()

	case bridge.Op_EVM_BlockDifficulty:
		difficulty := evm.BlockDifficulty()
		return difficulty.Bytes()

	case bridge.Op_EVM_BlockGasLimit:
		gasLimit := evm.BlockGasLimit()
		return gasLimit.Bytes()

	case bridge.Op_EVM_BlockCoinbase:
		coinbase := evm.BlockCoinbase()
		return coinbase.Bytes()

	case bridge.Op_EVM_Block:
		block := bridge.BlockData{
			Timestamp:  evm.BlockTimestamp(),
			Number:     evm.BlockNumber(),
			Difficulty: evm.BlockDifficulty(),
			GasLimit:   evm.BlockGasLimit(),
			Coinbase:   evm.BlockCoinbase(),
		}
		return block.Encode()
	}

	return nil
}
