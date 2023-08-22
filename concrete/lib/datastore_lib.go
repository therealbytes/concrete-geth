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
)

type Counter struct {
	StorageSlot
}

func NewCounter(ref StorageSlot) *Counter {
	return &Counter{ref}
}

func (c *Counter) Get() *big.Int {
	return c.StorageSlot.GetBig()
}

func (c *Counter) Set(value *big.Int) {
	c.StorageSlot.SetBig(value)
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