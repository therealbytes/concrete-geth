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

func TestPersistentStorage(t *testing.T) {
	sdb := NewMockStateDB()
	address := common.HexToAddress("0x01")
	storage := &persistentStorage{address, sdb}
	TestStorage(t, storage)
	FuzzStorage(t, storage)
}

func TestEphemeralStorage(t *testing.T) {
	sdb := NewMockStateDB()
	address := common.HexToAddress("0x01")
	storage := &ephemeralStorage{address, sdb}
	TestStorage(t, storage)
	FuzzStorage(t, storage)
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
