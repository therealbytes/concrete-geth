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

package bridge

import (
	"encoding/binary"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

func Uint64ToBytes(value uint64) []byte {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, value)
	return data
}

func BytesToUint64(data []byte) uint64 {
	return binary.LittleEndian.Uint64(data)
}

const (
	Err_Success byte = iota
	Err_Error
)

type OpCode uint64

const (
	Op_StateDB_SetPersistentState OpCode = iota
	Op_StateDB_GetPersistentState
	Op_StateDB_SetEphemeralState
	Op_StateDB_GetEphemeralState
	Op_StateDB_AddPersistentPreimage
	Op_StateDB_GetPersistentPreimage
	Op_StateDB_GetPersistentPreimageSize
	Op_StateDB_AddEphemeralPreimage
	Op_StateDB_GetEphemeralPreimage
	Op_StateDB_GetEphemeralPreimageSize
)

const (
	Op_EVM_BlockHash OpCode = iota
	Op_EVM_BlockTimestamp
	Op_EVM_BlockNumber
	Op_EVM_BlockDifficulty
	Op_EVM_BlockGasLimit
	Op_EVM_BlockCoinbase
	Op_EVM_Block
)

func (opcode OpCode) Encode() []byte {
	return Uint64ToBytes(uint64(opcode))
}

func (opcode *OpCode) Decode(data []byte) {
	*opcode = OpCode(BytesToUint64(data))
}

type BlockData struct {
	Timestamp  *big.Int
	Number     *big.Int
	Difficulty *big.Int
	GasLimit   *big.Int
	Coinbase   common.Address
}

func (block *BlockData) Encode() []byte {
	data := make([]byte, 32*4+20)
	block.Timestamp.FillBytes(data[:32])
	block.Number.FillBytes(data[32:64])
	block.Difficulty.FillBytes(data[64:96])
	block.GasLimit.FillBytes(data[96:128])
	copy(data[128:148], block.Coinbase.Bytes())
	return data
}

func (block *BlockData) Decode(data []byte) {
	block.Timestamp = new(big.Int).SetBytes(data[:32])
	block.Number = new(big.Int).SetBytes(data[32:64])
	block.Difficulty = new(big.Int).SetBytes(data[64:96])
	block.GasLimit = new(big.Int).SetBytes(data[96:128])
	block.Coinbase = common.BytesToAddress(data[128:148])
}
