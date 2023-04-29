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
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/concrete/api"
	"github.com/ethereum/go-ethereum/concrete/wasm/bridge"
)

type WasmBridgeFunc func(pointer uint64) uint64

type Memory interface {
	Ref(data []byte) bridge.MemPointer
	Deref(pointer bridge.MemPointer) []byte
}

func PutValue(memory Memory, value []byte) bridge.MemPointer {
	return memory.Ref(value)
}

func GetValue(memory Memory, pointer bridge.MemPointer) []byte {
	return memory.Deref(pointer)
}

func PutValues(memory Memory, values [][]byte) bridge.MemPointer {
	if len(values) == 0 {
		return bridge.NullPointer
	}
	var pointers []bridge.MemPointer
	for _, v := range values {
		pointers = append(pointers, PutValue(memory, v))
	}
	packedPointers := bridge.PackPointers(pointers)
	return PutValue(memory, packedPointers)
}

func GetValues(memory Memory, pointer bridge.MemPointer) [][]byte {
	if pointer.IsNull() {
		return [][]byte{}
	}
	var values [][]byte
	valPointers := bridge.UnpackPointers(GetValue(memory, pointer))
	for _, p := range valPointers {
		values = append(values, GetValue(memory, p))
	}
	return values
}

func PutArgs(memory Memory, args [][]byte) bridge.MemPointer {
	return PutValues(memory, args)
}

func GetArgs(memory Memory, pointer bridge.MemPointer) [][]byte {
	return GetValues(memory, pointer)
}

func PutReturn(memory Memory, retValues [][]byte) bridge.MemPointer {
	return PutValues(memory, retValues)
}

func GetReturn(memory Memory, retPointer bridge.MemPointer) [][]byte {
	return GetValues(memory, retPointer)
}

func PutReturnWithError(memory Memory, retValues [][]byte, retErr error) bridge.MemPointer {
	if retErr == nil {
		errFlag := []byte{bridge.Err_Success}
		retValues = append([][]byte{errFlag}, retValues...)
	} else {
		errFlag := []byte{bridge.Err_Error}
		errMsg := []byte(retErr.Error())
		retValues = append([][]byte{errFlag, errMsg}, retValues...)
	}
	return PutReturn(memory, retValues)
}

func GetReturnWithError(memory Memory, retPointer bridge.MemPointer) ([][]byte, error) {
	retValues := GetReturn(memory, retPointer)
	if len(retValues) == 0 {
		return nil, nil
	}
	if retValues[0][0] == bridge.Err_Success {
		return retValues[1:], nil
	} else {
		return retValues[2:], errors.New(string(retValues[1]))
	}
}

type Proxy struct {
	memory     Memory
	bridgeFunc WasmBridgeFunc
}

func (p *Proxy) call(args ...[]byte) []byte {
	argsPointer := PutArgs(p.memory, args)
	retPointer := bridge.MemPointer(p.bridgeFunc(argsPointer.Uint64()))
	retValue := GetValue(p.memory, retPointer)
	return retValue
}

type ProxyStateDB struct {
	Proxy
}

func NewProxyStateDB(memory Memory, stateDBBridge WasmBridgeFunc) *ProxyStateDB {
	return &ProxyStateDB{Proxy{memory: memory, bridgeFunc: stateDBBridge}}
}

func (p *ProxyStateDB) SetPersistentState(addr common.Address, key, value common.Hash) {
	p.call(bridge.Op_StateDB_SetPersistentState.Encode(),
		addr.Bytes(),
		key.Bytes(),
		value.Bytes(),
	)
}

func (p *ProxyStateDB) GetPersistentState(addr common.Address, key common.Hash) common.Hash {
	retValue := p.call(
		bridge.Op_StateDB_GetPersistentState.Encode(),
		addr.Bytes(),
		key.Bytes(),
	)
	return common.BytesToHash(retValue)
}

func (p *ProxyStateDB) SetEphemeralState(addr common.Address, key common.Hash, value common.Hash) {
	p.call(bridge.Op_StateDB_SetEphemeralState.Encode(),
		addr.Bytes(),
		key.Bytes(),
		value.Bytes(),
	)
}

func (p *ProxyStateDB) GetEphemeralState(addr common.Address, key common.Hash) common.Hash {
	retValue := p.call(
		bridge.Op_StateDB_GetEphemeralState.Encode(),
		addr.Bytes(),
		key.Bytes(),
	)
	return common.BytesToHash(retValue)
}

func (p *ProxyStateDB) AddPersistentPreimage(hash common.Hash, preimage []byte) {
	p.call(
		bridge.Op_StateDB_AddPersistentPreimage.Encode(),
		hash.Bytes(),
		preimage,
	)
}

func (p *ProxyStateDB) GetPersistentPreimage(hash common.Hash) []byte {
	retValue := p.call(
		bridge.Op_StateDB_GetPersistentPreimage.Encode(),
		hash.Bytes(),
	)
	return retValue
}

func (p *ProxyStateDB) GetPersistentPreimageSize(hash common.Hash) int {
	retValue := p.call(
		bridge.Op_StateDB_GetPersistentPreimageSize.Encode(),
		hash.Bytes(),
	)
	return int(bridge.BytesToUint64(retValue))
}

func (p *ProxyStateDB) AddEphemeralPreimage(hash common.Hash, preimage []byte) {
	p.call(
		bridge.Op_StateDB_AddEphemeralPreimage.Encode(),
		hash.Bytes(),
		preimage,
	)
}

func (p *ProxyStateDB) GetEphemeralPreimage(hash common.Hash) []byte {
	return p.call(
		bridge.Op_StateDB_GetEphemeralPreimage.Encode(),
		hash.Bytes(),
	)
}

func (p *ProxyStateDB) GetEphemeralPreimageSize(hash common.Hash) int {
	retValue := p.call(
		bridge.Op_StateDB_GetEphemeralPreimageSize.Encode(),
		hash.Bytes(),
	)
	return int(bridge.BytesToUint64(retValue))
}

var _ api.StateDB = (*ProxyStateDB)(nil)

type ProxyEVM struct {
	Proxy
	db *ProxyStateDB
}

func NewProxyEVM(memory Memory, evmBridge WasmBridgeFunc, stateDBBridge WasmBridgeFunc) *ProxyEVM {
	return &ProxyEVM{
		Proxy: Proxy{memory: memory, bridgeFunc: evmBridge},
		db:    NewProxyStateDB(memory, stateDBBridge),
	}
}

func (p *ProxyEVM) StateDB() api.StateDB {
	return p.db
}

func (p *ProxyEVM) BlockHash(block *big.Int) common.Hash {
	retValue := p.call(
		bridge.Op_EVM_BlockHash.Encode(),
		block.Bytes(),
	)
	return common.BytesToHash(retValue)
}

func (p *ProxyEVM) BlockTimestamp() *big.Int {
	retValue := p.call(bridge.Op_EVM_BlockTimestamp.Encode())
	return new(big.Int).SetBytes(retValue)
}

func (p *ProxyEVM) BlockNumber() *big.Int {
	retValue := p.call(bridge.Op_EVM_BlockNumber.Encode())
	return new(big.Int).SetBytes(retValue)
}

func (p *ProxyEVM) BlockDifficulty() *big.Int {
	retValue := p.call(bridge.Op_EVM_BlockDifficulty.Encode())
	return new(big.Int).SetBytes(retValue)
}

func (p *ProxyEVM) BlockGasLimit() *big.Int {
	retValue := p.call(bridge.Op_EVM_BlockGasLimit.Encode())
	return new(big.Int).SetBytes(retValue)
}

func (p *ProxyEVM) BlockCoinbase() common.Address {
	retValue := p.call(bridge.Op_EVM_BlockCoinbase.Encode())
	return common.BytesToAddress(retValue)
}

var _ api.EVM = (*ProxyEVM)(nil)
