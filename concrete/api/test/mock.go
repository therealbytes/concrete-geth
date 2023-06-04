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

package test

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/concrete/api"
)

type MockStateDB struct {
	persistentState     map[common.Address]map[common.Hash]common.Hash
	ephemeralState      map[common.Address]map[common.Hash]common.Hash
	persistentPreimages map[common.Hash][]byte
	ephemeralPreimages  map[common.Hash][]byte
}

func NewMockStateDB() api.StateDB {
	return &MockStateDB{
		persistentState:     make(map[common.Address]map[common.Hash]common.Hash),
		ephemeralState:      make(map[common.Address]map[common.Hash]common.Hash),
		persistentPreimages: make(map[common.Hash][]byte),
		ephemeralPreimages:  make(map[common.Hash][]byte),
	}
}

func (m *MockStateDB) SetPersistentState(addr common.Address, key, value common.Hash) {
	if _, ok := m.persistentState[addr]; !ok {
		m.persistentState[addr] = make(map[common.Hash]common.Hash)
	}
	m.persistentState[addr][key] = value
}

func (m *MockStateDB) GetPersistentState(addr common.Address, key common.Hash) common.Hash {
	if _, ok := m.persistentState[addr]; !ok {
		return common.Hash{}
	}
	return m.persistentState[addr][key]
}

func (m *MockStateDB) SetEphemeralState(addr common.Address, key, value common.Hash) {
	if _, ok := m.ephemeralState[addr]; !ok {
		m.ephemeralState[addr] = make(map[common.Hash]common.Hash)
	}
	m.ephemeralState[addr][key] = value
}

func (m *MockStateDB) GetEphemeralState(addr common.Address, key common.Hash) common.Hash {
	if _, ok := m.ephemeralState[addr]; !ok {
		return common.Hash{}
	}
	return m.ephemeralState[addr][key]
}

func (m *MockStateDB) AddPersistentPreimage(hash common.Hash, preimage []byte) {
	pi := make([]byte, len(preimage))
	copy(pi, preimage)
	m.persistentPreimages[hash] = pi
}

func (m *MockStateDB) GetPersistentPreimage(hash common.Hash) []byte {
	if pi, ok := m.persistentPreimages[hash]; ok {
		return pi
	}
	return []byte{}
}

func (m *MockStateDB) GetPersistentPreimageSize(hash common.Hash) int {
	return len(m.persistentPreimages[hash])
}

func (m *MockStateDB) AddEphemeralPreimage(hash common.Hash, preimage []byte) {
	pi := make([]byte, len(preimage))
	copy(pi, preimage)
	m.ephemeralPreimages[hash] = pi
}

func (m *MockStateDB) GetEphemeralPreimage(hash common.Hash) []byte {
	if pi, ok := m.ephemeralPreimages[hash]; ok {
		return pi
	}
	return []byte{}
}

func (m *MockStateDB) GetEphemeralPreimageSize(hash common.Hash) int {
	return len(m.ephemeralPreimages[hash])
}

var _ api.StateDB = &MockStateDB{}

type mockEVM struct {
	db api.StateDB
}

func NewMockEVM(db api.StateDB) api.EVM {
	return &mockEVM{
		db: db,
	}
}

func (m *mockEVM) StateDB() api.StateDB                 { return m.db }
func (m *mockEVM) BlockHash(block *big.Int) common.Hash { return common.Hash{} }
func (m *mockEVM) BlockTimestamp() uint64               { return 0 }
func (m *mockEVM) BlockGasLimit() uint64                { return 0 }
func (m *mockEVM) BlockNumber() *big.Int                { return common.Big0 }
func (m *mockEVM) BlockDifficulty() *big.Int            { return common.Big0 }
func (m *mockEVM) BlockCoinbase() common.Address        { return common.Address{} }

var _ api.EVM = &mockEVM{}
