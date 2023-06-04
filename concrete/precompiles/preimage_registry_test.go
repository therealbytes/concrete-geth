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
	"github.com/ethereum/go-ethereum/concrete/lib"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

type preimageRegistryPCWrapper struct {
	*lib.PrecompileWithABI
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
		r        = require.New(t)
		address  = api.PreimageRegistryAddress
		pc       = precompiles[address].(*lib.PrecompileWithABI)
		evm      = api_test.NewMockEVM(api_test.NewMockStateDB())
		API      = api.New(evm, address)
		wpc      = &preimageRegistryPCWrapper{PrecompileWithABI: pc, API: API}
		preimage = []byte("test.data")
		hash     = crypto.Keccak256Hash(preimage)
	)

	has, err := wpc.hasPreimage(hash)
	r.NoError(err)
	r.False(has)

	size, err := wpc.getPreimageSize(hash)
	r.NoError(err)
	r.Equal(uint64(0), size)

	_, err = wpc.getPreimage(size, hash)
	r.Error(err)

	// retHash, err := wpc.addPreimage(preimage)
	// r.NoError(err)
	retHash := API.Persistent().AddPreimage(preimage)
	r.Equal(hash, retHash)

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

func TestBigPreimageRegistry(t *testing.T) {
	var (
		r        = require.New(t)
		radix    = 16
		leafSize = 64
		address  = api.BigPreimageRegistryAddress
		pc       = precompiles[address].(*lib.PrecompileWithABI)
		evm      = api_test.NewMockEVM(api_test.NewMockStateDB())
		API      = api.New(evm, address)
		wpc      = &preimageRegistryPCWrapper{PrecompileWithABI: pc, API: API}
		preimage = []byte("test.data")
	)

	otherHash := crypto.Keccak256Hash([]byte("test.other"))

	has, err := wpc.hasPreimage(otherHash)
	r.NoError(err)
	r.False(has)

	size, err := wpc.getPreimageSize(otherHash)
	r.NoError(err)
	r.Equal(uint64(0), size)

	_, err = wpc.getPreimage(size, otherHash)
	r.Error(err)

	// retHash, err := wpc.addPreimage(preimage)
	// r.NoError(err)
	retHash := api.NewPersistentBigPreimageStore(API, radix, leafSize).AddPreimage(preimage)
	// r.Equal(hash, retHash)

	has, err = wpc.hasPreimage(retHash)
	r.NoError(err)
	r.True(has)

	size, err = wpc.getPreimageSize(retHash)
	r.NoError(err)
	r.Equal(uint64(len(preimage)), size)

	retPreimage, err := wpc.getPreimage(size, retHash)
	r.NoError(err)
	r.Equal(preimage, retPreimage)
}
