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
	"hash"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/crypto/sha3"
)

var (
	EmptyPreimageHash = Keccak256Hash(nil) // c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470
)

var (
	HashRegistryAddress = common.BytesToAddress(Keccak256([]byte("concrete.HashRegistry.v0")))
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

type readOnlyStateDB struct {
	StateDB
}

func NewReadOnlyStateDB(db StateDB) StateDB {
	return &readOnlyStateDB{db}
}

func (db *readOnlyStateDB) SetPersistentState(addr common.Address, key common.Hash, value common.Hash) {
	panic("stateDB write protection")
}

func (db *readOnlyStateDB) SetEphemeralState(addr common.Address, key common.Hash, value common.Hash) {
	panic("stateDB write protection")
}

func (db *readOnlyStateDB) AddPersistentPreimage(hash common.Hash, preimage []byte) {
	panic("stateDB write protection")
}

func (db *readOnlyStateDB) AddEphemeralPreimage(hash common.Hash, preimage []byte) {
	panic("stateDB write protection")
}

var _ StateDB = &readOnlyStateDB{}

type commitSafeStateDB struct {
	StateDB
}

func NewCommitSafeStateDB(db StateDB) StateDB {
	return &commitSafeStateDB{db}
}

func (db *commitSafeStateDB) SetPersistentState(addr common.Address, key common.Hash, value common.Hash) {
	panic("stateDB write protection")
}

var _ StateDB = &commitSafeStateDB{}

type EVM interface {
	StateDB() StateDB
	BlockHash(block *big.Int) common.Hash
	BlockTimestamp() *big.Int
	BlockNumber() *big.Int
	BlockDifficulty() *big.Int
	BlockGasLimit() *big.Int
	BlockCoinbase() common.Address
}

type readOnlyEVM struct {
	EVM
}

func NewReadOnlyEVM(evm EVM) EVM {
	return &readOnlyEVM{evm}
}

func (evm *readOnlyEVM) StateDB() StateDB {
	return &readOnlyStateDB{evm.EVM.StateDB()}
}

var _ EVM = &readOnlyEVM{}

type commitSafeEVM struct {
	EVM
}

func NewCommitSafeEVM(evm EVM) EVM {
	return &commitSafeEVM{evm}
}

func (evm *commitSafeEVM) StateDB() StateDB {
	return &commitSafeStateDB{evm.EVM.StateDB()}
}

var _ EVM = &commitSafeEVM{}

type Storage interface {
	Set(key common.Hash, value common.Hash)
	Get(key common.Hash) common.Hash
	AddPreimage(preimage []byte)
	HasPreimage(hash common.Hash) bool
	GetPreimage(hash common.Hash) []byte
	GetPreimageSize(hash common.Hash) int
}

type persistentStorage struct {
	address common.Address
	db      StateDB
}

func (s *persistentStorage) Set(key common.Hash, value common.Hash) {
	s.db.SetPersistentState(s.address, key, value)
}

func (s *persistentStorage) Get(key common.Hash) common.Hash {
	return s.db.GetPersistentState(s.address, key)
}

func (s *persistentStorage) AddPreimage(preimage []byte) {
	if len(preimage) == 0 {
		return
	}
	hash := Keccak256Hash(preimage)
	s.db.SetPersistentState(HashRegistryAddress, hash, common.BytesToHash(common.Big1.Bytes()))
	s.db.AddPersistentPreimage(hash, preimage)
}

func (s *persistentStorage) HasPreimage(hash common.Hash) bool {
	if hash == EmptyPreimageHash {
		return true
	}
	return s.db.GetPersistentState(HashRegistryAddress, hash) == common.BytesToHash(common.Big1.Bytes())
}

func (s *persistentStorage) GetPreimage(hash common.Hash) []byte {
	if hash == EmptyPreimageHash {
		return []byte{}
	}
	if !s.HasPreimage(hash) {
		return nil
	}
	return s.db.GetPersistentPreimage(hash)
}

func (s *persistentStorage) GetPreimageSize(hash common.Hash) int {
	if hash == EmptyPreimageHash {
		return 0
	}
	if !s.HasPreimage(hash) {
		return -1
	}
	return s.db.GetPersistentPreimageSize(hash)
}

var _ Storage = (*persistentStorage)(nil)

type ephemeralStorage struct {
	address common.Address
	db      StateDB
}

func (s *ephemeralStorage) Set(key common.Hash, value common.Hash) {
	s.db.SetEphemeralState(s.address, key, value)
}

func (s *ephemeralStorage) Get(key common.Hash) common.Hash {
	return s.db.GetEphemeralState(s.address, key)
}

func (s *ephemeralStorage) AddPreimage(preimage []byte) {
	if len(preimage) == 0 {
		return
	}
	hash := Keccak256Hash(preimage)
	s.db.SetEphemeralState(HashRegistryAddress, hash, common.BytesToHash(common.Big1.Bytes()))
	s.db.AddEphemeralPreimage(hash, preimage)
}

func (s *ephemeralStorage) HasPreimage(hash common.Hash) bool {
	if hash == EmptyPreimageHash {
		return true
	}
	return s.db.GetEphemeralState(HashRegistryAddress, hash) == common.BytesToHash(common.Big1.Bytes())
}

func (s *ephemeralStorage) GetPreimage(hash common.Hash) []byte {
	if hash == EmptyPreimageHash {
		return []byte{}
	}
	if !s.HasPreimage(hash) {
		return nil
	}
	return s.db.GetEphemeralPreimage(hash)
}

func (s *ephemeralStorage) GetPreimageSize(hash common.Hash) int {
	if hash == EmptyPreimageHash {
		return 0
	}
	if !s.HasPreimage(hash) {
		return -1
	}
	return s.db.GetEphemeralPreimageSize(hash)
}

var _ Storage = (*ephemeralStorage)(nil)

type Block interface {
	Timestamp() *big.Int
	Number() *big.Int
	Difficulty() *big.Int
	GasLimit() *big.Int
	Coinbase() common.Address
}

type block struct {
	evm EVM
}

func (b *block) Timestamp() *big.Int      { return b.evm.BlockTimestamp() }
func (b *block) Number() *big.Int         { return b.evm.BlockNumber() }
func (b *block) Difficulty() *big.Int     { return b.evm.BlockDifficulty() }
func (b *block) GasLimit() *big.Int       { return b.evm.BlockGasLimit() }
func (b *block) Coinbase() common.Address { return b.evm.BlockCoinbase() }

var _ Block = (*block)(nil)

type API interface {
	Address() common.Address
	EVM() EVM
	StateDB() StateDB
	Persistent() Datastore
	Ephemeral() Datastore
	BlockHash(block *big.Int) common.Hash
	Block() Block
}

type stateApi struct {
	address    common.Address
	db         StateDB
	persistent Datastore
	ephemeral  Datastore
}

func NewStateAPI(db StateDB, address common.Address) API {
	return &stateApi{
		address: address,
		db:      db,
	}
}

func (s *stateApi) Address() common.Address {
	return s.address
}

func (s *stateApi) EVM() EVM {
	return nil
}

func (s *stateApi) StateDB() StateDB {
	return s.db
}

func (s *stateApi) Persistent() Datastore {
	if s.persistent == nil {
		s.persistent = &datastore{&persistentStorage{
			address: s.address,
			db:      s.db,
		}}
	}
	return s.persistent
}

func (s *stateApi) Ephemeral() Datastore {
	if s.ephemeral == nil {
		s.ephemeral = &datastore{&ephemeralStorage{
			address: s.address,
			db:      s.db,
		}}
	}
	return s.ephemeral
}

func (s *stateApi) BlockHash(block *big.Int) common.Hash {
	panic("API method not available")
}

func (s *stateApi) Block() Block {
	panic("API method not available")
}

var _ API = (*stateApi)(nil)

type fullApi struct {
	stateApi
	evm EVM
}

func New(evm EVM, address common.Address) API {
	return &fullApi{
		stateApi: stateApi{address: address, db: evm.StateDB()},
		evm:      evm,
	}
}

func (a *fullApi) EVM() EVM {
	return a.evm
}

func (a *fullApi) BlockHash(block *big.Int) common.Hash {
	return a.evm.BlockHash(block)
}

func (a *fullApi) Block() Block {
	return &block{evm: a.evm}
}

var _ API = (*fullApi)(nil)

type Precompile interface {
	MutatesStorage(input []byte) bool
	RequiredGas(input []byte) uint64
	Finalise(api API) error
	Commit(api API) error
	Run(api API, input []byte) ([]byte, error)
}

// Re-implementation of Keccak256Hash so we it can be used from tinyGo

type KeccakState interface {
	hash.Hash
	Read([]byte) (int, error)
}

func NewKeccakState() KeccakState {
	return sha3.NewLegacyKeccak256().(KeccakState)
}

func Keccak256(data ...[]byte) []byte {
	b := make([]byte, 32)
	d := NewKeccakState()
	for _, b := range data {
		d.Write(b)
	}
	d.Read(b)
	return b
}

func Keccak256Hash(data ...[]byte) (h common.Hash) {
	d := NewKeccakState()
	for _, b := range data {
		d.Write(b)
	}
	d.Read(h[:])
	return h
}
