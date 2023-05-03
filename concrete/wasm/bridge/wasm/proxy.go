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

type CachedProxyStateDB struct {
	ProxyStateDB
	persistentState          map[common.Address]map[common.Hash]common.Hash
	persistentStateDirty     map[common.Address]map[common.Hash]struct{}
	ephemeralState           map[common.Address]map[common.Hash]common.Hash
	ephemeralStateDirty      map[common.Address]map[common.Hash]struct{}
	persistentPreimages      map[common.Hash][]byte
	persistentPreimagesDirty map[common.Hash]struct{}
	ephemeralPreimages       map[common.Hash][]byte
	ephemeralPreimagesDirty  map[common.Hash]struct{}
}

func NewCachedProxyStateDB(memory bridge.Memory, allocator bridge.Allocator, stateDBCaller HostFuncCaller) *CachedProxyStateDB {
	p := &CachedProxyStateDB{}
	p.memory = memory
	p.allocator = allocator
	p.caller = stateDBCaller
	p.persistentState = make(map[common.Address]map[common.Hash]common.Hash)
	p.persistentStateDirty = make(map[common.Address]map[common.Hash]struct{})
	p.ephemeralState = make(map[common.Address]map[common.Hash]common.Hash)
	p.ephemeralStateDirty = make(map[common.Address]map[common.Hash]struct{})
	p.persistentPreimages = make(map[common.Hash][]byte)
	p.persistentPreimagesDirty = make(map[common.Hash]struct{})
	p.ephemeralPreimages = make(map[common.Hash][]byte)
	p.ephemeralPreimagesDirty = make(map[common.Hash]struct{})
	return p
}

func (p *CachedProxyStateDB) SetPersistentState(addr common.Address, key, value common.Hash) {
	if _, ok := p.persistentState[addr]; !ok {
		p.persistentState[addr] = make(map[common.Hash]common.Hash)
		p.persistentStateDirty[addr] = make(map[common.Hash]struct{})
	}
	p.persistentState[addr][key] = value
	p.persistentStateDirty[addr][key] = struct{}{}
}

func (p *CachedProxyStateDB) GetPersistentState(addr common.Address, key common.Hash) common.Hash {
	if _, ok := p.persistentState[addr]; ok {
		if value, ok := p.persistentState[addr][key]; ok {
			return value
		}
	}
	value := p.ProxyStateDB.GetPersistentState(addr, key)
	p.SetPersistentState(addr, key, value)
	return value
}

func (p *CachedProxyStateDB) SetEphemeralState(addr common.Address, key common.Hash, value common.Hash) {
	if _, ok := p.ephemeralState[addr]; !ok {
		p.ephemeralState[addr] = make(map[common.Hash]common.Hash)
		p.ephemeralStateDirty[addr] = make(map[common.Hash]struct{})
	}
	p.ephemeralState[addr][key] = value
	p.ephemeralStateDirty[addr][key] = struct{}{}
}

func (p *CachedProxyStateDB) GetEphemeralState(addr common.Address, key common.Hash) common.Hash {
	if _, ok := p.ephemeralState[addr]; ok {
		if value, ok := p.ephemeralState[addr][key]; ok {
			return value
		}
	}
	value := p.ProxyStateDB.GetEphemeralState(addr, key)
	p.SetEphemeralState(addr, key, value)
	return value
}

func (p *CachedProxyStateDB) AddPersistentPreimage(hash common.Hash, preimage []byte) {
	if _, ok := p.persistentPreimages[hash]; !ok {
		p.persistentPreimages[hash] = preimage
		p.persistentPreimagesDirty[hash] = struct{}{}
	}
}

func (p *CachedProxyStateDB) GetPersistentPreimage(hash common.Hash) []byte {
	if _, ok := p.persistentPreimages[hash]; ok {
		return p.persistentPreimages[hash]
	}
	preimage := p.ProxyStateDB.GetPersistentPreimage(hash)
	p.AddPersistentPreimage(hash, preimage)
	return preimage
}

func (p *CachedProxyStateDB) GetPersistentPreimageSize(hash common.Hash) int {
	if _, ok := p.persistentPreimages[hash]; ok {
		return len(p.persistentPreimages[hash])
	}
	return p.ProxyStateDB.GetPersistentPreimageSize(hash)
}

func (p *CachedProxyStateDB) AddEphemeralPreimage(hash common.Hash, preimage []byte) {
	if _, ok := p.ephemeralPreimages[hash]; !ok {
		p.ephemeralPreimages[hash] = preimage
		p.ephemeralPreimagesDirty[hash] = struct{}{}
	}
}

func (p *CachedProxyStateDB) GetEphemeralPreimage(hash common.Hash) []byte {
	if _, ok := p.ephemeralPreimages[hash]; ok {
		return p.ephemeralPreimages[hash]
	}
	preimage := p.ProxyStateDB.GetEphemeralPreimage(hash)
	p.AddEphemeralPreimage(hash, preimage)
	return preimage
}

func (p *CachedProxyStateDB) GetEphemeralPreimageSize(hash common.Hash) int {
	if _, ok := p.ephemeralPreimages[hash]; ok {
		return len(p.ephemeralPreimages[hash])
	}
	return p.ProxyStateDB.GetEphemeralPreimageSize(hash)
}

func (p *CachedProxyStateDB) Commit() {
	for addr, dirties := range p.persistentStateDirty {
		for key := range dirties {
			p.ProxyStateDB.SetPersistentState(addr, key, p.persistentState[addr][key])
		}
	}
	for addr, dirties := range p.ephemeralStateDirty {
		for key := range dirties {
			p.ProxyStateDB.SetEphemeralState(addr, key, p.ephemeralState[addr][key])
		}
	}
	for hash := range p.persistentPreimagesDirty {
		p.ProxyStateDB.AddPersistentPreimage(hash, p.persistentPreimages[hash])
	}
	for hash := range p.ephemeralPreimagesDirty {
		p.ProxyStateDB.AddEphemeralPreimage(hash, p.ephemeralPreimages[hash])
	}
	if len(p.persistentStateDirty) > 0 {
		p.persistentStateDirty = make(map[common.Address]map[common.Hash]struct{})
	}
	if len(p.ephemeralStateDirty) > 0 {
		p.ephemeralStateDirty = make(map[common.Address]map[common.Hash]struct{})
	}
	if len(p.persistentPreimagesDirty) > 0 {
		p.persistentPreimagesDirty = make(map[common.Hash]struct{})
	}
	if len(p.ephemeralPreimagesDirty) > 0 {
		p.ephemeralPreimagesDirty = make(map[common.Hash]struct{})
	}
}

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

type CachedProxyEVM struct {
	ProxyEVM
	db          *CachedProxyStateDB
	block       *bridge.BlockData
	blockHashes map[uint64]common.Hash
}

func NewCachedProxyEVM(memory bridge.Memory, allocator bridge.Allocator, evmCaller HostFuncCaller, stateDBCaller HostFuncCaller) *CachedProxyEVM {
	return &CachedProxyEVM{
		ProxyEVM: ProxyEVM{Proxy: Proxy{memory: memory, allocator: allocator, caller: evmCaller}},
		db:       NewCachedProxyStateDB(memory, allocator, stateDBCaller),
	}
}

func (p *CachedProxyEVM) StateDB() api.StateDB {
	return p.db
}

func (p *CachedProxyEVM) getBlock() {
	if p.block == nil {
		data := p.call(bridge.Op_EVM_Block.Encode())
		p.block = &bridge.BlockData{}
		p.block.Decode(data)
	}
}

func (p *CachedProxyEVM) BlockHash(block *big.Int) common.Hash {
	blockNum := block.Uint64()
	if p.blockHashes == nil {
		p.blockHashes = make(map[uint64]common.Hash)
	}
	if hash, ok := p.blockHashes[blockNum]; ok {
		return hash
	}
	hash := p.ProxyEVM.BlockHash(block)
	p.blockHashes[blockNum] = hash
	return hash
}

func (p *CachedProxyEVM) BlockTimestamp() *big.Int {
	p.getBlock()
	return p.block.Timestamp
}

func (p *CachedProxyEVM) BlockNumber() *big.Int {
	p.getBlock()
	return p.block.Number
}

func (p *CachedProxyEVM) BlockDifficulty() *big.Int {
	p.getBlock()
	return p.block.Difficulty
}

func (p *CachedProxyEVM) BlockGasLimit() *big.Int {
	p.getBlock()
	return p.block.GasLimit
}

func (p *CachedProxyEVM) BlockCoinbase() common.Address {
	p.getBlock()
	return p.block.Coinbase
}

func (p *CachedProxyEVM) Commit() {
	p.db.Commit()
}
