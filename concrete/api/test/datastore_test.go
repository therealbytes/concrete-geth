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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/concrete/api"
	"github.com/stretchr/testify/require"
)

func newDatastore() api.Datastore {
	address := common.HexToAddress("0x01")
	db := NewMockStateDB()
	return api.NewCoreDatastore(api.NewPersistentStorage(db, address))
}

func TestDatastore(t *testing.T) {
	var (
		r  = require.New(t)
		ds = newDatastore()
	)

	// Test NewReference
	refKey := common.HexToHash("0x01")
	ref := ds.NewReference(refKey)
	r.NotNil(ref)
	r.Equal(refKey, ref.Key())

	// Test NewMap
	mapId := common.HexToHash("0x02")
	mp := ds.NewMap(mapId)
	r.NotNil(mp)
	r.Equal(mapId, mp.Id())

	// Test NewArray
	arrId := common.HexToHash("0x03")
	arr := ds.NewArray(arrId)
	r.NotNil(arr)
	r.Equal(arrId, arr.Id())

	// Test NewSet
	setId := common.HexToHash("0x04")
	set := ds.NewSet(setId)
	r.NotNil(set)
	r.Equal(setId, set.Id())
}

func TestReference(t *testing.T) {
	var (
		r      = require.New(t)
		ds     = newDatastore()
		refKey = common.HexToHash("0x01")
		ref    = ds.NewReference(refKey)
	)

	// Test Datastore
	r.Equal(ds, ref.Datastore())

	// Test Key
	r.Equal(refKey, ref.Key())

	// Test Set and Get
	value := common.HexToHash("0x02")
	ref.Set(value)
	storedValue := ref.Get()
	r.Equal(value, storedValue)

	value = common.HexToHash("0x03")
	ds.Set(refKey, value)
	storedValue = ref.Get()
	r.Equal(value, storedValue)

	value = common.HexToHash("0x04")
	ref.Set(value)
	storedValue = ds.Get(refKey)
	r.Equal(value, storedValue)
}

func TestMapping(t *testing.T) {
	var (
		r     = require.New(t)
		ds    = newDatastore()
		mapId = common.HexToHash("0x01")
		mp    = ds.NewMap(mapId)
		key   = common.HexToHash("0x02")
	)

	// Test Datastore
	r.Equal(ds, mp.Datastore())

	// Test Id
	r.Equal(mapId, mp.Id())

	// Test Set and Get
	value := common.HexToHash("0x03")
	mp.Set(key, value)
	storedValue := mp.Get(key)
	r.Equal(value, storedValue)

	// Test Get for non-existent key
	r.Equal(common.Hash{}, mp.Get(common.HexToHash("0x04")))

	// Test GetReference
	ref := mp.GetReference(key)
	r.NotNil(ref)

	value = common.HexToHash("0x05")
	ref.Set(value)
	storedValue = mp.Get(key)
	r.Equal(value, storedValue)

	value = common.HexToHash("0x06")
	mp.Set(key, value)
	storedValue = ref.Get()
	r.Equal(value, storedValue)
}

func TestArray(t *testing.T) {
	var (
		r     = require.New(t)
		ds    = newDatastore()
		arrId = common.HexToHash("0x01")
		arr   = ds.NewArray(arrId)
	)

	// Test Datastore
	r.Equal(ds, arr.Datastore())

	// Test Id
	r.Equal(arrId, arr.Id())

	// Test Push and Length

	length := arr.Length()
	r.Equal(0, length)

	for i := 0; i < 10; i++ {
		arr.Push(common.BytesToHash([]byte{byte(i)}))
	}

	length = arr.Length()
	r.Equal(10, length)

	// Test Pop
	poppedValue := arr.Pop()
	r.Equal(common.BytesToHash([]byte{byte(9)}), poppedValue)
	r.Equal(9, arr.Length())

	// Test Set and Get
	value := common.HexToHash("0x02")
	arr.Set(0, value)
	storedValue := arr.Get(0)
	r.Equal(value, storedValue)

	// Test Get for index out of bounds
	r.Equal(common.Hash{}, arr.Get(100))

	// Test Set for index out of bounds
	r.Panics(func() { arr.Set(100, common.HexToHash("0x03")) }, "Set should panic for index out of bounds")

	// Test Swap
	i1, v1 := 0, common.HexToHash("0x04")
	i2, v2 := 1, common.HexToHash("0x05")
	arr.Set(i1, v1)
	arr.Set(i2, v2)
	arr.Swap(0, 1)
	r.Equal(v2, arr.Get(i1))
	r.Equal(v1, arr.Get(i2))

	// Test Swap for index out of bounds
	r.Panics(func() { arr.Swap(0, 100) }, "Swap should panic for index out of bounds")

	// Test GetReference
	ref := arr.GetReference(1)
	r.NotNil(ref)

	value = common.HexToHash("0x6")
	ref.Set(value)
	storedValue = arr.Get(1)
	r.Equal(value, storedValue)

	value = common.HexToHash("0x07")
	arr.Set(1, value)
	storedValue = ref.Get()
	r.Equal(value, storedValue)
}

func TestSet(t *testing.T) {
	var (
		r     = require.New(t)
		ds    = newDatastore()
		setId = common.HexToHash("0x01")
		ss    = ds.NewSet(setId)
	)

	// Test Datastore
	r.Equal(ds, ss.Datastore())

	// Test Id
	r.Equal(setId, ss.Id())

	// Test Add and Size
	size := ss.Size()
	r.Equal(0, size)

	for i := 0; i < 10; i++ {
		ss.Add(common.BytesToHash([]byte{byte(i)}))
	}

	size = ss.Size()
	r.Equal(10, size)

	// Test Has

	r.True(ss.Has(common.BytesToHash([]byte{byte(0)})))
	r.True(ss.Has(common.BytesToHash([]byte{byte(1)})))
	r.False(ss.Has(common.BytesToHash([]byte{byte(11)})))

	// Test Remove
	ss.Remove(common.BytesToHash([]byte{byte(0)}))
	r.False(ss.Has(common.BytesToHash([]byte{byte(0)})))
	r.True(ss.Has(common.BytesToHash([]byte{byte(1)})))

	// Test Values
	values := ss.Values()
	r.NotNil(values)
	r.Equal(ss.Size(), values.Length())

	for i := 0; i < values.Length(); i++ {
		r.True(ss.Has(values.Get(i)))
	}
}
