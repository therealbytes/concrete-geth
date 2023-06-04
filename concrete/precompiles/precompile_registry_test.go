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
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/concrete/api"
	api_test "github.com/ethereum/go-ethereum/concrete/api/test"
	"github.com/ethereum/go-ethereum/concrete/lib"
	"github.com/stretchr/testify/require"
)

func TestPrecompileRegistry(t *testing.T) {
	var (
		r                = require.New(t)
		address          = api.PrecompileRegistryAddress
		pc               = precompiles[address].(*lib.PrecompileWithABI)
		abiJson, _       = json.Marshal(pc.ABI)
		expectedMetadata = PrecompileMetadata{
			Addr:        address,
			Name:        PrecompileRegistryMetadata.Name,
			Version:     PrecompileRegistryMetadata.Version,
			Description: PrecompileRegistryMetadata.Description,
			Author:      PrecompileRegistryMetadata.Author,
			Source:      PrecompileRegistryMetadata.Source,
			ABI:         string(abiJson),
		}
		evm = api_test.NewMockEVM(api_test.NewMockStateDB())
		API = api.New(evm, address)
	)

	// Test getFramework
	input, err := pc.ABI.Pack("getFramework")
	r.NoError(err)
	output, err := pc.Run(API, input)
	r.NoError(err)
	_frameworkData, err := pc.ABI.Unpack("getFramework", output)
	r.NoError(err)
	frameworkData, ok := _frameworkData[0].(FrameworkMetadata)
	r.True(ok)
	r.NotEmpty(frameworkData.Name)
	r.NotEmpty(frameworkData.Version)
	r.NotEmpty(frameworkData.Source)

	// Test getPrecompile
	input, err = pc.ABI.Pack("getPrecompile", address)
	r.NoError(err)
	output, err = pc.Run(API, input)
	r.NoError(err)
	_precompileData, err := pc.ABI.Unpack("getPrecompile", output)
	r.NoError(err)
	precompileData, ok := _precompileData[0].(PrecompileMetadata)
	r.True(ok)
	r.EqualValues(expectedMetadata, precompileData)

	// Test getPrecompileByName
	input, err = pc.ABI.Pack("getPrecompileByName", expectedMetadata.Name)
	r.NoError(err)
	output, err = pc.Run(API, input)
	r.NoError(err)
	_pc_addr, err := pc.ABI.Unpack("getPrecompileByName", output)
	r.NoError(err)
	pc_addr, ok := _pc_addr[0].(common.Address)
	r.True(ok)
	r.Equal(address, pc_addr)

	// Test getPrecompiledAddresses
	input, err = pc.ABI.Pack("getPrecompiledAddresses")
	r.NoError(err)
	output, err = pc.Run(API, input)
	r.NoError(err)
	_pc_addrs, err := pc.ABI.Unpack("getPrecompiledAddresses", output)
	r.NoError(err)
	pc_addrs, ok := _pc_addrs[0].([]common.Address)
	r.True(ok)
	r.Equal(3, len(pc_addrs))
	r.Contains(pc_addrs, address)

	// Test getPrecompiles
	input, err = pc.ABI.Pack("getPrecompiles")
	r.NoError(err)
	output, err = pc.Run(API, input)
	r.NoError(err)
	_pcs, err := pc.ABI.Unpack("getPrecompiles", output)
	r.NoError(err)
	pcs, ok := _pcs[0].([]PrecompileMetadata)
	r.True(ok)
	r.Equal(3, len(pcs))
	contains := false
	for _, pc := range pcs {
		if pc.Addr == address {
			contains = true
			r.EqualValues(expectedMetadata, pc)
		}
	}
	r.True(contains)
}
