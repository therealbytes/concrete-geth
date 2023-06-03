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

package precompiles

import (
	cc_api "github.com/ethereum/go-ethereum/concrete/api"
	"github.com/ethereum/go-ethereum/concrete/crypto"
	"github.com/ethereum/go-ethereum/concrete/lib"
	tinygo_lib "github.com/ethereum/go-ethereum/tinygo/lib"
)

// A precompile used for testing.

// We use tinygo_lib.Keccak256Hash instead of crypto.Keccak256Hash because the latter
// may not compile in tinygo as may depend on a host function.
var (
	runCounterKey = tinygo_lib.Keccak256Hash([]byte("typical.counter.0"))
	hashSetKey    = tinygo_lib.Keccak256Hash([]byte("typical.set.0"))
)

type TypicalPrecompile struct{}

func (pc *TypicalPrecompile) MutatesStorage(input []byte) bool {
	return true
}

func (pc *TypicalPrecompile) RequiredGas(input []byte) uint64 {
	return 10
}

func (pc *TypicalPrecompile) Finalise(api cc_api.API) error {
	return nil
}

func (pc *TypicalPrecompile) Commit(api cc_api.API) error {
	eph := api.Ephemeral()
	newHashesSet := eph.NewSet(hashSetKey)
	arr := newHashesSet.Values()
	for ii := 0; ii < arr.Length(); ii++ {
		hash := arr.Get(ii)
		preimage := eph.GetPreimage(hash)
		api.StateDB().AddPersistentPreimage(hash, preimage)
	}
	return nil
}

func (pc *TypicalPrecompile) Run(api cc_api.API, input []byte) ([]byte, error) {
	per := api.Persistent()
	eph := api.Ephemeral()

	counter := lib.NewCounter(per.NewReference(runCounterKey))
	counter.Inc()

	hashSet := per.NewSet(hashSetKey)
	newHashesSet := eph.NewSet(hashSetKey)
	hash := crypto.Keccak256Hash(input)
	hashSet.Add(hash)
	newHashesSet.Add(hash)
	eph.AddPreimage(input)

	return nil, nil
}
