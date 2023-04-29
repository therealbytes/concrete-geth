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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/concrete/api"
	"github.com/ethereum/go-ethereum/concrete/wasm/bridge"
	"github.com/ethereum/go-ethereum/concrete/wasm/bridge/native"
	"github.com/ethereum/go-ethereum/concrete/wasm/bridge/wasm"
)

func newBridgeFunc(memory wasm.Memory, db api.StateDB) wasm.WasmBridgeFunc {
	return func(pointer uint64) uint64 {
		args := wasm.GetArgs(memory, bridge.MemPointer(pointer))
		var opcode bridge.OpCode
		opcode.Decode(args[0])
		args = args[1:]
		out := native.CallStateDB(db, opcode, args)
		return wasm.PutValue(memory, out).Uint64()
	}
}

func newProxyStateDB(db api.StateDB) api.StateDB {
	mem := wasm.NewMockMemory()
	bridgeFunc := newBridgeFunc(mem, db)
	return wasm.NewProxyStateDB(mem, bridgeFunc)
}

type readWriteStorage struct {
	read, write api.Storage
}

func NewReadWriteStorage(read, write api.Storage) api.Storage {
	return &readWriteStorage{
		read:  read,
		write: write,
	}
}

func (s *readWriteStorage) Address() common.Address {
	return s.read.Address()
}

func (s *readWriteStorage) Set(key common.Hash, value common.Hash) {
	s.write.Set(key, value)
}

func (s *readWriteStorage) Get(key common.Hash) common.Hash {
	return s.read.Get(key)
}

func (s *readWriteStorage) AddPreimage(preimage []byte) {
	s.write.AddPreimage(preimage)
}

func (s *readWriteStorage) HasPreimage(hash common.Hash) bool {
	return s.read.HasPreimage(hash)
}

func (s *readWriteStorage) GetPreimage(hash common.Hash) []byte {
	return s.read.GetPreimage(hash)
}

func (s *readWriteStorage) GetPreimageSize(hash common.Hash) int {
	return s.read.GetPreimageSize(hash)
}

func TestStateDBBridge(t *testing.T) {
	address := common.HexToAddress("0x01")
	statedb := api.NewMockStateDB()
	proxy := newProxyStateDB(statedb)
	stateApi := api.NewStateAPI(statedb, address)
	proxyStateApi := api.NewStateAPI(proxy, address)

	persistent := stateApi.Persistent()
	proxyPersistent := proxyStateApi.Persistent()
	ephemeral := stateApi.Ephemeral()
	proxyEphemeral := proxyStateApi.Ephemeral()

	// Test persistent methods
	api.TestStorage(t, NewReadWriteStorage(persistent, proxyPersistent))
	api.TestStorage(t, NewReadWriteStorage(proxyPersistent, persistent))

	// Test ephemeral methods
	api.TestStorage(t, NewReadWriteStorage(ephemeral, proxyEphemeral))
	api.TestStorage(t, NewReadWriteStorage(proxyEphemeral, ephemeral))

	// Fuzz proxy
	api.FuzzStorage(t, proxyPersistent)
	api.FuzzStorage(t, proxyEphemeral)
}
