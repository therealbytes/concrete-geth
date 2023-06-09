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
	"github.com/ethereum/go-ethereum/concrete/api"
	api_test "github.com/ethereum/go-ethereum/concrete/api/test"
	"github.com/ethereum/go-ethereum/concrete/wasm/bridge"
	"github.com/ethereum/go-ethereum/concrete/wasm/bridge/host"
	"github.com/ethereum/go-ethereum/concrete/wasm/bridge/wasm"
	"github.com/stretchr/testify/require"
)

func newStateDBHostFunc(memory bridge.Memory, db api.StateDB) wasm.HostFuncCaller {
	return func(pointer uint64) uint64 {
		args := bridge.GetArgs(memory, bridge.MemPointer(pointer))
		var opcode bridge.OpCode
		opcode.Decode(args[0])
		args = args[1:]
		out := host.CallStateDB(db, opcode, args)
		return bridge.PutValue(memory, out).Uint64()
	}
}

func newProxyStateDB(db api.StateDB) api.StateDB {
	memory := newMockMemory()
	alloc := newMockAllocator()
	caller := newStateDBHostFunc(memory, db)
	return wasm.NewProxyStateDB(memory, alloc, caller)
}

type readWriteStorage struct {
	read, write api.Storage
}

func newReadWriteStorage(read, write api.Storage) api.Storage {
	return &readWriteStorage{
		read:  read,
		write: write,
	}
}

func (s *readWriteStorage) StateDB() api.StateDB {
	panic("not implemented")
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

func (s *readWriteStorage) AddPreimage(preimage []byte) common.Hash {
	return s.write.AddPreimage(preimage)
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
	var (
		address       = common.HexToAddress("0x01")
		statedb       = newTestStateDB()
		proxy         = newProxyStateDB(statedb)
		stateAPI      = api.NewStateAPI(statedb, address)
		proxyStateAPI = api.NewStateAPI(proxy, address)

		persistent      = stateAPI.Persistent()
		proxyPersistent = proxyStateAPI.Persistent()
		ephemeral       = stateAPI.Ephemeral()
		proxyEphemeral  = proxyStateAPI.Ephemeral()
	)

	// Test persistent methods
	api_test.TestStorage(t, newReadWriteStorage(persistent, proxyPersistent))
	api_test.TestStorage(t, newReadWriteStorage(proxyPersistent, persistent))

	// Test ephemeral methods
	api_test.TestStorage(t, newReadWriteStorage(ephemeral, proxyEphemeral))
	api_test.TestStorage(t, newReadWriteStorage(proxyEphemeral, ephemeral))

	// Fuzz proxy
	api_test.FuzzStorage(t, proxyPersistent)
	api_test.FuzzStorage(t, proxyEphemeral)
}

type mockEVM struct {
	db api.StateDB
}

func newEVMStub(db api.StateDB) api.EVM {
	return &mockEVM{
		db: db,
	}
}

func (m *mockEVM) StateDB() api.StateDB                 { return m.db }
func (m *mockEVM) BlockHash(block *big.Int) common.Hash { return common.Hash{2} }
func (m *mockEVM) BlockTimestamp() uint64               { return 0 }
func (m *mockEVM) BlockGasLimit() uint64                { return 0 }
func (m *mockEVM) BlockNumber() *big.Int                { return common.Big2 }
func (m *mockEVM) BlockDifficulty() *big.Int            { return common.Big2 }
func (m *mockEVM) BlockCoinbase() common.Address        { return common.Address{2} }

var _ api.EVM = &mockEVM{}

func newEVMHostFunc(memory bridge.Memory, evm api.EVM) wasm.HostFuncCaller {
	return func(pointer uint64) uint64 {
		args := bridge.GetArgs(memory, bridge.MemPointer(pointer))
		var opcode bridge.OpCode
		opcode.Decode(args[0])
		args = args[1:]
		out := host.CallEVM(evm, opcode, args)
		return bridge.PutValue(memory, out).Uint64()
	}
}

func newProxyEVM(evm api.EVM) api.EVM {
	memory := newMockMemory()
	alloc := newMockAllocator()
	stateDBCaller := newStateDBHostFunc(memory, evm.StateDB())
	evmCaller := newEVMHostFunc(memory, evm)
	return wasm.NewProxyEVM(memory, alloc, evmCaller, stateDBCaller)
}

func TestEVMProxy(t *testing.T) {
	var (
		statedb = api_test.NewMockStateDB()
		evm     = newEVMStub(statedb)
		proxy   = newProxyEVM(evm)
	)
	r := require.New(t)
	r.Equal(evm.BlockHash(common.Big1), proxy.BlockHash(common.Big1))
	r.Equal(evm.BlockTimestamp(), proxy.BlockTimestamp())
	r.Equal(evm.BlockNumber(), proxy.BlockNumber())
	r.Equal(evm.BlockDifficulty(), proxy.BlockDifficulty())
	r.Equal(evm.BlockGasLimit(), proxy.BlockGasLimit())
	r.Equal(evm.BlockCoinbase(), proxy.BlockCoinbase())
}

func TestBlockEncodeDecode(t *testing.T) {
	block1 := bridge.BlockData{
		Timestamp:  0,
		GasLimit:   0,
		Number:     common.Big2,
		Difficulty: common.Big3,
		Coinbase:   common.Address{1},
	}
	encoded := block1.Encode()
	block2 := bridge.BlockData{}
	block2.Decode(encoded)
	require.Equal(t, block1, block2)
}
