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

package native

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/concrete/api"
	"github.com/ethereum/go-ethereum/concrete/wasm/bridge"
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

type NativeBridgeFunc func(ctx context.Context, module wz_api.Module, pointer uint64) uint64

func AllocMemory(ctx context.Context, module wz_api.Module, size uint32) (bridge.MemPointer, error) {
	if size == 0 {
		return bridge.NullPointer, nil
	}
	_offset, err := module.ExportedFunction(WASM_MALLOC).Call(ctx, uint64(size))
	if err != nil {
		return bridge.NullPointer, err
	}
	var pointer bridge.MemPointer
	pointer.Pack(uint32(_offset[0]), size)
	return pointer, nil
}

func FreeMemory(ctx context.Context, module wz_api.Module, pointer bridge.MemPointer) error {
	if pointer.IsNull() {
		return nil
	}
	_, err := module.ExportedFunction(WASM_FREE).Call(ctx, uint64(pointer.Offset()))
	return err
}

func PruneMemory(ctx context.Context, module wz_api.Module) error {
	_, err := module.ExportedFunction(WASM_PRUNE).Call(ctx)
	return err
}

func WriteMemory(ctx context.Context, module wz_api.Module, data []byte) (bridge.MemPointer, error) {
	if len(data) == 0 {
		return bridge.NullPointer, nil
	}
	pointer, err := AllocMemory(ctx, module, uint32(len(data)))
	if err != nil {
		return bridge.NullPointer, err
	}
	ok := module.Memory().Write(pointer.Offset(), data)
	if !ok {
		return bridge.NullPointer, ErrMemoryReadOutOfRange
	}
	return pointer, nil
}

func ReadMemory(ctx context.Context, module wz_api.Module, pointer bridge.MemPointer) ([]byte, error) {
	if pointer.IsNull() {
		return []byte{}, nil
	}
	output, ok := module.Memory().Read(pointer.Offset(), pointer.Size())
	if !ok {
		return nil, ErrMemoryReadOutOfRange
	}
	return output, nil
}

func PutValue(ctx context.Context, module wz_api.Module, value []byte) bridge.MemPointer {
	pointer, err := WriteMemory(ctx, module, value)
	if err != nil {
		panic(err)
	}
	return pointer
}

func GetValue(ctx context.Context, module wz_api.Module, pointer bridge.MemPointer) []byte {
	value, err := ReadMemory(ctx, module, pointer)
	if err != nil {
		panic(err)
	}
	return value
}

func PutValues(ctx context.Context, module wz_api.Module, values [][]byte) bridge.MemPointer {
	if len(values) == 0 {
		return bridge.NullPointer
	}
	var pointers []bridge.MemPointer
	for _, v := range values {
		pointer := PutValue(ctx, module, v)
		pointers = append(pointers, pointer)
	}
	pointer := PutValue(ctx, module, bridge.PackPointers(pointers))
	return pointer
}

func GetValues(ctx context.Context, module wz_api.Module, pointer bridge.MemPointer) [][]byte {
	if pointer.IsNull() {
		return [][]byte{}
	}
	encodedPointers := GetValue(ctx, module, pointer)
	var values [][]byte
	valPointers := bridge.UnpackPointers(encodedPointers)
	for _, p := range valPointers {
		v := GetValue(ctx, module, p)
		values = append(values, v)
	}
	return values
}

func PutArgs(ctx context.Context, module wz_api.Module, args [][]byte) bridge.MemPointer {
	return PutValues(ctx, module, args)
}

func GetArgs(ctx context.Context, module wz_api.Module, pointer bridge.MemPointer) [][]byte {
	return GetValues(ctx, module, pointer)
}

func PutReturn(ctx context.Context, module wz_api.Module, retValues [][]byte) bridge.MemPointer {
	return PutValues(ctx, module, retValues)
}

func GetReturn(ctx context.Context, module wz_api.Module, retPointer bridge.MemPointer) [][]byte {
	return GetValues(ctx, module, retPointer)
}

func PutReturnWithError(ctx context.Context, module wz_api.Module, retValues [][]byte, retErr error) bridge.MemPointer {
	if retErr == nil {
		errFlag := []byte{bridge.Err_Success}
		retValues = append([][]byte{errFlag}, retValues...)
	} else {
		errFlag := []byte{bridge.Err_Error}
		errMsg := []byte(retErr.Error())
		retValues = append([][]byte{errFlag, errMsg}, retValues...)
	}
	return PutValues(ctx, module, retValues)
}

func GetReturnWithError(ctx context.Context, module wz_api.Module, retPointer bridge.MemPointer) ([][]byte, error) {
	retValues := GetReturn(ctx, module, retPointer)
	if len(retValues) == 0 {
		return [][]byte{}, nil
	}
	if retValues[0][0] == bridge.Err_Success {
		return retValues[1:], nil
	} else {
		return retValues[2:], errors.New(string(retValues[1]))
	}
}

func DisabledBridge(ctx context.Context, module wz_api.Module, pointer uint64) uint64 {
	panic("go: disabled bridge -- this likely means you are trying to access the concrete API from a wasm precompile declared as pure")
}

func BridgeCallStateDB(ctx context.Context, module wz_api.Module, pointer uint64, db api.StateDB) uint64 {
	args := GetArgs(ctx, module, bridge.MemPointer(pointer))
	var opcode bridge.OpCode
	opcode.Decode(args[0])
	args = args[1:]
	out := CallStateDB(db, opcode, args)
	return PutValue(ctx, module, out).Uint64()
}

func CallStateDB(db api.StateDB, opcode bridge.OpCode, args [][]byte) []byte {
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

func BridgeCallEVM(ctx context.Context, module wz_api.Module, pointer uint64, evm api.EVM) uint64 {
	args := GetArgs(ctx, module, bridge.MemPointer(pointer))
	var opcode bridge.OpCode
	opcode.Decode(args[0])
	args = args[1:]
	out := callEVM(evm, opcode, args)
	return PutValue(ctx, module, out).Uint64()
}

func callEVM(evm api.EVM, opcode bridge.OpCode, args [][]byte) []byte {
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
	}

	return nil
}

func BridgeAddress(ctx context.Context, module wz_api.Module, pointer uint64, addr common.Address) uint64 {
	return PutValue(ctx, module, addr.Bytes()).Uint64()
}

func BridgeLog(ctx context.Context, module wz_api.Module, pointer uint64) uint64 {
	msg := GetValue(ctx, module, bridge.MemPointer(pointer))
	log.Debug("wasm:", string(msg))
	return bridge.NullPointer.Uint64()
}
