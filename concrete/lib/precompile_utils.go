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
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	cc_api "github.com/ethereum/go-ethereum/concrete/api"
)

type BlankPrecompile struct{}

func (pc *BlankPrecompile) MutatesStorage(input []byte) bool {
	return false
}

func (pc *BlankPrecompile) RequiredGas(input []byte) uint64 {
	return 0
}

func (pc *BlankPrecompile) Finalise(api cc_api.API) error {
	return nil
}

func (pc *BlankPrecompile) Commit(api cc_api.API) error {
	return nil
}

func (pc *BlankPrecompile) Run(api cc_api.API, input []byte) ([]byte, error) {
	return []byte{}, nil
}

var _ cc_api.Precompile = &BlankPrecompile{}

type PrecompileDemux map[uint]cc_api.Precompile

func (d PrecompileDemux) getSelect(input []byte) uint {
	return uint(new(big.Int).SetBytes(GetData(input, 0, 32)).Uint64())
}

func (d PrecompileDemux) MutatesStorage(input []byte) bool {
	sel := d.getSelect(input)
	pc, ok := d[sel]
	if !ok {
		return false
	}
	return pc.MutatesStorage(input[32:])
}

func (d PrecompileDemux) RequiredGas(input []byte) uint64 {
	sel := d.getSelect(input)
	pc, ok := d[sel]
	if !ok {
		return 0
	}
	return pc.RequiredGas(input[32:])
}

func (d PrecompileDemux) Finalise(api cc_api.API) error {
	for _, pc := range d {
		if err := pc.Finalise(api); err != nil {
			return err
		}
	}
	return nil
}

func (d PrecompileDemux) Commit(api cc_api.API) error {
	for _, pc := range d {
		if err := pc.Commit(api); err != nil {
			return err
		}
	}
	return nil
}

func (d PrecompileDemux) Run(api cc_api.API, input []byte) ([]byte, error) {
	sel := d.getSelect(input)
	pc, ok := d[sel]
	if !ok {
		return nil, errors.New("invalid select value")
	}
	return pc.Run(api, input[32:])
}

var _ cc_api.Precompile = &PrecompileDemux{}

type MethodPrecompile interface {
	cc_api.Precompile
	Init(method abi.Method)
}

type BlankMethodPrecompile struct {
	BlankPrecompile
	Method abi.Method
}

func (p *BlankMethodPrecompile) Init(method abi.Method) {
	p.Method = method
}

func (p *BlankMethodPrecompile) MutatesStorage(input []byte) bool {
	return !p.Method.IsConstant()
}

func (p *BlankMethodPrecompile) CallRequiredGasWithArgs(requiredGas func(args []interface{}) uint64, input []byte) uint64 {
	args, err := p.Method.Inputs.UnpackValues(input)
	if err != nil {
		return 0
	}
	return requiredGas(args)
}

func (p *BlankMethodPrecompile) CallRunWithArgs(run func(concrete cc_api.API, args []interface{}) ([]interface{}, error), concrete cc_api.API, input []byte) ([]byte, error) {
	args, err := p.Method.Inputs.UnpackValues(input)
	if err != nil {
		return nil, errors.New("error unpacking arguments: " + err.Error())
	}
	returns, err := run(concrete, args)
	if err != nil {
		return nil, err
	}
	output, err := p.Method.Outputs.PackValues(returns)
	if err != nil {
		return nil, errors.New("error packing return values: " + err.Error())
	}
	return output, nil
}

var _ MethodPrecompile = &BlankMethodPrecompile{}

type PrecompileWithABI struct {
	ABI             abi.ABI
	Implementations map[string]cc_api.Precompile
}

func NewPrecompileWithABI(contractABI abi.ABI, implementations map[string]MethodPrecompile) *PrecompileWithABI {
	p := &PrecompileWithABI{
		ABI:             contractABI,
		Implementations: make(map[string]cc_api.Precompile),
	}
	for name, method := range contractABI.Methods {
		impl, ok := implementations[name]
		if !ok {
			panic("missing implementation for " + name)
		}
		impl.Init(method)
		p.Implementations[string(method.ID)] = impl
	}
	return p
}

func (p *PrecompileWithABI) getImplementation(input []byte) (cc_api.Precompile, []byte, error) {
	id := input[:4]
	input = input[4:]
	impl, ok := p.Implementations[string(id)]
	if !ok {
		return nil, nil, errors.New("invalid method ID")
	}
	return impl, input, nil
}

func (p *PrecompileWithABI) MutatesStorage(input []byte) bool {
	pc, input, err := p.getImplementation(input)
	if err != nil {
		return false
	}
	return pc.MutatesStorage(input)
}

func (p *PrecompileWithABI) RequiredGas(input []byte) uint64 {
	pc, input, err := p.getImplementation(input)
	if err != nil {
		return 0
	}
	return pc.RequiredGas(input)
}

func (p *PrecompileWithABI) Finalise(api cc_api.API) error {
	for _, pc := range p.Implementations {
		if err := pc.Finalise(api); err != nil {
			return err
		}
	}
	return nil
}

func (p *PrecompileWithABI) Commit(api cc_api.API) error {
	for _, pc := range p.Implementations {
		if err := pc.Commit(api); err != nil {
			return err
		}
	}
	return nil
}

func (p *PrecompileWithABI) Run(api cc_api.API, input []byte) ([]byte, error) {
	pc, input, err := p.getImplementation(input)
	if err != nil {
		return nil, err
	}
	return pc.Run(api, input)
}

var _ cc_api.Precompile = &PrecompileWithABI{}
