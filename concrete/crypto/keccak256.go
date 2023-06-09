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

package crypto

import (
	"hash"

	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/crypto/sha3"
)

type keccakState interface {
	hash.Hash
	Read([]byte) (int, error)
}

func newKeccakState() keccakState {
	return sha3.NewLegacyKeccak256().(keccakState)
}

func ReimplementedKeccak256(data ...[]byte) []byte {
	b := make([]byte, 32)
	d := newKeccakState()
	for _, b := range data {
		d.Write(b)
	}
	d.Read(b)
	return b
}

func ReimplementedKeccak256Hash(data ...[]byte) (h common.Hash) {
	d := newKeccakState()
	for _, b := range data {
		d.Write(b)
	}
	d.Read(h[:])
	return h
}
