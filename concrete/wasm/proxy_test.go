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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	cc_api "github.com/ethereum/go-ethereum/concrete/api"
	cc_api_test "github.com/ethereum/go-ethereum/concrete/api/test"
	"github.com/ethereum/go-ethereum/concrete/wasm/bridge"
	"github.com/ethereum/go-ethereum/concrete/wasm/bridge/native"
	"github.com/ethereum/go-ethereum/concrete/wasm/bridge/wasm"
	"github.com/stretchr/testify/require"
)

func newStateDBBridgeFunc(memory wasm.Memory, db cc_api.StateDB) wasm.WasmBridgeFunc {
	return func(pointer uint64) uint64 {
		args := wasm.GetArgs(memory, bridge.MemPointer(pointer))
		var opcode bridge.OpCode
		opcode.Decode(args[0])
		args = args[1:]
		out := native.CallStateDB(db, opcode, args)
		return wasm.PutValue(memory, out).Uint64()
	}
}

func newProxyStateDB(db cc_api.StateDB) cc_api.StateDB {
	mem := wasm.NewMockMemory()
	bridgeFunc := newStateDBBridgeFunc(mem, db)
	return wasm.NewProxyStateDB(mem, bridgeFunc)
}

type readWriteStorage struct {
	read, write cc_api.Storage
}

func NewReadWriteStorage(read, write cc_api.Storage) cc_api.Storage {
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

func TestStateDBBProxy(t *testing.T) {
	address := common.HexToAddress("0x01")
	statedb := cc_api_test.NewMockStateDB()
	proxy := newProxyStateDB(statedb)
	stateApi := cc_api.NewStateAPI(statedb, address)
	proxyStateApi := cc_api.NewStateAPI(proxy, address)

	persistent := stateApi.Persistent()
	proxyPersistent := proxyStateApi.Persistent()
	ephemeral := stateApi.Ephemeral()
	proxyEphemeral := proxyStateApi.Ephemeral()

	// Test persistent methods
	cc_api_test.TestStorage(t, NewReadWriteStorage(persistent, proxyPersistent))
	cc_api_test.TestStorage(t, NewReadWriteStorage(proxyPersistent, persistent))

	// Test ephemeral methods
	cc_api_test.TestStorage(t, NewReadWriteStorage(ephemeral, proxyEphemeral))
	cc_api_test.TestStorage(t, NewReadWriteStorage(proxyEphemeral, ephemeral))

	// Fuzz proxy
	cc_api_test.FuzzStorage(t, proxyPersistent)
	cc_api_test.FuzzStorage(t, proxyEphemeral)
}

type mockEVM struct {
	db cc_api.StateDB
}

func newEVMStub(db cc_api.StateDB) cc_api.EVM {
	return &mockEVM{
		db: db,
	}
}

func (m *mockEVM) StateDB() cc_api.StateDB              { return m.db }
func (m *mockEVM) BlockHash(block *big.Int) common.Hash { return common.Hash{2} }
func (m *mockEVM) BlockTimestamp() *big.Int             { return common.Big2 }
func (m *mockEVM) BlockNumber() *big.Int                { return common.Big2 }
func (m *mockEVM) BlockDifficulty() *big.Int            { return common.Big2 }
func (m *mockEVM) BlockGasLimit() *big.Int              { return common.Big2 }
func (m *mockEVM) BlockCoinbase() common.Address        { return common.Address{2} }

var _ cc_api.EVM = &mockEVM{}

func newEVMBridgeFunc(memory wasm.Memory, evm cc_api.EVM) wasm.WasmBridgeFunc {
	return func(pointer uint64) uint64 {
		args := wasm.GetArgs(memory, bridge.MemPointer(pointer))
		var opcode bridge.OpCode
		opcode.Decode(args[0])
		args = args[1:]
		out := native.CallEVM(evm, opcode, args)
		return wasm.PutValue(memory, out).Uint64()
	}
}

func newProxyEVM(evm cc_api.EVM) cc_api.EVM {
	mem := wasm.NewMockMemory()
	stateDBBridgeFunc := newStateDBBridgeFunc(mem, evm.StateDB())
	evmBridgeFunc := newEVMBridgeFunc(mem, evm)
	return wasm.NewProxyEVM(mem, evmBridgeFunc, stateDBBridgeFunc)
}

func TestEVMBridge(t *testing.T) {
	db := cc_api_test.NewMockStateDB()
	evm := newEVMStub(db)
	proxy := newProxyEVM(evm)

	require.Equal(t, evm.BlockHash(common.Big1), proxy.BlockHash(common.Big1))
	require.Equal(t, evm.BlockTimestamp(), proxy.BlockTimestamp())
	require.Equal(t, evm.BlockNumber(), proxy.BlockNumber())
	require.Equal(t, evm.BlockDifficulty(), proxy.BlockDifficulty())
	require.Equal(t, evm.BlockGasLimit(), proxy.BlockGasLimit())
	require.Equal(t, evm.BlockCoinbase(), proxy.BlockCoinbase())
}
