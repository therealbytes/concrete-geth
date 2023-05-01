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
	"github.com/ethereum/go-ethereum/crypto"
	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/require"
)

func TestStorage(t *testing.T, storage api.Storage) {
	var (
		key                     = common.HexToHash("0x01")
		value                   = common.HexToHash("0x02")
		preimage                = []byte("test preimage")
		preimageHash            = crypto.Keccak256Hash(preimage)
		nonExistentKey          = common.HexToHash("0x03")
		nonExistentPreimageHash = common.HexToHash("0x04")
		nilPreimageHash         = crypto.Keccak256Hash(nil)
	)

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

func FuzzStorage(t *testing.T, storage api.Storage) {
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
