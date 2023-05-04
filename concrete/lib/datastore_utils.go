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
	cc_api "github.com/ethereum/go-ethereum/concrete/api"
)

type Counter struct {
	cc_api.Reference
}

func NewCounter(ref cc_api.Reference) *Counter {
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

func (c *Counter) Inc() {
	c.Add(common.Big1)
}

type nestedMap struct {
	cc_api.Mapping
	depth int
}

func NewNestedMap(mapping cc_api.Mapping, depth int) cc_api.Mapping {
	if depth < 2 {
		panic("depth must be at least 2")
	}
	return &nestedMap{mapping, depth}
}

func (m *nestedMap) GetNested(keys ...common.Hash) common.Hash {
	if len(keys) != m.depth {
		panic("wrong number of keys")
	}
	Array := m.GetNestedMap(keys[0 : m.depth-1]...)
	return Array.Get(keys[m.depth-1])
}

func (m *nestedMap) GetNestedMap(keys ...common.Hash) cc_api.Mapping {
	if len(keys) != m.depth-1 {
		panic("wrong number of keys")
	}
	next := m.Mapping
	for ii := 0; ii < m.depth-1; ii++ {
		next = next.GetMap(keys[ii])
	}
	return next
}

type nestedArray struct {
	cc_api.Array
	depth int
}

func NewNestedArray(array cc_api.Array, depth int) cc_api.Array {
	if depth < 2 {
		panic("depth must be at least 2")
	}
	return &nestedArray{array, depth}
}

func (m *nestedArray) GetNested(indexes ...int) common.Hash {
	if len(indexes) != m.depth {
		panic("wrong number of indexes")
	}
	Array := m.GetNestedArray(indexes[0 : m.depth-1]...)
	return Array.Get(indexes[m.depth-1])
}

func (m *nestedArray) GetNestedArray(indexes ...int) cc_api.Array {
	if len(indexes) != m.depth-1 {
		panic("wrong number of indexes")
	}
	next := m.Array
	for ii := 0; ii < m.depth-1; ii++ {
		next = next.GetArray(indexes[ii])
	}
	return next
}
