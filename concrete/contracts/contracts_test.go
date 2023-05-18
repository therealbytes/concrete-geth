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

package contracts

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	cc_api "github.com/ethereum/go-ethereum/concrete/api"
	cc_api_test "github.com/ethereum/go-ethereum/concrete/api/test"
	"github.com/ethereum/go-ethereum/concrete/lib"
	"github.com/stretchr/testify/require"
)

func TestAddPrecompile(t *testing.T) {
	var (
		r   = require.New(t)
		n   = 10
		pcs = ActivePrecompiles()
	)

	r.Empty(pcs, "Expected no precompiles")

	for i := byte(0); i < byte(n); i++ {
		addr := common.BytesToAddress([]byte{i})
		err := AddPrecompile(addr, &lib.BlankPrecompile{})
		r.NoError(err, "AddPrecompile should not return an error")
		_, ok := GetPrecompile(addr)
		r.True(ok, "Expected precompile at address %x", addr)
		pcAddr := ActivePrecompiles()[i]
		r.Equal(addr, pcAddr, "Expected precompile at address %x, got %x", addr, pcAddr)
	}

	pcs = ActivePrecompiles()
	r.Len(pcs, n, "Expected %d precompiles", n)
}

var (
	REQUIRED_GAS    uint64
	MUTATES_STORAGE bool
)

type testPrecompile struct {
	lib.BlankPrecompile
}

func (p *testPrecompile) RequiredGas(input []byte) uint64 {
	return REQUIRED_GAS
}

func (p *testPrecompile) MutatesStorage(input []byte) bool {
	return MUTATES_STORAGE
}

func (p *testPrecompile) Run(api cc_api.API, input []byte) (output []byte, err error) {
	api.StateDB().SetPersistentState(api.Address(), common.BytesToHash([]byte{1}), common.BytesToHash([]byte{1}))
	return nil, nil
}

var _ cc_api.Precompile = (*testPrecompile)(nil)

func TestRunPrecompile(t *testing.T) {
	var (
		r     = require.New(t)
		addr  = common.BytesToAddress([]byte{1})
		pc    = &testPrecompile{}
		evm   = cc_api_test.NewMockEVM(cc_api_test.NewMockStateDB())
		input = []byte{0}
	)
	var (
		suppliedGas = uint64(0)
		readOnly    = false
	)

	REQUIRED_GAS = uint64(10)
	MUTATES_STORAGE = true

	// Test invalid supplied gas
	_, _, err := RunPrecompile(evm, addr, pc, input, suppliedGas, readOnly)
	r.Error(err, "Expected error")

	// Test successful run and gas consumption
	for ii := uint64(1); ii < 3; ii++ {
		suppliedGas = ii * REQUIRED_GAS
		_, remainingGas, err := RunPrecompile(evm, addr, pc, input, suppliedGas, readOnly)
		r.NoError(err, "Error should be nil")
		r.Equal(suppliedGas-REQUIRED_GAS, remainingGas, "unexpected remaining gas")
	}

	suppliedGas = REQUIRED_GAS
	readOnly = true

	// Test read-only error when precompile mutates storage
	_, _, err = RunPrecompile(evm, addr, pc, input, suppliedGas, readOnly)
	r.Error(err, "Expected error")

	MUTATES_STORAGE = false

	// Test panic when non-mutating precompile attempts to mutates storage
	r.Panics(func() {
		_, _, _ = RunPrecompile(evm, addr, pc, input, suppliedGas, readOnly)
	}, "Expected panic")
}
