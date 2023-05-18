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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/concrete/crypto"
)

var (
	// EmptyPreimageHash = crypto.Keccak256Hash(nil)
	EmptyPreimageHash = common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470")
)

var (
	// HashRegistryAddress = common.BytesToAddress(crypto.Keccak256([]byte("concrete.HashRegistry.v0")))
	HashRegistryAddress = common.HexToAddress("0x5a1aca093af3a4645ae880200333893724c94e92")
)

type StateDB interface {
	SetPersistentState(addr common.Address, key common.Hash, value common.Hash)
	GetPersistentState(addr common.Address, key common.Hash) common.Hash
	SetEphemeralState(addr common.Address, key common.Hash, value common.Hash)
	GetEphemeralState(addr common.Address, key common.Hash) common.Hash
	AddPersistentPreimage(hash common.Hash, preimage []byte)
	GetPersistentPreimage(hash common.Hash) []byte
	GetPersistentPreimageSize(hash common.Hash) int
	AddEphemeralPreimage(hash common.Hash, preimage []byte)
	GetEphemeralPreimage(hash common.Hash) []byte
	GetEphemeralPreimageSize(hash common.Hash) int
}

type ReadOnlyStateDB struct {
	StateDB
}

func NewReadOnlyStateDB(db StateDB) StateDB {
	return &ReadOnlyStateDB{db}
}

func (db *ReadOnlyStateDB) SetPersistentState(addr common.Address, key common.Hash, value common.Hash) {
	panic("stateDB write protection")
}

func (db *ReadOnlyStateDB) SetEphemeralState(addr common.Address, key common.Hash, value common.Hash) {
	panic("stateDB write protection")
}

func (db *ReadOnlyStateDB) AddPersistentPreimage(hash common.Hash, preimage []byte) {
	panic("stateDB write protection")
}

func (db *ReadOnlyStateDB) AddEphemeralPreimage(hash common.Hash, preimage []byte) {
	panic("stateDB write protection")
}

var _ StateDB = &ReadOnlyStateDB{}

type CommitSafeStateDB struct {
	StateDB
}

func NewCommitSafeStateDB(db StateDB) StateDB {
	return &CommitSafeStateDB{db}
}

func (db *CommitSafeStateDB) SetPersistentState(addr common.Address, key common.Hash, value common.Hash) {
	panic("stateDB write protection")
}

var _ StateDB = &CommitSafeStateDB{}

type EVM interface {
	StateDB() StateDB
	BlockHash(block *big.Int) common.Hash
	BlockTimestamp() *big.Int
	BlockNumber() *big.Int
	BlockDifficulty() *big.Int
	BlockGasLimit() *big.Int
	BlockCoinbase() common.Address
}

type ReadOnlyEVM struct {
	EVM
}

func NewReadOnlyEVM(evm EVM) EVM {
	return &ReadOnlyEVM{evm}
}

func (evm *ReadOnlyEVM) StateDB() StateDB {
	return &ReadOnlyStateDB{evm.EVM.StateDB()}
}

var _ EVM = &ReadOnlyEVM{}

type CommitSafeEVM struct {
	EVM
}

func NewCommitSafeEVM(evm EVM) EVM {
	return &CommitSafeEVM{evm}
}

func (evm *CommitSafeEVM) StateDB() StateDB {
	return &CommitSafeStateDB{evm.EVM.StateDB()}
}

var _ EVM = &CommitSafeEVM{}

type Storage interface {
	Address() common.Address
	Set(key common.Hash, value common.Hash)
	Get(key common.Hash) common.Hash
	AddPreimage(preimage []byte) common.Hash
	HasPreimage(hash common.Hash) bool
	GetPreimage(hash common.Hash) []byte
	GetPreimageSize(hash common.Hash) int
}

func hashRegistryKey(hash common.Hash) common.Hash {
	return crypto.Keccak256Hash(hash.Bytes(), common.Big0.Bytes())
}

type PersistentStorage struct {
	address common.Address
	db      StateDB
}

func (s *PersistentStorage) Address() common.Address {
	return s.address
}

func (s *PersistentStorage) Set(key common.Hash, value common.Hash) {
	s.db.SetPersistentState(s.address, key, value)
}

func (s *PersistentStorage) Get(key common.Hash) common.Hash {
	return s.db.GetPersistentState(s.address, key)
}

func (s *PersistentStorage) AddPreimage(preimage []byte) common.Hash {
	if len(preimage) == 0 {
		return EmptyPreimageHash
	}
	hash := crypto.Keccak256Hash(preimage)
	s.db.SetPersistentState(HashRegistryAddress, hashRegistryKey(hash), common.BytesToHash(common.Big1.Bytes()))
	s.db.AddPersistentPreimage(hash, preimage)
	return hash
}

func (s *PersistentStorage) HasPreimage(hash common.Hash) bool {
	if hash == EmptyPreimageHash {
		return true
	}
	return s.db.GetPersistentState(HashRegistryAddress, hashRegistryKey(hash)) == common.BytesToHash(common.Big1.Bytes())
}

func (s *PersistentStorage) GetPreimage(hash common.Hash) []byte {
	if hash == EmptyPreimageHash {
		return []byte{}
	}
	if !s.HasPreimage(hash) {
		return nil
	}
	return s.db.GetPersistentPreimage(hash)
}

func (s *PersistentStorage) GetPreimageSize(hash common.Hash) int {
	if hash == EmptyPreimageHash {
		return 0
	}
	if !s.HasPreimage(hash) {
		return -1
	}
	return s.db.GetPersistentPreimageSize(hash)
}

var _ Storage = (*PersistentStorage)(nil)

type EphemeralStorage struct {
	address common.Address
	db      StateDB
}

func (s *EphemeralStorage) Address() common.Address {
	return s.address
}

func (s *EphemeralStorage) Set(key common.Hash, value common.Hash) {
	s.db.SetEphemeralState(s.address, key, value)
}

func (s *EphemeralStorage) Get(key common.Hash) common.Hash {
	return s.db.GetEphemeralState(s.address, key)
}

func (s *EphemeralStorage) AddPreimage(preimage []byte) common.Hash {
	if len(preimage) == 0 {
		return EmptyPreimageHash
	}
	hash := crypto.Keccak256Hash(preimage)
	s.db.SetEphemeralState(HashRegistryAddress, hashRegistryKey(hash), common.BytesToHash(common.Big1.Bytes()))
	s.db.AddEphemeralPreimage(hash, preimage)
	return hash
}

func (s *EphemeralStorage) HasPreimage(hash common.Hash) bool {
	if hash == EmptyPreimageHash {
		return true
	}
	return s.db.GetEphemeralState(HashRegistryAddress, hashRegistryKey(hash)) == common.BytesToHash(common.Big1.Bytes())
}

func (s *EphemeralStorage) GetPreimage(hash common.Hash) []byte {
	if hash == EmptyPreimageHash {
		return []byte{}
	}
	if !s.HasPreimage(hash) {
		return nil
	}
	return s.db.GetEphemeralPreimage(hash)
}

func (s *EphemeralStorage) GetPreimageSize(hash common.Hash) int {
	if hash == EmptyPreimageHash {
		return 0
	}
	if !s.HasPreimage(hash) {
		return -1
	}
	return s.db.GetEphemeralPreimageSize(hash)
}

var _ Storage = (*EphemeralStorage)(nil)

type Block interface {
	Timestamp() *big.Int
	Number() *big.Int
	Difficulty() *big.Int
	GasLimit() *big.Int
	Coinbase() common.Address
}

type FullBlock struct {
	evm EVM
}

func (b *FullBlock) Timestamp() *big.Int      { return b.evm.BlockTimestamp() }
func (b *FullBlock) Number() *big.Int         { return b.evm.BlockNumber() }
func (b *FullBlock) Difficulty() *big.Int     { return b.evm.BlockDifficulty() }
func (b *FullBlock) GasLimit() *big.Int       { return b.evm.BlockGasLimit() }
func (b *FullBlock) Coinbase() common.Address { return b.evm.BlockCoinbase() }

var _ Block = (*FullBlock)(nil)

type LiteAPI interface {
	Address() common.Address
	Persistent() Datastore
	BlockHash(block *big.Int) common.Hash
	Block() Block
}

type API interface {
	Address() common.Address
	EVM() EVM
	StateDB() StateDB
	Persistent() Datastore
	Ephemeral() Datastore
	BlockHash(block *big.Int) common.Hash
	Block() Block
}

type StateAPI struct {
	address    common.Address
	db         StateDB
	persistent Datastore
	ephemeral  Datastore
}

func NewStateAPI(db StateDB, address common.Address) API {
	return &StateAPI{
		address: address,
		db:      db,
	}
}

func (s *StateAPI) Address() common.Address {
	return s.address
}

func (s *StateAPI) EVM() EVM {
	return nil
}

func (s *StateAPI) StateDB() StateDB {
	return s.db
}

func (s *StateAPI) Persistent() Datastore {
	if s.persistent == nil {
		s.persistent = &CoreDatastore{&PersistentStorage{
			address: s.address,
			db:      s.db,
		}}
	}
	return s.persistent
}

func (s *StateAPI) Ephemeral() Datastore {
	if s.ephemeral == nil {
		s.ephemeral = &CoreDatastore{&EphemeralStorage{
			address: s.address,
			db:      s.db,
		}}
	}
	return s.ephemeral
}

func (s *StateAPI) BlockHash(block *big.Int) common.Hash {
	panic("API method not available")
}

func (s *StateAPI) Block() Block {
	panic("API method not available")
}

var _ API = (*StateAPI)(nil)

type FullAPI struct {
	StateAPI
	evm EVM
}

func New(evm EVM, address common.Address) API {
	return NewAPI(evm, address)
}

func NewAPI(evm EVM, address common.Address) API {
	return &FullAPI{
		StateAPI: StateAPI{address: address, db: evm.StateDB()},
		evm:      evm,
	}
}

func (a *FullAPI) EVM() EVM {
	return a.evm
}

func (a *FullAPI) BlockHash(block *big.Int) common.Hash {
	return a.evm.BlockHash(block)
}

func (a *FullAPI) Block() Block {
	return &FullBlock{evm: a.evm}
}

var _ API = (*FullAPI)(nil)

type Precompile interface {
	MutatesStorage(input []byte) bool
	RequiredGas(input []byte) uint64
	Finalise(api API) error
	Commit(api API) error
	Run(api API, input []byte) ([]byte, error)
}
