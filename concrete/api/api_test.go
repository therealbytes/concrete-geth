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

package api

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/require"
)

func TestReadOnlyStateDB(t *testing.T) {
	sdb := NewReadOnlyStateDB(NewMockStateDB())

	// Test type
	require.IsType(t, &readOnlyStateDB{}, sdb, "NewReadOnlyStateDB should return a readOnlyStateDB")

	// Test SetPersistentState panics
	require.Panics(t, func() {
		sdb.SetPersistentState(common.Address{}, common.Hash{}, common.Hash{})
	}, "stateDB write protection")

	// Test SetEphemeralState panics
	require.Panics(t, func() {
		sdb.SetEphemeralState(common.Address{}, common.Hash{}, common.Hash{})
	}, "stateDB write protection")

	// Test AddPersistentPreimage panics
	require.Panics(t, func() {
		sdb.AddPersistentPreimage(common.Hash{}, []byte{})
	}, "stateDB write protection")

	// Test AddEphemeralPreimage panics
	require.Panics(t, func() {
		sdb.AddEphemeralPreimage(common.Hash{}, []byte{})
	}, "stateDB write protection")

	// Test GetPersistentState
	require.NotPanics(t, func() {
		sdb.GetPersistentState(common.Address{}, common.Hash{})
	}, "GetPersistentState should not panic")

	// Test GetEphemeralState
	require.NotPanics(t, func() {
		sdb.GetEphemeralState(common.Address{}, common.Hash{})
	}, "GetEphemeralState should not panic")

	// Test GetPersistentPreimage
	require.NotPanics(t, func() {
		sdb.GetPersistentPreimage(common.Hash{})
	}, "GetPersistentPreimage should not panic")

	// Test GetPersistentPreimageSize
	require.NotPanics(t, func() {
		sdb.GetPersistentPreimageSize(common.Hash{})
	}, "GetPersistentPreimageSize should not panic")

	// Test GetEphemeralPreimage
	require.NotPanics(t, func() {
		sdb.GetEphemeralPreimage(common.Hash{})
	}, "GetEphemeralPreimage should not panic")

	// Test GetEphemeralPreimageSize
	require.NotPanics(t, func() {
		sdb.GetEphemeralPreimageSize(common.Hash{})
	}, "GetEphemeralPreimageSize should not panic")
}

func TestCommitSafeStateDB(t *testing.T) {
	sdb := NewCommitSafeStateDB(NewMockStateDB())

	// Test type
	require.IsType(t, &commitSafeStateDB{}, sdb, "NewCommitSafeStateDB should return a commitSafeStateDB")

	// Test SetPersistentState
	require.Panics(t, func() {
		sdb.SetPersistentState(common.Address{}, common.Hash{}, common.Hash{})
	}, "stateDB write protection")

	// Test SetEphemeralState
	require.NotPanics(t, func() {
		sdb.SetEphemeralState(common.Address{}, common.Hash{}, common.Hash{})
	}, "SetEphemeralState should not panic")

	// Test AddPersistentPreimage
	require.NotPanics(t, func() {
		sdb.AddPersistentPreimage(common.Hash{}, []byte{})
	}, "AddPersistentPreimage should not panic")

	// Test AddEphemeralPreimage
	require.NotPanics(t, func() {
		sdb.AddEphemeralPreimage(common.Hash{}, []byte{})
	}, "AddEphemeralPreimage should not panic")

	// Test GetPersistentState
	require.NotPanics(t, func() {
		sdb.GetPersistentState(common.Address{}, common.Hash{})
	}, "GetPersistentState should not panic")

	// Test GetEphemeralState
	require.NotPanics(t, func() {
		sdb.GetEphemeralState(common.Address{}, common.Hash{})
	}, "GetEphemeralState should not panic")

	// Test GetPersistentPreimage
	require.NotPanics(t, func() {
		sdb.GetPersistentPreimage(common.Hash{})
	}, "GetPersistentPreimage should not panic")

	// Test GetPersistentPreimageSize
	require.NotPanics(t, func() {
		sdb.GetPersistentPreimageSize(common.Hash{})
	}, "GetPersistentPreimageSize should not panic")

	// Test GetEphemeralPreimage
	require.NotPanics(t, func() {
		sdb.GetEphemeralPreimage(common.Hash{})
	}, "GetEphemeralPreimage should not panic")

	// Test GetEphemeralPreimageSize
	require.NotPanics(t, func() {
		sdb.GetEphemeralPreimageSize(common.Hash{})
	}, "GetEphemeralPreimageSize should not panic")
}

func TestReadOnlyEVM(t *testing.T) {
	evm := NewReadOnlyEVM(NewMockEVM(NewMockStateDB()))

	// Test type
	require.IsType(t, &readOnlyEVM{}, evm, "NewReadOnlyEVM should return a readOnlyEVM")

	// Test StateDB returns readOnlyStateDB
	sdb := evm.StateDB()
	require.IsType(t, &readOnlyStateDB{}, sdb, "StateDB should return readOnlyStateDB")
}

func TestCommitSafeEVM(t *testing.T) {
	evm := NewCommitSafeEVM(NewMockEVM(NewMockStateDB()))

	// Test type
	require.IsType(t, &commitSafeEVM{}, evm, "NewCommitSafeEVM should return a commitSafeEVM")

	// Test StateDB returns commitSafeStateDB
	sdb := evm.StateDB()
	require.IsType(t, &commitSafeStateDB{}, sdb, "StateDB should return commitSafeStateDB")
}

func testStorage(t *testing.T, storage Storage) {
	key := common.HexToHash("0x01")
	value := common.HexToHash("0x02")
	preimage := []byte("test preimage")
	preimageHash := crypto.Keccak256Hash(preimage)
	nonExistentKey := common.HexToHash("0x03")
	nonExistentPreimageHash := common.HexToHash("0x04")
	nilPreimageHash := crypto.Keccak256Hash(nil)

	// Test Get with non-existent key
	storedValue := storage.Get(nonExistentKey)
	require.Equal(t, common.Hash{}, storedValue, "Get should return an empty hash for non-existent key")

	// Test Set and Get
	storage.Set(key, value)
	storedValue = storage.Get(key)
	require.Equal(t, value, storedValue, "Set and Get should work correctly")

	// Test HasPreimage, GetPreimage, GetPreimageSize for non-existent preimage
	require.False(t, storage.HasPreimage(nonExistentPreimageHash), "HasPreimage should return false for non-existent preimage")
	storedPreimage := storage.GetPreimage(nonExistentPreimageHash)
	require.Nil(t, storedPreimage, "GetPreimage should return nil for non-existent preimage")
	preimageSize := storage.GetPreimageSize(nonExistentPreimageHash)
	require.Equal(t, -1, preimageSize, "GetPreimageSize should return -1 for non-existent preimage")

	// Test AddPreimage, HasPreimage, GetPreimage, GetPreimageSize for nil preimage
	storage.AddPreimage(nil)
	require.True(t, storage.HasPreimage(nilPreimageHash), "HasPreimage should return true")
	storedPreimage = storage.GetPreimage(nilPreimageHash)
	require.Equal(t, []byte{}, storedPreimage, "GetPreimage should return correct preimage")
	preimageSize = storage.GetPreimageSize(nilPreimageHash)
	require.Equal(t, 0, preimageSize, "GetPreimageSize should return correct size")

	// Test AddPreimage, HasPreimage, GetPreimage, GetPreimageSize
	storage.AddPreimage(preimage)
	require.True(t, storage.HasPreimage(preimageHash), "HasPreimage should return true")
	storedPreimage = storage.GetPreimage(preimageHash)
	require.Equal(t, preimage, storedPreimage, "GetPreimage should return correct preimage")
	preimageSize = storage.GetPreimageSize(preimageHash)
	require.Len(t, preimage, preimageSize, "GetPreimageSize should return correct size")
}

func fuzzStorage(t *testing.T, storage Storage) {
	f := fuzz.NewWithSeed(1)

	// Fuzz test Set and Get
	for i := 0; i < 100; i++ {
		var key, value common.Hash
		f.Fuzz(&key)
		f.Fuzz(&value)

		storage.Set(key, value)
		storedValue := storage.Get(key)
		require.Equal(t, value, storedValue, "Set and Get should work correctly")
	}

	// Fuzz test AddPreimage, HasPreimage, GetPreimage, GetPreimageSize
	for i := 0; i < 100; i++ {
		var preimage []byte
		f.Fuzz(&preimage)

		preimageHash := crypto.Keccak256Hash(preimage)

		storage.AddPreimage(preimage)
		require.True(t, storage.HasPreimage(preimageHash), "HasPreimage should return true")
		storedPreimage := storage.GetPreimage(preimageHash)
		if preimage == nil {
			require.Equal(t, []byte{}, storedPreimage, "GetPreimage should return correct preimage")
		} else {
			require.Equal(t, preimage, storedPreimage, "GetPreimage should return correct preimage")
		}
		preimageSize := storage.GetPreimageSize(preimageHash)
		require.Len(t, preimage, preimageSize, "GetPreimageSize should return correct size")
	}
}

func TestPersistentStorage(t *testing.T) {
	sdb := NewMockStateDB()
	address := common.HexToAddress("0x01")
	storage := &persistentStorage{address, sdb}
	testStorage(t, storage)
	fuzzStorage(t, storage)
}

func TestEphemeralStorage(t *testing.T) {
	sdb := NewMockStateDB()
	address := common.HexToAddress("0x01")
	storage := &ephemeralStorage{address, sdb}
	testStorage(t, storage)
	fuzzStorage(t, storage)
}

func TestStateAPI(t *testing.T) {
	address := common.HexToAddress("0x01")
	sdb := NewMockStateDB()
	api := NewStateAPI(sdb, address)

	// Test Address
	require.Equal(t, address, api.Address(), "Address should return correct address")

	// Test StateDB
	require.Equal(t, sdb, api.StateDB(), "StateDB should return correct StateDB")

	// Test EVM
	require.Equal(t, nil, api.EVM(), "EVM should return nil")

	// Test Persistent
	persistent := api.Persistent()
	require.NotNil(t, persistent, "Persistent should not be nil")
	require.IsType(t, &datastore{}, persistent, "Persistent should return a datastore instance")
	persistentStruct, _ := persistent.(*datastore)
	require.IsType(t, &persistentStorage{}, persistentStruct.Storage, "Persistent should return a PersistentStorage instance")

	// Test Ephemeral
	ephemeral := api.Ephemeral()
	require.NotNil(t, ephemeral, "Ephemeral should not be nil")
	require.IsType(t, &datastore{}, ephemeral, "Ephemeral should return a datastore instance")
	ephemeralStruct, _ := ephemeral.(*datastore)
	require.IsType(t, &ephemeralStorage{}, ephemeralStruct.Storage, "Ephemeral should return a EphemeralStorage instance")

	// Test BlockHash
	require.Panics(t, func() {
		api.BlockHash(big.NewInt(0))
	}, "BlockHash should panic as it's not available")

	// Test Block
	require.Panics(t, func() {
		api.Block()
	}, "Block should panic as it's not available")
}

func TestAPI(t *testing.T) {
	address := common.HexToAddress("0x01")
	sdb := NewMockStateDB()
	evm := NewMockEVM(sdb)
	api := New(evm, address)

	// Test Address
	require.Equal(t, address, api.Address(), "Address should return correct address")

	// Test StateDB
	require.Equal(t, sdb, api.StateDB(), "StateDB should return correct StateDB")

	// Test EVM
	require.Equal(t, evm, api.EVM(), "EVM should return correct EVM")

	// Test Persistent
	persistent := api.Persistent()
	require.NotNil(t, persistent, "Persistent should not be nil")
	require.IsType(t, &datastore{}, persistent, "Persistent should return a datastore instance")
	persistentStruct, _ := persistent.(*datastore)
	require.IsType(t, &persistentStorage{}, persistentStruct.Storage, "Persistent should return a PersistentStorage instance")

	// Test Ephemeral
	ephemeral := api.Ephemeral()
	require.NotNil(t, ephemeral, "Ephemeral should not be nil")
	require.IsType(t, &datastore{}, ephemeral, "Ephemeral should return a datastore instance")
	ephemeralStruct, _ := ephemeral.(*datastore)
	require.IsType(t, &ephemeralStorage{}, ephemeralStruct.Storage, "Ephemeral should return a EphemeralStorage instance")

	// Test BlockHash
	require.NotPanics(t, func() {
		api.BlockHash(big.NewInt(1))
	}, "BlockHash should not panic")

	// Test Block
	require.NotPanics(t, func() {
		api.Block()
	}, "Block should not panic")
}

func TestReadOnlyAPI(t *testing.T) {
	address := common.HexToAddress("0x01")
	evm := NewReadOnlyEVM(NewMockEVM(NewMockStateDB()))
	api := New(evm, address)
	apiStruct, _ := api.(*fullApi)
	require.IsType(t, &readOnlyStateDB{}, apiStruct.db, "StateDB should be readOnlyStateDB")
}

func TestCommitSafeAPI(t *testing.T) {
	address := common.HexToAddress("0x01")
	evm := NewCommitSafeEVM(NewMockEVM(NewMockStateDB()))
	api := New(evm, address)
	apiStruct, _ := api.(*fullApi)
	require.IsType(t, &commitSafeStateDB{}, apiStruct.db, "StateDB should be commitSafeStateDB")
}

func TestReadOnlyStateAPI(t *testing.T) {
	address := common.HexToAddress("0x01")
	sdb := NewReadOnlyStateDB(NewMockStateDB())
	api := NewStateAPI(sdb, address)
	apiStruct, _ := api.(*stateApi)
	require.IsType(t, &readOnlyStateDB{}, apiStruct.db, "StateDB should be readOnlyStateDB")
}

func TestCommitSafeStateAPI(t *testing.T) {
	address := common.HexToAddress("0x01")
	sdb := NewCommitSafeStateDB(NewMockStateDB())
	api := NewStateAPI(sdb, address)
	apiStruct, _ := api.(*stateApi)
	require.IsType(t, &commitSafeStateDB{}, apiStruct.db, "StateDB should be commitSafeStateDB")
}
