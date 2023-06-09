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

type Datastore interface {
	Storage
	NewReference(key common.Hash) Reference
	NewMap(id common.Hash) Mapping
	NewArray(id common.Hash) Array
	NewSet(id common.Hash) Set
}

type CoreDatastore struct {
	Storage
}

func (d *CoreDatastore) NewReference(key common.Hash) Reference {
	return &reference{
		ds:  d,
		key: key,
	}
}

func (d *CoreDatastore) NewMap(id common.Hash) Mapping {
	return &mapping{
		ds: d,
		id: id,
	}
}

func (d *CoreDatastore) NewArray(id common.Hash) Array {
	return &array{
		ds: d,
		id: id,
	}
}

func (d *CoreDatastore) NewSet(id common.Hash) Set {
	return &set{
		ds: d,
		id: id,
	}
}

var _ Datastore = (*CoreDatastore)(nil)

// Reference

type Reference interface {
	Datastore() Datastore
	Key() common.Hash
	Get() common.Hash
	Set(value common.Hash)
}

type reference struct {
	ds  Datastore
	key common.Hash
}

func (r *reference) Datastore() Datastore {
	return r.ds
}

func (r *reference) Key() common.Hash {
	return r.key
}

func (r *reference) Get() common.Hash {
	return r.ds.Get(r.key)
}

func (r *reference) Set(value common.Hash) {
	r.ds.Set(r.key, value)
}

var _ Reference = (*reference)(nil)

// Map

type Mapping interface {
	Datastore() Datastore
	Id() common.Hash
	Get(key common.Hash) common.Hash
	Set(key common.Hash, value common.Hash)
	GetReference(key common.Hash) Reference
	GetMap(key common.Hash) Mapping
	GetArray(key common.Hash) Array
}

type mapping struct {
	ds Datastore
	id common.Hash
}

func (m *mapping) key(key common.Hash) common.Hash {
	return crypto.Keccak256Hash(key.Bytes(), m.id.Bytes())
}

func (m *mapping) Datastore() Datastore {
	return m.ds
}

func (m *mapping) Id() common.Hash {
	return m.id
}

func (m *mapping) Get(key common.Hash) common.Hash {
	return m.ds.Get(m.key(key))
}

func (m *mapping) Set(key common.Hash, value common.Hash) {
	m.ds.Set(m.key(key), value)
}

func (m *mapping) GetReference(key common.Hash) Reference {
	return &reference{
		key: m.key(key),
		ds:  m.ds,
	}
}

func (m *mapping) GetMap(key common.Hash) Mapping {
	return &mapping{
		id: m.key(key),
		ds: m.ds,
	}
}

func (m *mapping) GetArray(key common.Hash) Array {
	return &array{
		id: m.key(key),
		ds: m.ds,
	}
}

var _ Mapping = (*mapping)(nil)

// Array

type Array interface {
	Datastore() Datastore
	Id() common.Hash
	Length() int
	Get(index int) common.Hash
	Set(index int, value common.Hash)
	Push(value common.Hash)
	Pop() common.Hash
	Swap(i, j int)
	GetReference(index int) Reference
	GetMap(index int) Mapping
	GetArray(index int) Array
}

type array struct {
	ds     Datastore
	id     common.Hash
	idHash common.Hash
}

func (a *array) getIdHash() common.Hash {
	if a.idHash == (common.Hash{}) {
		a.idHash = crypto.Keccak256Hash(a.id.Bytes())
	}
	return a.idHash
}

func (a *array) key(index int) common.Hash {
	a.getIdHash()
	slot := new(big.Int).Add(a.idHash.Big(), big.NewInt(int64(index)))
	return common.BigToHash(slot)
}

func (a *array) setLength(length int) {
	a.ds.Set(a.id, common.BigToHash(big.NewInt(int64(length))))
}

func (a *array) getLength() int {
	return int(a.ds.Get(a.id).Big().Int64())
}

func (a *array) Datastore() Datastore {
	return a.ds
}

func (a *array) Id() common.Hash {
	return a.id
}

func (a *array) Length() int {
	return a.getLength()
}

func (a *array) Get(index int) common.Hash {
	if index >= a.Length() {
		return common.Hash{}
	}
	return a.ds.Get(a.key(index))
}

func (a *array) Set(index int, value common.Hash) {
	if index >= a.Length() {
		panic("index out of bounds")
	}
	a.ds.Set(a.key(index), value)
}

func (a *array) Push(value common.Hash) {
	length := a.Length()
	a.setLength(length + 1)
	a.Set(length, value)
}

func (a *array) Pop() common.Hash {
	length := a.Length()
	if length == 0 {
		return common.Hash{}
	}
	value := a.Get(length - 1)
	a.setLength(length - 1)
	return value
}

func (a *array) Swap(i, j int) {
	if i >= a.Length() || j >= a.Length() {
		panic("index out of bounds")
	}
	iVal := a.Get(i)
	a.Set(i, a.Get(j))
	a.Set(j, iVal)
}

func (a *array) GetReference(index int) Reference {
	return &reference{
		ds:  a.ds,
		key: a.key(index),
	}
}

func (a *array) GetMap(index int) Mapping {
	return &mapping{
		ds: a.ds,
		id: a.key(index),
	}
}

func (a *array) GetArray(index int) Array {
	return &array{
		ds: a.ds,
		id: a.key(index),
	}
}

var _ Array = (*array)(nil)

type Set interface {
	Datastore() Datastore
	Id() common.Hash
	Has(value common.Hash) bool
	Add(value common.Hash)
	Remove(value common.Hash)
	Size() int
	Values() Array
}

type set struct {
	ds      Datastore
	id      common.Hash
	idHash  common.Hash
	array   Array
	mapping Mapping
}

func (s *set) getIdHash() common.Hash {
	if s.idHash == (common.Hash{}) {
		s.idHash = crypto.Keccak256Hash(s.id.Bytes())
	}
	return s.idHash
}

func (s *set) valueArray() Array {
	if s.array == nil {
		s.getIdHash()
		s.array = s.ds.NewArray(s.idHash)
	}
	return s.array
}

func (s *set) indexMap() Mapping {
	if s.mapping == nil {
		s.getIdHash()
		keyBN := new(big.Int).Add(s.idHash.Big(), common.Big1)
		s.mapping = s.ds.NewMap(common.BigToHash(keyBN))
	}
	return s.mapping
}

func (s *set) Datastore() Datastore {
	return s.ds
}

func (s *set) Id() common.Hash {
	return s.id
}

func (s *set) Has(value common.Hash) bool {
	if s.Size() == 0 {
		return false
	}
	index := s.indexMap().Get(value)
	if index == (common.Hash{}) {
		return value == s.valueArray().Get(0)
	}
	return index != common.Hash{}
}

func (s *set) Add(value common.Hash) {
	if s.Has(value) {
		return
	}
	index := s.valueArray().Length()
	s.indexMap().Set(value, common.BigToHash(big.NewInt(int64(index))))
	s.valueArray().Push(value)
}

func (s *set) Remove(value common.Hash) {
	if !s.Has(value) {
		return
	}
	index := int(s.indexMap().Get(value).Big().Int64())
	s.valueArray().Swap(index, s.valueArray().Length()-1)
	s.valueArray().Pop()
	s.indexMap().Set(value, common.Hash{})
}

func (s *set) Size() int {
	return s.valueArray().Length()
}

func (s *set) Values() Array {
	return s.valueArray()
}

var _ Set = (*set)(nil)

// Constructors for for testing.
// Testing is done in a separate package as it may introduce dependencies
// incompatible with tinygo.

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
