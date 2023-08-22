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

package api

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/concrete/utils"
	"github.com/ethereum/go-ethereum/log"
)

type Environment interface {
	Execute(op OpCode, args [][]byte) [][]byte

	// Meta
	EnableGasMetering(meter bool)
	Debug(msg string)
	TimeNow() uint64

	// Aliases
	PersistentLoad(key common.Hash) common.Hash
	PersistentStore(key common.Hash, value common.Hash)

	// Utils
	Keccak256(data []byte) common.Hash

	// Ephemeral
	EphemeralLoad_Unsafe(key common.Hash) common.Hash
	EphemeralStore_Unsafe(key common.Hash, value common.Hash)

	// Preimage oracle
	PersistentPreimageStore_Unsafe(preimage []byte) common.Hash
	PersistentPreimageLoad_Unsafe(hash common.Hash) []byte
	PersistentPreimageLoadSize_Unsafe(hash common.Hash) int
	EphemeralPreimageStore_Unsafe(preimage []byte) common.Hash
	EphemeralPreimageLoad_Unsafe(hash common.Hash) []byte
	EphemeralPreimageLoadSize_Unsafe(hash common.Hash) int

	// INTERNAL - READ
	// Address
	GetAddress() common.Address
	// Gas
	GetGasLeft() uint64
	// Block
	GetBlockNumber() uint64
	GetBlockGasLimit() uint64
	GetBlockTimestamp() uint64
	GetBlockDifficulty() *big.Int
	GetBlockBasefee() *big.Int
	GetBlockCoinbase() common.Address
	GetPrevRandao() common.Hash
	// Block hash
	GetBlockHash(block uint64) common.Hash
	// Balance
	GetBalance(address common.Address) *big.Int
	// Transaction
	GetTxGasPrice() *big.Int
	GetTxOrigin() common.Address
	// Call
	GetCallData() []byte
	GetCallDataSize() int
	GetCaller() common.Address
	GetCallValue() *big.Int
	// Storage
	StorageLoad(key common.Hash) common.Hash
	// Code
	GetCode(address common.Address) []byte
	GetCodeSize() int

	// INTERNAL - WRITE
	// Gas
	UseGas(amount uint64)
	// Storage
	StorageStore(key common.Hash, value common.Hash)
	// Log
	Log(topics []common.Hash, data []byte)

	// EXTERNAL - READ
	// Balance
	GetExternalBalance(address common.Address) *big.Int
	// Call
	CallStatic(address common.Address, data []byte, gas uint64) ([]byte, error)
	// Code
	GetExternalCode(address common.Address) []byte
	GetExternalCodeSize(address common.Address) int
	GetExternalCodeHash(address common.Address) common.Hash

	// EXTERNAL - WRITE
	// Call
	Call(address common.Address, data []byte, gas uint64, value *big.Int) ([]byte, error)
	CallDelegate(address common.Address, data []byte, gas uint64) ([]byte, error)
	// Create
	Create(data []byte, value *big.Int) (common.Address, error)
	Create2(data []byte, salt common.Hash, value *big.Int) (common.Address, error)
}

type EnvConfig struct {
	Static    bool
	Ephemeral bool
	Preimages bool
	Trusted   bool
}

type logger struct{}

func (logger) Debug(msg string) {
	log.Debug(msg)
}

type Env struct {
	table    JumpTable
	_execute func(op OpCode, env *Env, args [][]byte) [][]byte

	address common.Address
	config  EnvConfig

	logger  Logger
	statedb StateDB
	block   BlockContext
	call    CallContext
	caller  Caller

	meterGas bool
	gas      uint64

	envErr error
}

func NewEnvironment(
	address common.Address,
	config EnvConfig,
	statedb StateDB,
	block BlockContext,
	call CallContext,
	caller Caller,
	meterGas bool,
	gas uint64,
) *Env {
	env := NewNoCallEnvironment(address, config, statedb, meterGas, gas)
	env.block = block
	env.call = call
	env.caller = caller
	return env
}

func NewNoCallEnvironment(
	address common.Address,
	config EnvConfig,
	statedb StateDB,
	meterGas bool,
	gas uint64,
) *Env {
	env := &Env{
		address:  address,
		config:   config,
		logger:   logger{},
		statedb:  statedb,
		meterGas: meterGas,
		gas:      gas,
	}
	env.table = newEnvironmentMethods()
	env._execute = execute
	return env
}

func NewProxyEnvironment(execute func(op OpCode, env *Env, args [][]byte) [][]byte) *Env {
	env := &Env{
		_execute: execute,
	}
	return env
}

func execute(op OpCode, env *Env, args [][]byte) [][]byte {
	operation := env.table[op]

	if env.meterGas {
		gas := operation.constantGas
		if operation.dynamicGas != nil {
			dynamicGas, err := operation.dynamicGas(env, args)
			if err != nil {
				env.setError(err)
				return nil
			}
			gas += dynamicGas
		}
		env.UseGas(gas)
	}

	output, err := operation.execute(env, args)

	if env.config.Trusted {
		if err == ErrFeatureDisabled {
			// Panicking is preferable in trusted execution, as mistakenly using a
			// disabled feature should be caught during testing.
			panic(err)
		}
	}

	if err != nil {
		env.setError(err)
		return nil
	}
	return output
}

func (env *Env) setError(err error) {
	if env.envErr == nil {
		env.envErr = err
	}
}

func (env *Env) execute(op OpCode, args [][]byte) [][]byte {
	return env._execute(op, env, args)
}

func (env *Env) Config() EnvConfig {
	return env.config
}

func (env *Env) Gas() uint64 {
	return env.gas
}

func (env *Env) Error() error {
	return env.envErr
}

func (env *Env) Execute(op OpCode, args [][]byte) [][]byte {
	return env.execute(op, args)
}

func (env *Env) EnableGasMetering(meter bool) {
	input := [][]byte{{0x00}}
	if meter {
		input[0][0] = byte(0x01)
	}
	env.execute(EnableGasMetering_OpCode, input)
}

func (env *Env) Debug(msg string) {
	input := [][]byte{[]byte(msg)}
	env.execute(Debug_OpCode, input)
}

func (env *Env) TimeNow() uint64 {
	output := env.execute(TimeNow_OpCode, nil)
	return utils.BytesToUint64(output[0])
}

func (env *Env) PersistentLoad(key common.Hash) common.Hash {
	return env.StorageLoad(key)
}

func (env *Env) PersistentStore(key common.Hash, value common.Hash) {
	env.StorageStore(key, value)
}

func (env *Env) Keccak256(data []byte) common.Hash {
	input := [][]byte{data}
	output := env.execute(Keccak256_OpCode, input)
	hash := common.BytesToHash(output[0])
	return hash
}

func (env *Env) EphemeralLoad_Unsafe(key common.Hash) common.Hash {
	input := [][]byte{key.Bytes()}
	output := env.execute(EphemeralLoad_OpCode, input)
	hash := common.BytesToHash(output[0])
	return hash
}

func (env *Env) EphemeralStore_Unsafe(key common.Hash, value common.Hash) {
	input := [][]byte{key.Bytes(), value.Bytes()}
	env.execute(EphemeralStore_OpCode, input)
}

func (env *Env) PersistentPreimageStore_Unsafe(preimage []byte) common.Hash {
	input := [][]byte{preimage}
	output := env.execute(PersistentPreimageStore_OpCode, input)
	return common.BytesToHash(output[0])
}

func (env *Env) PersistentPreimageLoad_Unsafe(hash common.Hash) []byte {
	input := [][]byte{hash.Bytes()}
	output := env.execute(PersistentPreimageLoad_OpCode, input)
	return output[0]
}

func (env *Env) PersistentPreimageLoadSize_Unsafe(hash common.Hash) int {
	input := [][]byte{hash.Bytes()}
	output := env.execute(PersistentPreimageLoadSize_OpCode, input)
	return int(utils.BytesToUint64(output[0]))
}

func (env *Env) EphemeralPreimageStore_Unsafe(preimage []byte) common.Hash {
	input := [][]byte{preimage}
	output := env.execute(EphemeralPreimageStore_OpCode, input)
	return common.BytesToHash(output[0])
}

func (env *Env) EphemeralPreimageLoad_Unsafe(hash common.Hash) []byte {
	input := [][]byte{hash.Bytes()}
	output := env.execute(EphemeralPreimageLoad_OpCode, input)
	return output[0]
}

func (env *Env) EphemeralPreimageLoadSize_Unsafe(hash common.Hash) int {
	input := [][]byte{hash.Bytes()}
	output := env.execute(EphemeralPreimageLoadSize_OpCode, input)
	return int(utils.BytesToUint64(output[0]))
}

func (env *Env) GetAddress() common.Address {
	output := env.execute(GetAddress_OpCode, nil)
	return common.BytesToAddress(output[0])
}

func (env *Env) GetGasLeft() uint64 {
	output := env.execute(GetGasLeft_OpCode, nil)
	return utils.BytesToUint64(output[0])
}

func (env *Env) GetBlockNumber() uint64 {
	output := env.execute(GetBlockNumber_OpCode, nil)
	return utils.BytesToUint64(output[0])
}

func (env *Env) GetBlockGasLimit() uint64 {
	output := env.execute(GetBlockGasLimit_OpCode, nil)
	return utils.BytesToUint64(output[0])
}

func (env *Env) GetBlockTimestamp() uint64 {
	output := env.execute(GetBlockTimestamp_OpCode, nil)
	return utils.BytesToUint64(output[0])
}

func (env *Env) GetBlockDifficulty() *big.Int {
	output := env.execute(GetBlockDifficulty_OpCode, nil)
	return new(big.Int).SetBytes(output[0])
}

func (env *Env) GetBlockBasefee() *big.Int {
	output := env.execute(GetBlockBasefee_OpCode, nil)
	return new(big.Int).SetBytes(output[0])
}

func (env *Env) GetBlockCoinbase() common.Address {
	output := env.execute(GetBlockCoinbase_OpCode, nil)
	return common.BytesToAddress(output[0])
}

func (env *Env) GetPrevRandao() common.Hash {
	output := env.execute(GetPrevRandao_OpCode, nil)
	return common.BytesToHash(output[0])
}

func (env *Env) GetBlockHash(number uint64) common.Hash {
	input := [][]byte{utils.Uint64ToBytes(number)}
	output := env.execute(GetBlockHash_OpCode, input)
	return common.BytesToHash(output[0])
}

func (env *Env) GetBalance(address common.Address) *big.Int {
	input := [][]byte{address.Bytes()}
	output := env.execute(GetBalance_OpCode, input)
	return new(big.Int).SetBytes(output[0])
}

func (env *Env) GetTxGasPrice() *big.Int {
	output := env.execute(GetTxGasPrice_OpCode, nil)
	return new(big.Int).SetBytes(output[0])
}

func (env *Env) GetTxOrigin() common.Address {
	output := env.execute(GetTxOrigin_OpCode, nil)
	return common.BytesToAddress(output[0])
}

func (env *Env) GetCallData() []byte {
	output := env.execute(GetCallData_OpCode, nil)
	return output[0]
}

func (env *Env) GetCallDataSize() int {
	output := env.execute(GetCallDataSize_OpCode, nil)
	return int(utils.BytesToUint64(output[0]))
}

func (env *Env) GetCaller() common.Address {
	output := env.execute(GetCaller_OpCode, nil)
	return common.BytesToAddress(output[0])
}

func (env *Env) GetCallValue() *big.Int {
	output := env.execute(GetCallValue_OpCode, nil)
	return new(big.Int).SetBytes(output[0])
}

func (env *Env) StorageLoad(key common.Hash) common.Hash {
	input := [][]byte{key.Bytes()}
	output := env.execute(StorageLoad_OpCode, input)
	return common.BytesToHash(output[0])
}

func (env *Env) GetCode(address common.Address) []byte {
	input := [][]byte{address.Bytes()}
	output := env.execute(GetCode_OpCode, input)
	return output[0]
}

func (env *Env) GetCodeSize() int {
	output := env.execute(GetCodeSize_OpCode, nil)
	return int(utils.BytesToUint64(output[0]))
}

func (env *Env) UseGas(gas uint64) {
	input := [][]byte{utils.Uint64ToBytes(gas)}
	env.execute(UseGas_OpCode, input)
}

func (env *Env) StorageStore(key common.Hash, value common.Hash) {
	input := [][]byte{key.Bytes(), value.Bytes()}
	env.execute(StorageStore_OpCode, input)
}

func (env *Env) Log(topics []common.Hash, data []byte) {
	input := make([][]byte, len(topics)+1)
	for i := 0; i < len(topics); i++ {
		input[i] = topics[i].Bytes()
	}
	input[len(topics)] = data
	env.execute(Log_OpCode, input)
}

func (env *Env) GetExternalBalance(address common.Address) *big.Int {
	input := [][]byte{address.Bytes()}
	output := env.execute(GetExternalBalance_OpCode, input)
	return new(big.Int).SetBytes(output[0])
}

func (env *Env) CallStatic(address common.Address, data []byte, gas uint64) ([]byte, error) {
	input := [][]byte{utils.Uint64ToBytes(gas), address.Bytes(), data}
	output := env.execute(CallStatic_OpCode, input)
	return output[0], utils.DecodeError(output[1])
}

func (env *Env) GetExternalCode(address common.Address) []byte {
	input := [][]byte{address.Bytes()}
	output := env.execute(GetExternalCode_OpCode, input)
	return output[0]
}

func (env *Env) GetExternalCodeSize(address common.Address) int {
	input := [][]byte{address.Bytes()}
	output := env.execute(GetExternalCodeSize_OpCode, input)
	return int(utils.BytesToUint64(output[0]))
}

func (env *Env) GetExternalCodeHash(address common.Address) common.Hash {
	input := [][]byte{address.Bytes()}
	output := env.execute(GetExternalCodeHash_OpCode, input)
	return common.BytesToHash(output[0])
}

func (env *Env) Call(address common.Address, data []byte, gas uint64, value *big.Int) ([]byte, error) {
	input := [][]byte{utils.Uint64ToBytes(gas), address.Bytes(), value.Bytes(), data}
	output := env.execute(Call_OpCode, input)
	return output[0], utils.DecodeError(output[1])
}

func (env *Env) CallDelegate(address common.Address, data []byte, gas uint64) ([]byte, error) {
	input := [][]byte{utils.Uint64ToBytes(gas), address.Bytes(), data}
	output := env.execute(CallDelegate_OpCode, input)
	return output[0], utils.DecodeError(output[1])
}

func (env *Env) Create(data []byte, value *big.Int) (common.Address, error) {
	input := [][]byte{value.Bytes(), data}
	output := env.execute(Create_OpCode, input)
	return common.BytesToAddress(output[0]), utils.DecodeError(output[1])
}

func (env *Env) Create2(data []byte, salt common.Hash, value *big.Int) (common.Address, error) {
	input := [][]byte{value.Bytes(), data, salt.Bytes()}
	output := env.execute(Create2_OpCode, input)
	return common.BytesToAddress(output[0]), utils.DecodeError(output[1])
}

var _ Environment = (*Env)(nil)
