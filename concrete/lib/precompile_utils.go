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

type PrecompileDemux map[string]cc_api.Precompile

func (d PrecompileDemux) getSelection(input []byte) (cc_api.Precompile, []byte, error) {
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
		return 0
	}
	return pc.RequiredGas(input)
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
	pc, input, err := d.getSelection(input)
	if err != nil {
		return nil, err
	}
	return pc.Run(api, input)
}

var _ cc_api.Precompile = &PrecompileDemux{}
