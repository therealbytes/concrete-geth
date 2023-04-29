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

package lib

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/concrete/api"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

func NewTestStateDB() vm.StateDB {
	db, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	return db
}

func NewTestEVMWithStateDB(db vm.StateDB) api.EVM {
	return vm.NewEVM(vm.BlockContext{}, vm.TxContext{}, db, params.TestChainConfig, vm.Config{}).NewConcreteEVM()
}

func NewTestEVM() api.EVM {
	return NewTestEVMWithStateDB(NewTestStateDB())
}

func NewTestAPI(addr common.Address) api.API {
	return api.New(NewTestEVM(), addr)
}
