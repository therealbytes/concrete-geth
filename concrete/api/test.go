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

import "github.com/ethereum/go-ethereum/common"

func NewPersistentStorage(db StateDB, address common.Address) *PersistentStorage {
	return &PersistentStorage{
		db:      db,
		address: address,
	}
}

func NewEphemeralStorage(db StateDB, address common.Address) *EphemeralStorage {
	return &EphemeralStorage{
		db:      db,
		address: address,
	}
}

func NewFullBlock(evm EVM) *FullBlock {
	return &FullBlock{evm}
}

func NewCoreDatastore(storage Storage) *CoreDatastore {
	return &CoreDatastore{storage}
}
