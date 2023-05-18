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

package concrete

import (
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/cmd/geth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/concrete/api"
	"github.com/ethereum/go-ethereum/concrete/contracts"
	"github.com/ethereum/go-ethereum/concrete/wasm"
	"github.com/urfave/cli/v2"
)

type ConcreteApp interface {
	RunWithArgs(args []string) error
	RunWithOsArgs() error
	Run() error
	AddPrecompile(addr common.Address, pc api.Precompile) error
	AddPrecompileWasm(addr common.Address, code []byte) error
}

type concreteGeth struct {
	app *cli.App
}

var ConcreteGeth ConcreteApp = &concreteGeth{
	app: geth.App,
}

func (a *concreteGeth) RunWithArgs(arguments []string) error {
	return a.app.Run(arguments)
}

func (a *concreteGeth) RunWithOsArgs() error {
	return a.RunWithArgs(os.Args)
}

func (a *concreteGeth) Run() error {
	return a.RunWithOsArgs()
}

func (a *concreteGeth) validateNewPCAddress(addr common.Address) error {
	if addr.Big().Cmp(big.NewInt(128)) < 0 {
		return fmt.Errorf("precompile address cannot be below 0x80")
	}
	return nil
}

func (a *concreteGeth) AddPrecompile(addr common.Address, pc api.Precompile) error {
	if err := a.validateNewPCAddress(addr); err != nil {
		return err
	}
	return contracts.AddPrecompile(addr, pc)
}

func (a *concreteGeth) AddPrecompileWasm(addr common.Address, code []byte) error {
	if err := a.validateNewPCAddress(addr); err != nil {
		return err
	}
	return contracts.AddPrecompile(addr, wasm.NewWasmPrecompile(code))
}
