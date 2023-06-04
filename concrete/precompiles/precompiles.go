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
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/concrete/api"
	"github.com/ethereum/go-ethereum/concrete/lib"
)

var (
	precompiles          = make(map[common.Address]api.Precompile)
	precompiledAddresses = make([]common.Address, 0)
	precompileMetadata   = make([]PrecompileMetadata, 0)
	metadataByAddress    = make(map[common.Address]*PrecompileMetadata)
	metadataByName       = make(map[string]*PrecompileMetadata)
)

type PrecompileMetadata struct {
	Addr        common.Address `json:"addr"`
	Name        string         `json:"name"`
	Version     string         `json:"version"`
	Author      string         `json:"author"`
	Description string         `json:"description"`
	Source      string         `json:"source"`
	ABI         string         `json:"ABI"`
}

func AddPrecompile(addr common.Address, p api.Precompile, args ...interface{}) error {
	var metadata PrecompileMetadata

	if len(args) > 0 {
		if m, ok := args[0].(PrecompileMetadata); ok {
			metadata = m
		}
	}

	if _, ok := metadataByName[metadata.Name]; ok {
		return fmt.Errorf("precompile already exists with name %s", metadata.Name)
	}
	if _, ok := precompiles[addr]; ok {
		return fmt.Errorf("precompile already exists at address %x", addr)
	}

	metadata.Addr = addr

	if pwabi, ok := p.(*lib.PrecompileWithABI); ok {
		abiJson, err := json.Marshal(pwabi.ABI)
		if err != nil {
			return err
		}
		metadata.ABI = string(abiJson)
	}

	precompiles[addr] = p
	precompiledAddresses = append(precompiledAddresses, addr)
	precompileMetadata = append(precompileMetadata, metadata)
	metadataByAddress[addr] = &metadata

	if metadata.Name != "" {
		metadataByName[metadata.Name] = &metadata
	}

	return nil
}

func GetPrecompile(addr common.Address) (api.Precompile, bool) {
	pc, ok := precompiles[addr]
	return pc, ok
}

func ActivePrecompiles() []common.Address {
	return precompiledAddresses
}

func RunPrecompile(evm api.EVM, addr common.Address, p api.Precompile, input []byte, suppliedGas uint64, readOnly bool) (ret []byte, remainingGas uint64, err error) {
	gasCost := p.RequiredGas(input)
	if suppliedGas < gasCost {
		return nil, 0, errors.New("out of gas")
	}
	suppliedGas -= gasCost

	if p.MutatesStorage(input) {
		if readOnly {
			return nil, suppliedGas, errors.New("write protection")
		}
	} else {
		evm = api.NewReadOnlyEVM(evm)
	}

	API := api.New(evm, addr)
	output, err := p.Run(API, input)
	return output, suppliedGas, err
}
