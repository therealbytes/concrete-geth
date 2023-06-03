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
	_ "embed"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	cc_api "github.com/ethereum/go-ethereum/concrete/api"
	"github.com/ethereum/go-ethereum/concrete/lib"
	"github.com/ethereum/go-ethereum/concrete/lib/precompiles"
	cc_precompiles "github.com/ethereum/go-ethereum/concrete/precompiles"
	"github.com/ethereum/go-ethereum/concrete/wasm"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

//go:embed wasm/testdata/typical.wasm
var typicalWasm []byte

var implementations = []struct {
	name    string
	address common.Address
	pc      cc_api.Precompile
}{
	{
		name:    "Native",
		address: common.BytesToAddress([]byte{128}),
		pc:      &precompiles.TypicalPrecompile{},
	},
	{
		name:    "Wasm",
		address: common.BytesToAddress([]byte{129}),
		pc:      wasm.NewWasmPrecompile(typicalWasm),
	},
}

func TestPrecompile(t *testing.T) {
	var (
		r             = require.New(t)
		runCounterKey = crypto.Keccak256Hash([]byte("typical.counter.0"))
		hashSetKey    = crypto.Keccak256Hash([]byte("typical.set.0"))
		key, _        = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		senderAddress = crypto.PubkeyToAddress(key.PublicKey)
		nBlocks       = 5
		nTx           = 5
	)
	for _, impl := range implementations {
		cc_precompiles.AddPrecompile(impl.address, impl.pc)
		t.Run(impl.name, func(t *testing.T) {
			var (
				gspec = &core.Genesis{
					Config: params.TestChainConfig,
					Alloc: core.GenesisAlloc{
						senderAddress: {Balance: big.NewInt(1000000000000000)},
					},
				}
				signer    = types.LatestSigner(gspec.Config)
				pcAddress = impl.address
			)

			hashes := make([]common.Hash, 0, nBlocks)
			preimages := make([][]byte, 0, nBlocks)

			db, blocks, receipts := core.GenerateChainWithGenesis(gspec, ethash.NewFaker(), nBlocks, func(ii int, block *core.BlockGen) {
				for jj := 0; jj < nTx; jj++ {
					str := fmt.Sprintf("preimage %d %d", ii, jj)
					preimage := []byte(str)
					hash := crypto.Keccak256Hash(preimage)
					preimages = append(preimages, preimage)
					hashes = append(hashes, hash)
					tx, err := types.SignTx(types.NewTransaction(block.TxNonce(senderAddress), pcAddress, common.Big0, 1_000_000, block.BaseFee(), preimage), signer, key)
					r.NoError(err)
					block.AddTx(tx)
				}
			})

			for _, block := range receipts {
				for _, receipt := range block {
					r.Equal(uint64(1), receipt.Status)
				}
			}

			root := blocks[nBlocks-1].Root()
			statedb, err := state.New(root, state.NewDatabase(db), nil)
			r.NoError(err)

			persistent := cc_api.NewCoreDatastore(cc_api.NewPersistentStorage(statedb, pcAddress))
			counter := lib.NewCounter(persistent.NewReference(runCounterKey))
			set := persistent.NewSet(hashSetKey)

			totalTxs := nBlocks * nTx

			r.Equal(new(big.Int).SetInt64(int64(totalTxs)), counter.Get())
			r.Equal(totalTxs, set.Size())

			for ii, hash := range hashes {
				preimage := preimages[ii]
				r.True(set.Has(hash))
				r.Equal(preimage, statedb.GetPersistentPreimage(hash))
			}
		})
	}
}
