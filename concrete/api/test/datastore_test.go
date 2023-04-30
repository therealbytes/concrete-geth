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
	ds := newDatastore()

	// Test NewReference
	refKey := common.HexToHash("0x01")
	ref := ds.NewReference(refKey)
	require.NotNil(t, ref)
	require.Equal(t, refKey, ref.Key())

	// Test NewMap
	mapId := common.HexToHash("0x02")
	mp := ds.NewMap(mapId)
	require.NotNil(t, mp)
	require.Equal(t, mapId, mp.Id())

	// Test NewArray
	arrId := common.HexToHash("0x03")
	arr := ds.NewArray(arrId)
	require.NotNil(t, arr)
	require.Equal(t, arrId, arr.Id())

	// Test NewSet
	setId := common.HexToHash("0x04")
	set := ds.NewSet(setId)
	require.NotNil(t, set)
	require.Equal(t, setId, set.Id())
}

func TestReference(t *testing.T) {
	ds := newDatastore()

	refKey := common.HexToHash("0x01")
	ref := ds.NewReference(refKey)

	// Test Datastore
	require.Equal(t, ds, ref.Datastore())

	// Test Key
	require.Equal(t, refKey, ref.Key())

	// Test Set and Get
	value := common.HexToHash("0x02")
	ref.Set(value)
	storedValue := ref.Get()
	require.Equal(t, value, storedValue)

	value = common.HexToHash("0x03")
	ds.Set(refKey, value)
	storedValue = ref.Get()
	require.Equal(t, value, storedValue)

	value = common.HexToHash("0x04")
	ref.Set(value)
	storedValue = ds.Get(refKey)
	require.Equal(t, value, storedValue)
}

func TestMapping(t *testing.T) {
	ds := newDatastore()

	mapId := common.HexToHash("0x01")
	mp := ds.NewMap(mapId)
	key := common.HexToHash("0x02")

	// Test Datastore
	require.Equal(t, ds, mp.Datastore())

	// Test Id
	require.Equal(t, mapId, mp.Id())

	// Test Set and Get
	value := common.HexToHash("0x03")
	mp.Set(key, value)
	storedValue := mp.Get(key)
	require.Equal(t, value, storedValue)

	// Test Get for non-existent key
	require.Equal(t, common.Hash{}, mp.Get(common.HexToHash("0x04")))

	// Test GetReference
	ref := mp.GetReference(key)
	require.NotNil(t, ref)

	value = common.HexToHash("0x05")
	ref.Set(value)
	storedValue = mp.Get(key)
	require.Equal(t, value, storedValue)

	value = common.HexToHash("0x06")
	mp.Set(key, value)
	storedValue = ref.Get()
	require.Equal(t, value, storedValue)
}

func TestArray(t *testing.T) {
	ds := newDatastore()

	arrId := common.HexToHash("0x01")
	arr := ds.NewArray(arrId)

	// Test Datastore
	require.Equal(t, ds, arr.Datastore())

	// Test Id
	require.Equal(t, arrId, arr.Id())

	// Test Push and Length

	length := arr.Length()
	require.Equal(t, 0, length)

	for i := 0; i < 10; i++ {
		arr.Push(common.BytesToHash([]byte{byte(i)}))
	}

	length = arr.Length()
	require.Equal(t, 10, length)

	// Test Pop
	poppedValue := arr.Pop()
	require.Equal(t, common.BytesToHash([]byte{byte(9)}), poppedValue)
	require.Equal(t, 9, arr.Length())

	// Test Set and Get
	value := common.HexToHash("0x02")
	arr.Set(0, value)
	storedValue := arr.Get(0)
	require.Equal(t, value, storedValue)

	// Test Get for index out of bounds
	require.Equal(t, common.Hash{}, arr.Get(100))

	// Test Set for index out of bounds
	require.Panics(t, func() { arr.Set(100, common.HexToHash("0x03")) }, "Set should panic for index out of bounds")

	// Test Swap
	i1, v1 := 0, common.HexToHash("0x04")
	i2, v2 := 1, common.HexToHash("0x05")
	arr.Set(i1, v1)
	arr.Set(i2, v2)
	arr.Swap(0, 1)
	require.Equal(t, v2, arr.Get(i1))
	require.Equal(t, v1, arr.Get(i2))

	// Test Swap for index out of bounds
	require.Panics(t, func() { arr.Swap(0, 100) }, "Swap should panic for index out of bounds")

	// Test GetReference
	ref := arr.GetReference(1)
	require.NotNil(t, ref)

	value = common.HexToHash("0x6")
	ref.Set(value)
	storedValue = arr.Get(1)
	require.Equal(t, value, storedValue)

	value = common.HexToHash("0x07")
	arr.Set(1, value)
	storedValue = ref.Get()
	require.Equal(t, value, storedValue)
}

func TestSet(t *testing.T) {
	ds := newDatastore()

	setId := common.HexToHash("0x01")
	ss := ds.NewSet(setId)

	// Test Datastore
	require.Equal(t, ds, ss.Datastore())

	// Test Id
	require.Equal(t, setId, ss.Id())

	// Test Add and Size
	size := ss.Size()
	require.Equal(t, 0, size)

	for i := 0; i < 10; i++ {
		ss.Add(common.BytesToHash([]byte{byte(i)}))
	}

	size = ss.Size()
	require.Equal(t, 10, size)

	// Test Has

	require.True(t, ss.Has(common.BytesToHash([]byte{byte(0)})))
	require.True(t, ss.Has(common.BytesToHash([]byte{byte(1)})))
	require.False(t, ss.Has(common.BytesToHash([]byte{byte(11)})))

	// Test Remove
	ss.Remove(common.BytesToHash([]byte{byte(0)}))
	require.False(t, ss.Has(common.BytesToHash([]byte{byte(0)})))
	require.True(t, ss.Has(common.BytesToHash([]byte{byte(1)})))

	// Test Values
	values := ss.Values()
	require.NotNil(t, values)
	require.Equal(t, ss.Size(), values.Length())

	for i := 0; i < values.Length(); i++ {
		require.True(t, ss.Has(values.Get(i)))
	}
}
