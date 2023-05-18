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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	cc_api "github.com/ethereum/go-ethereum/concrete/api"
	"github.com/stretchr/testify/require"
)

var statedbs = []struct {
	name        string
	constructor func() cc_api.StateDB
	readOnly    bool
	commitSafe  bool
}{
	{
		name: "StateDB",
		constructor: func() cc_api.StateDB {
			return NewMockStateDB()
		},
		readOnly:   false,
		commitSafe: false,
	},
	{
		name: "ReadOnlyStateDB",
		constructor: func() cc_api.StateDB {
			return cc_api.NewReadOnlyStateDB(NewMockStateDB())
		},
		readOnly:   true,
		commitSafe: true,
	},
	{
		name: "CommitSafeStateDB",
		constructor: func() cc_api.StateDB {
			return cc_api.NewCommitSafeStateDB(NewMockStateDB())
		},
		readOnly:   false,
		commitSafe: true,
	},
}

var statedbMethods = []struct {
	name       string
	call       func(statedb cc_api.StateDB)
	readOnly   bool
	commitSafe bool
}{
	{
		name: "SetPersistentState",
		call: func(statedb cc_api.StateDB) {
			statedb.SetPersistentState(common.Address{}, common.Hash{}, common.Hash{})
		},
		readOnly:   false,
		commitSafe: false,
	},
	{
		name: "SetEphemeralState",
		call: func(statedb cc_api.StateDB) {
			statedb.SetEphemeralState(common.Address{}, common.Hash{}, common.Hash{})
		},
		readOnly:   false,
		commitSafe: true,
	},
	{
		name: "AddPersistentPreimage",
		call: func(statedb cc_api.StateDB) {
			statedb.AddPersistentPreimage(common.Hash{}, []byte{})
		},
		readOnly:   false,
		commitSafe: true,
	},
	{
		name: "AddEphemeralPreimage",
		call: func(statedb cc_api.StateDB) {
			statedb.AddEphemeralPreimage(common.Hash{}, []byte{})
		},
		readOnly:   false,
		commitSafe: true,
	},
	{
		name: "GetPersistentState",
		call: func(statedb cc_api.StateDB) {
			statedb.GetPersistentState(common.Address{}, common.Hash{})
		},
		readOnly:   true,
		commitSafe: true,
	},
	{
		name: "GetEphemeralState",
		call: func(statedb cc_api.StateDB) {
			statedb.GetEphemeralState(common.Address{}, common.Hash{})
		},
		readOnly:   true,
		commitSafe: true,
	},
	{
		name: "GetPersistentPreimage",
		call: func(statedb cc_api.StateDB) {
			statedb.GetPersistentPreimage(common.Hash{})
		},
		readOnly:   true,
		commitSafe: true,
	},
	{
		name: "GetPersistentPreimageSize",
		call: func(statedb cc_api.StateDB) {
			statedb.GetPersistentPreimageSize(common.Hash{})
		},
		readOnly:   true,
		commitSafe: true,
	},
	{
		name: "GetEphemeralPreimage",
		call: func(statedb cc_api.StateDB) {
			statedb.GetEphemeralPreimage(common.Hash{})
		},
		readOnly:   true,
		commitSafe: true,
	},
	{
		name: "GetEphemeralPreimageSize",
		call: func(statedb cc_api.StateDB) {
			statedb.GetEphemeralPreimageSize(common.Hash{})
		},
		readOnly:   true,
		commitSafe: true,
	},
}

func TestStateDB(t *testing.T) {
	var (
		r = require.New(t)
	)
	for _, specs := range statedbs {
		t.Run(specs.name, func(t *testing.T) {
			statedb := specs.constructor()
			for _, method := range statedbMethods {
				if (specs.readOnly && !method.readOnly) || (specs.commitSafe && !method.commitSafe) {
					r.Panics(func() { method.call(statedb) }, method.name+" should panic")
				} else {
					r.NotPanics(func() { method.call(statedb) }, method.name+" should not panic")
				}
			}
		})
	}
}

var evms = []struct {
	name        string
	constructor func() cc_api.EVM
	statedbType interface{}
}{
	{
		name: "EVM",
		constructor: func() cc_api.EVM {
			return NewMockEVM(NewMockStateDB())
		},
		statedbType: &MockStateDB{},
	},
	{
		name: "ReadOnlyEVM",
		constructor: func() cc_api.EVM {
			return cc_api.NewReadOnlyEVM(NewMockEVM(NewMockStateDB()))
		},
		statedbType: &cc_api.ReadOnlyStateDB{},
	},
	{
		name: "CommitSafeEVM",
		constructor: func() cc_api.EVM {
			return cc_api.NewCommitSafeEVM(NewMockEVM(NewMockStateDB()))
		},
		statedbType: &cc_api.CommitSafeStateDB{},
	},
}

func TestEVM(t *testing.T) {
	var (
		r = require.New(t)
	)
	for _, specs := range evms {
		t.Run(specs.name, func(t *testing.T) {
			evm := specs.constructor()
			r.IsType(specs.statedbType, evm.StateDB(), "StateDB should return "+specs.name)
		})
	}
}

var storages = []struct {
	name        string
	constructor func() cc_api.Storage
}{
	{
		name: "PersistentStorage",
		constructor: func() cc_api.Storage {
			return cc_api.NewPersistentStorage(NewMockStateDB(), common.Address{})
		},
	},
	{
		name: "EphemeralStorage",
		constructor: func() cc_api.Storage {
			return cc_api.NewEphemeralStorage(NewMockStateDB(), common.Address{})
		},
	},
}

func TestAPIStorage(t *testing.T) {
	for _, specs := range storages {
		t.Run(specs.name, func(t *testing.T) {
			storage := specs.constructor()
			TestStorage(t, storage)
			FuzzStorage(t, storage)
		})
	}
}

var apis = []struct {
	name        string
	constructor func() cc_api.API
	readOnly    bool
	stateOnly   bool
}{
	{
		name: "API",
		constructor: func() cc_api.API {
			statedb := NewMockStateDB()
			evm := NewMockEVM(statedb)
			return cc_api.New(evm, common.Address{})
		},
		readOnly:  false,
		stateOnly: false,
	},
	{
		name: "StateAPI",
		constructor: func() cc_api.API {
			statedb := NewMockStateDB()
			return cc_api.NewStateAPI(statedb, common.Address{})
		},
		readOnly:  false,
		stateOnly: true,
	},
	{
		name: "ReadOnlyAPI",
		constructor: func() cc_api.API {
			statedb := NewMockStateDB()
			evm := cc_api.NewReadOnlyEVM(NewMockEVM(statedb))
			return cc_api.New(evm, common.Address{})
		},
		readOnly:  true,
		stateOnly: false,
	},
	{
		name: "ReadOnlyStateAPI",
		constructor: func() cc_api.API {
			statedb := cc_api.NewReadOnlyStateDB(NewMockStateDB())
			return cc_api.NewStateAPI(statedb, common.Address{})
		},
		readOnly:  true,
		stateOnly: true,
	},
}

func TestStateAPI(t *testing.T) {
	// TODO: test read-only API
	// TODO: test lite API
	// TODO: test address method
	var (
		r = require.New(t)
	)
	for _, specs := range apis {
		t.Run(specs.name, func(t *testing.T) {
			api := specs.constructor()
			r.NotNil(api.StateDB(), "StateDB should not be nil")
			r.NotNil(api.Ephemeral(), "Ephemeral should not be nil")
			r.NotNil(api.Persistent(), "Persistent should not be nil")
			if specs.stateOnly {
				r.Nil(api.EVM(), "EVM should be nil")
				r.Panics(func() { api.BlockHash(big.NewInt(0)) }, "BlockHash should panic")
				r.Panics(func() { api.Block() }, "Block should panic")
			} else {
				r.NotNil(api.EVM(), "EVM should not be nil")
				r.NotPanics(func() { api.BlockHash(big.NewInt(0)) }, "BlockHash should not panic")
				r.NotPanics(func() { api.Block() }, "Block should not panic")
			}
		})
	}
}
