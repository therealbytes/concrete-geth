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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/concrete/api"
	api_test "github.com/ethereum/go-ethereum/concrete/api/test"
	"github.com/stretchr/testify/require"
)

// TODO: test preimage registry config

type preimageRegistryPCWrapper struct {
	PreimageRegistryPrecompile
	API api.API
}

func (p *preimageRegistryPCWrapper) addPreimage(preimage []byte) (common.Hash, error) {
	input, err := p.ABI.Pack("addPreimage", preimage)
	if err != nil {
		return common.Hash{}, err
	}
	output, err := p.Run(p.API, input)
	if err != nil {
		return common.Hash{}, err
	}
	_hash, err := p.ABI.Unpack("addPreimage", output)
	if err != nil {
		return common.Hash{}, err
	}
	hash := common.Hash(_hash[0].([32]byte))
	return hash, nil
}

func (p *preimageRegistryPCWrapper) hasPreimage(hash common.Hash) (bool, error) {
	input, err := p.ABI.Pack("hasPreimage", hash)
	if err != nil {
		return false, err
	}
	output, err := p.Run(p.API, input)
	if err != nil {
		return false, err
	}
	_has, err := p.ABI.Unpack("hasPreimage", output)
	if err != nil {
		return false, err
	}
	has := _has[0].(bool)
	return has, nil
}

func (p *preimageRegistryPCWrapper) getPreimageSize(hash common.Hash) (uint64, error) {
	input, err := p.ABI.Pack("getPreimageSize", hash)
	if err != nil {
		return 0, err
	}
	output, err := p.Run(p.API, input)
	if err != nil {
		return 0, err
	}
	_size, err := p.ABI.Unpack("getPreimageSize", output)
	if err != nil {
		return 0, err
	}
	size := _size[0].(*big.Int).Uint64()
	return size, nil
}

func (p *preimageRegistryPCWrapper) getPreimage(size uint64, hash common.Hash) ([]byte, error) {
	sizeBn := new(big.Int).SetUint64(size)
	input, err := p.ABI.Pack("getPreimage", sizeBn, hash)
	if err != nil {
		return nil, err
	}
	output, err := p.Run(p.API, input)
	if err != nil {
		return nil, err
	}
	_preimage, err := p.ABI.Unpack("getPreimage", output)
	if err != nil {
		return nil, err
	}
	preimage := _preimage[0].([]byte)
	return preimage, nil
}

func TestPreimageRegistry(t *testing.T) {
	var (
		r       = require.New(t)
		address = api.PreimageRegistryAddress
		evm     = api_test.NewMockEVM(api_test.NewMockStateDB())
		API     = api.New(evm, address)
	)

	PreimageRegistry.SetConfig(PreimageRegistryConfig{
		Enabled:  true,
		Writable: true,
	})
	BigPreimageRegistry.SetConfig(PreimageRegistryConfig{
		Enabled:  true,
		Writable: true,
	})

	registries := []struct {
		name     string
		registry *PreimageRegistryPrecompile
	}{
		{
			name:     "PreimageRegistry",
			registry: PreimageRegistry,
		},
		{
			name:     "BigPreimageRegistry",
			registry: BigPreimageRegistry,
		},
	}

	preimages := [][]byte{
		[]byte(""),
		[]byte("test.data"),
		[]byte("test.data.other"),
	}

	for _, registry := range registries {
		t.Run(registry.name, func(t *testing.T) {
			wpc := &preimageRegistryPCWrapper{*registry.registry, API}
			for _, preimage := range preimages {

				has, err := wpc.hasPreimage(common.Hash{})
				r.NoError(err)
				r.False(has)

				size, err := wpc.getPreimageSize(common.Hash{})
				r.NoError(err)
				r.Equal(uint64(0), size)

				_, err = wpc.getPreimage(size, common.Hash{})
				r.Error(err)

				hash, err := wpc.addPreimage(preimage)
				r.NoError(err)

				has, err = wpc.hasPreimage(hash)
				r.NoError(err)
				r.True(has)

				size, err = wpc.getPreimageSize(hash)
				r.NoError(err)
				r.Equal(uint64(len(preimage)), size)

				retPreimage, err := wpc.getPreimage(size, hash)
				r.NoError(err)
				r.Equal(preimage, retPreimage)
			}
		})
	}
}
