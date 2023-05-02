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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/concrete/api"
	"github.com/ethereum/go-ethereum/concrete/wasm/bridge"
)

type HostFuncCaller func(pointer uint64) uint64

func Call_BytesArr_Bytes(memory bridge.Memory, allocator bridge.Allocator, caller HostFuncCaller, args ...[]byte) []byte {
	argsPointer := bridge.PutArgs(memory, args)
	retPointer := bridge.MemPointer(caller(argsPointer.Uint64()))
	retValue := bridge.GetValue(memory, retPointer)
	if !retPointer.IsNull() {
		allocator.Free(retPointer)
	}
	return retValue
}

type Proxy struct {
	memory    bridge.Memory
	allocator bridge.Allocator
	caller    HostFuncCaller
}

func (p *Proxy) call(args ...[]byte) []byte {
	return Call_BytesArr_Bytes(p.memory, p.allocator, p.caller, args...)
}

type ProxyStateDB struct {
	Proxy
}

func NewProxyStateDB(memory bridge.Memory, allocator bridge.Allocator, stateDBCaller HostFuncCaller) *ProxyStateDB {
	return &ProxyStateDB{Proxy{memory: memory, allocator: allocator, caller: stateDBCaller}}
}

func (p *ProxyStateDB) SetPersistentState(addr common.Address, key, value common.Hash) {
	p.call(
		bridge.Op_StateDB_SetPersistentState.Encode(),
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

func NewProxyEVM(memory bridge.Memory, allocator bridge.Allocator, evmCaller HostFuncCaller, stateDBCaller HostFuncCaller) *ProxyEVM {
	return &ProxyEVM{
		Proxy: Proxy{memory: memory, allocator: allocator, caller: evmCaller},
		db:    NewProxyStateDB(memory, allocator, stateDBCaller),
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
