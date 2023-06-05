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

	"github.com/ethereum/go-ethereum/concrete/api"
)

var FailGas = uint64(10)

type BlankPrecompile struct{}

func (pc *BlankPrecompile) MutatesStorage(input []byte) bool {
	return false
}

func (pc *BlankPrecompile) RequiredGas(input []byte) uint64 {
	return 0
}

func (pc *BlankPrecompile) Finalise(API api.API) error {
	return nil
}

func (pc *BlankPrecompile) Commit(API api.API) error {
	return nil
}

func (pc *BlankPrecompile) Run(API api.API, input []byte) ([]byte, error) {
	return []byte{}, nil
}

var _ api.Precompile = &BlankPrecompile{}

type PrecompileDemux map[string]api.Precompile

func (d PrecompileDemux) getSelection(input []byte) (api.Precompile, []byte, error) {
	sel := input[:4]
	input = input[4:]
	pc, ok := d[string(sel)]
	if !ok {
		return nil, nil, errors.New("invalid select value")
	}
	return pc, input, nil
}

func (d PrecompileDemux) MutatesStorage(input []byte) bool {
	pc, input, err := d.getSelection(input)
	if err != nil {
		return false
	}
	return pc.MutatesStorage(input)
}

func (d PrecompileDemux) RequiredGas(input []byte) uint64 {
	pc, input, err := d.getSelection(input)
	if err != nil {
		return FailGas
	}
	return pc.RequiredGas(input)
}

func (d PrecompileDemux) Finalise(API api.API) error {
	for _, pc := range d {
		if err := pc.Finalise(API); err != nil {
			return err
		}
	}
	return nil
}

func (d PrecompileDemux) Commit(API api.API) error {
	for _, pc := range d {
		if err := pc.Commit(API); err != nil {
			return err
		}
	}
	return nil
}

func (d PrecompileDemux) Run(API api.API, input []byte) ([]byte, error) {
	pc, input, err := d.getSelection(input)
	if err != nil {
		return nil, err
	}
	return pc.Run(API, input)
}

var _ api.Precompile = &PrecompileDemux{}
