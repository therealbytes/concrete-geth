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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/concrete/api"
	"github.com/ethereum/go-ethereum/concrete/crypto"
)

func NewKey(name string) common.Hash {
	// Use /concrete/crypto instead of /crypto because the latter won't compile
	// in tinygo as it has unsupported dependencies.
	return crypto.Keccak256Hash([]byte(name))
}

type Counter struct {
	api.Reference
}

func NewCounter(ref api.Reference) *Counter {
	return &Counter{ref}
}

func (c *Counter) Get() *big.Int {
	return c.Reference.Get().Big()
}

func (c *Counter) Set(value *big.Int) {
	c.Reference.Set(common.BigToHash(value))
}

func (c *Counter) Add(delta *big.Int) {
	value := c.Get()
	value.Add(value, delta)
	c.Set(value)
}

func (c *Counter) Sub(delta *big.Int) {
	value := c.Get()
	value.Sub(value, delta)
	c.Set(value)
}

func (c *Counter) Inc() {
	c.Add(common.Big1)
}

func (c *Counter) Dec() {
	c.Sub(common.Big1)
}

type NestedMap struct {
	api.Mapping
}

func NewNestedMap(mapping api.Mapping) *NestedMap {
	return &NestedMap{mapping}
}

func (m *NestedMap) GetNested(keys ...common.Hash) common.Hash {
	lastIdx := len(keys) - 1
	mapping := m.GetNestedMap(keys[:lastIdx]...)
	return mapping.Get(keys[lastIdx])
}

func (m *NestedMap) GetNestedMap(keys ...common.Hash) api.Mapping {
	next := m.Mapping
	for _, key := range keys {
		next = next.GetMap(key)
	}
	return next
}
