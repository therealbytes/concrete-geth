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
)

type BigPreimageStore struct {
	preimageStorage    Storage
	bigPreimageStorage Storage
	radix              int
	leafSize           int
}

const (
	LEAF_FLAG = 0x00
	NODE_FLAG = 0x01
)

func NewPersistentBigPreimageStore(api API, radix, leafSize int) PreimageStore {
	statedb := api.StateDB()
	preimageStorage := &PersistentStorage{address: PreimageRegistryAddress, db: statedb}
	bigPreimageStorage := &PersistentStorage{address: BigPreimageRegistryAddress, db: statedb}
	return &BigPreimageStore{
		preimageStorage:    preimageStorage,
		bigPreimageStorage: bigPreimageStorage,
		radix:              radix,
		leafSize:           leafSize,
	}
}

func NewEphemeralBigPreimageStore(api API, radix, leafSize int) PreimageStore {
	statedb := api.StateDB()
	preimageStorage := &EphemeralStorage{address: PreimageRegistryAddress, db: statedb}
	bigPreimageStorage := &EphemeralStorage{address: BigPreimageRegistryAddress, db: statedb}
	return &BigPreimageStore{
		preimageStorage:    preimageStorage,
		bigPreimageStorage: bigPreimageStorage,
		radix:              radix,
		leafSize:           leafSize,
	}
}

func (s *BigPreimageStore) AddPreimage(preimage []byte) common.Hash {
	if len(preimage) == 0 {
		return EmptyPreimageHash
	}

	size := len(preimage)
	nHashes := (size + s.leafSize - 1) / s.leafSize
	hashes := make([][]byte, nHashes)

	// Add leaves
	for ii := 0; ii < nHashes; ii++ {
		l := ii * s.leafSize
		r := (ii + 1) * s.leafSize
		if r > len(preimage) {
			r = len(preimage)
		}
		leaf := s.newLeaf(preimage[l:r])
		hash := s.addNode(leaf).Bytes()
		hashes[ii] = hash
	}

	// Add internal nodes
	for nHashes != 1 {
		nHashes = (nHashes + s.radix - 1) / s.radix
		for ii := 0; ii < nHashes; ii++ {
			l := ii * s.radix
			r := (ii + 1) * s.radix
			if r > len(hashes) {
				r = len(hashes)
			}
			node := s.newNode(hashes[l:r])
			hash := s.addNode(node).Bytes()
			hashes[ii] = hash
		}
		hashes = hashes[:nHashes]
	}

	// Register root with size
	root := common.BytesToHash(hashes[0])
	sizeBn := big.NewInt(int64(size))
	s.bigPreimageStorage.Set(root, common.BigToHash(sizeBn))

	return root
}

func (s *BigPreimageStore) newLeaf(body []byte) []byte {
	leaf := make([]byte, 1+len(body))
	leaf[0] = LEAF_FLAG
	copy(leaf[1:], body)
	return leaf
}

func (s *BigPreimageStore) newNode(hashes [][]byte) []byte {
	node := make([]byte, 1+32*len(hashes))
	node[0] = NODE_FLAG
	for ii, hash := range hashes {
		copy(node[1+32*ii:], hash)
	}
	return node
}

func (s *BigPreimageStore) addNode(preimage []byte) common.Hash {
	return s.preimageStorage.AddPreimage(preimage)
}

func (s *BigPreimageStore) GetPreimage(hash common.Hash) []byte {
	if hash == EmptyPreimageHash {
		return []byte{}
	}
	size := s.GetPreimageSize(hash)
	if size == 0 {
		return nil
	}
	preimage := make([]byte, size)
	s.get(hash, preimage, 0)
	return preimage
}

func (s *BigPreimageStore) get(hash common.Hash, dst []byte, ptr int) int {
	preimage := s.preimageStorage.GetPreimage(hash)
	flag := preimage[0]
	body := preimage[1:]

	if flag == LEAF_FLAG {
		copy(dst[ptr:], body)
		return ptr + len(body)
	} else if flag == NODE_FLAG {
		nHashes := len(body) / 32
		for i := 0; i < nHashes; i++ {
			hash := body[i*32 : (i+1)*32]
			ptr = s.get(common.BytesToHash(hash), dst, ptr)
		}
		return ptr
	} else {
		panic("invalid flag")
	}
}

func (s *BigPreimageStore) GetPreimageSize(hash common.Hash) int {
	sizeHash := s.bigPreimageStorage.Get(hash)
	sizeBn := new(big.Int).SetBytes(sizeHash.Bytes())
	size := int(sizeBn.Int64())
	return size
}

func (s *BigPreimageStore) HasPreimage(hash common.Hash) bool {
	if hash == EmptyPreimageHash {
		return true
	}
	size := s.GetPreimageSize(hash)
	return size != 0
}

var _ PreimageStore = &BigPreimageStore{}
