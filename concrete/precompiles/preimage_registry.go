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
	_ "embed"
	"errors"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/concrete/api"
	"github.com/ethereum/go-ethereum/concrete/lib"
)

//go:embed sol/abi/PreimageRegistry.abi
var preimageRegistryABI string

var (
	BigPreimageStoreRadix    = 16
	BigPreimageStoreLeafSize = 512
)

var (
	PreimageRegistry    *PreimageRegistryPrecompile
	BigPreimageRegistry *PreimageRegistryPrecompile
)

var (
	PreimageRegistryMetadata = PrecompileMetadata{
		Name:        "PreimageRegistry",
		Version:     Version{common.Big0, common.Big1, common.Big0},
		Author:      "The concrete-geth Authors",
		Description: "A registry of stored preimages indexed by their hash.",
		Source:      "https://github.com/therealbytes/concrete-geth/tree/concrete/concrete/precompiles/preimage_registry.go",
	}
	BigPreimageRegistryMetadata = PrecompileMetadata{
		Name:        "BigPreimageRegistry",
		Version:     Version{common.Big0, common.Big1, common.Big0},
		Author:      "The concrete-geth Authors",
		Description: "A registry of stored preimage merkle trees indexed by their root hash.",
		Source:      "https://github.com/therealbytes/concrete-geth/tree/concrete/concrete/precompiles/pi_registry.go",
	}
	DefaultPreimageRegistryConfig = PreimageRegistryConfig{
		Enabled:  true,
		Writable: false,
	}
	DefaultPreimageRegistryGasTable = PreimageRegistryGasTable{
		FailGas:               10,
		AddPreimageGas:        20000,
		AddPreimagePerByteGas: 100,
		HasPreimageGas:        1000,
		GetPreimageSizeGas:    1000,
		GetPreimageGas:        1000,
		GetPreimagePerByteGas: 50,
	}
	DefaultBigPreimageRegistryGasTable = PreimageRegistryGasTable{
		FailGas:               10,
		AddPreimageGas:        20000,
		AddPreimagePerByteGas: 250,
		HasPreimageGas:        1000,
		GetPreimageSizeGas:    1000,
		GetPreimageGas:        1000,
		GetPreimagePerByteGas: 125,
	}
)

var (
	ErrPreimageRegistryDisabled = errors.New("preimage registry is disabled")
	ErrPreimageRegistryReadOnly = errors.New("preimage registry is read-only")
	ErrPreimageNotFound         = errors.New("preimage not found")
	ErrPreimageTooLarge         = errors.New("provided preimage size too small")
)

func init() {
	abiReader := strings.NewReader(preimageRegistryABI)
	ABI, err := abi.JSON(abiReader)
	if err != nil {
		panic(err)
	}

	PreimageRegistry = NewPreimageRegistry(
		ABI,
		func(registry *PreimageRegistryPrecompile, API api.API) api.PreimageStore {
			return API.Persistent()
		},
		DefaultPreimageRegistryConfig,
		DefaultPreimageRegistryGasTable,
	)
	BigPreimageRegistry = NewPreimageRegistry(
		ABI,
		func(registry *PreimageRegistryPrecompile, API api.API) api.PreimageStore {
			// TODO: make configurable
			return api.NewPersistentBigPreimageStore(API, BigPreimageStoreRadix, BigPreimageStoreLeafSize)
		},
		DefaultPreimageRegistryConfig,
		DefaultBigPreimageRegistryGasTable,
	)

	AddPrecompile(api.PreimageRegistryAddress, PreimageRegistry, PreimageRegistryMetadata)
	AddPrecompile(api.BigPreimageRegistryAddress, BigPreimageRegistry, BigPreimageRegistryMetadata)
}

type PreimageRegistryGasTable struct {
	FailGas               uint64
	AddPreimageGas        uint64
	AddPreimagePerByteGas uint64
	HasPreimageGas        uint64
	GetPreimageSizeGas    uint64
	GetPreimageGas        uint64
	GetPreimagePerByteGas uint64
}

type PreimageRegistryConfig struct {
	Enabled  bool
	Writable bool
}

type PreimageRegistryPrecompile struct {
	lib.PrecompileWithABI
	getStore func(*PreimageRegistryPrecompile, api.API) api.PreimageStore
	Config   PreimageRegistryConfig
	GasTable PreimageRegistryGasTable
}

func NewPreimageRegistry(ABI abi.ABI, getStore func(*PreimageRegistryPrecompile, api.API) api.PreimageStore, config PreimageRegistryConfig, gasTable PreimageRegistryGasTable) *PreimageRegistryPrecompile {
	registry := &PreimageRegistryPrecompile{
		getStore: getStore,
		Config:   config,
		GasTable: gasTable,
	}
	registryMethod := blankPreimageRegistryMethod{registry: registry}
	registry.PrecompileWithABI = *lib.NewPrecompileWithABI(ABI, map[string]lib.MethodPrecompile{
		"addPreimage":     &addPreimage{registryMethod},
		"hasPreimage":     &hasPreimage{registryMethod},
		"getPreimageSize": &getPreimageSize{registryMethod},
		"getPreimage":     &getPreimage{registryMethod},
	})
	return registry
}

func (p *PreimageRegistryPrecompile) SetConfig(config PreimageRegistryConfig) {
	p.Config = config
}

func (p *PreimageRegistryPrecompile) RequiredGas(input []byte) uint64 {
	if !p.Config.Enabled {
		return p.GasTable.FailGas
	}
	return p.PrecompileWithABI.RequiredGas(input)
}

func (p *PreimageRegistryPrecompile) Run(API api.API, input []byte) ([]byte, error) {
	if !p.Config.Enabled {
		return nil, ErrPreimageRegistryDisabled
	}
	return p.PrecompileWithABI.Run(API, input)
}

type blankPreimageRegistryMethod struct {
	lib.BlankMethodPrecompile
	registry *PreimageRegistryPrecompile
}

type addPreimage struct {
	blankPreimageRegistryMethod
}

func (p *addPreimage) RequiredGas(input []byte) uint64 {
	if !p.registry.Config.Writable {
		return p.registry.GasTable.FailGas
	}
	if len(input) < 64 {
		return p.registry.GasTable.FailGas
	}
	return p.registry.GasTable.AddPreimageGas + p.registry.GasTable.AddPreimagePerByteGas*uint64(len(input)-64)
}

func (p *addPreimage) Run(API api.API, input []byte) ([]byte, error) {
	if !p.registry.Config.Writable {
		return nil, ErrPreimageRegistryReadOnly
	}
	return p.CallRunWithArgs(func(API api.API, args []interface{}) ([]interface{}, error) {
		preimage := args[0].([]byte)
		hash := p.registry.getStore(p.registry, API).AddPreimage(preimage)
		return []interface{}{hash}, nil
	}, API, input)
}

type hasPreimage struct {
	blankPreimageRegistryMethod
}

func (p *hasPreimage) RequiredGas(input []byte) uint64 {
	return p.registry.GasTable.HasPreimageGas
}

func (p *hasPreimage) Run(API api.API, input []byte) ([]byte, error) {
	return p.CallRunWithArgs(func(API api.API, args []interface{}) ([]interface{}, error) {
		hash := common.Hash(args[0].([32]byte))
		has := p.registry.getStore(p.registry, API).HasPreimage(hash)
		return []interface{}{has}, nil
	}, API, input)
}

type getPreimageSize struct {
	blankPreimageRegistryMethod
}

func (p *getPreimageSize) RequiredGas(input []byte) uint64 {
	return p.registry.GasTable.GetPreimageSizeGas
}

func (p *getPreimageSize) Run(API api.API, input []byte) ([]byte, error) {
	return p.CallRunWithArgs(func(API api.API, args []interface{}) ([]interface{}, error) {
		hash := common.Hash(args[0].([32]byte))
		size := p.registry.getStore(p.registry, API).GetPreimageSize(hash)
		if size < 0 {
			size = 0
		}
		return []interface{}{big.NewInt(int64(size))}, nil
	}, API, input)
}

type getPreimage struct {
	blankPreimageRegistryMethod
}

func (p *getPreimage) RequiredGas(input []byte) uint64 {
	return p.CallRequiredGasWithArgs(func(args []interface{}) uint64 {
		size := args[0].(*big.Int)
		return p.registry.GasTable.GetPreimageGas + p.registry.GasTable.GetPreimagePerByteGas*uint64(size.Int64())
	}, input)
}

func (p *getPreimage) Run(API api.API, input []byte) ([]byte, error) {
	return p.CallRunWithArgs(func(API api.API, args []interface{}) ([]interface{}, error) {
		size := int(args[0].(*big.Int).Int64())
		hash := common.Hash(args[1].([32]byte))
		store := p.registry.getStore(p.registry, API)
		if !store.HasPreimage(hash) {
			return []interface{}{[]byte{}}, ErrPreimageNotFound
		}
		realSize := store.GetPreimageSize(hash)
		if size < realSize {
			return []interface{}{[]byte{}}, ErrPreimageTooLarge
		}
		preimage := store.GetPreimage(hash)
		return []interface{}{preimage}, nil
	}, API, input)
}
