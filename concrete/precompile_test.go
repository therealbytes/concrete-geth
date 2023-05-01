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
	"github.com/ethereum/go-ethereum/concrete/contracts"
	"github.com/ethereum/go-ethereum/concrete/lib"
	"github.com/ethereum/go-ethereum/concrete/wasm"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

//go:embed wasm/bin/typical.wasm
var typicalWasm []byte

func testPrecompile(t *testing.T, pcAddr common.Address) {
	var (
		runCounterKey = cc_api.Keccak256Hash([]byte("typical.counter.0"))
		hashSetKey    = cc_api.Keccak256Hash([]byte("typical.set.0"))
		key, _        = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		address       = crypto.PubkeyToAddress(key.PublicKey)
		funds         = big.NewInt(1000000000000000)
		gspec         = &core.Genesis{
			Config: params.TestChainConfig,
			Alloc: core.GenesisAlloc{
				pcAddr:  {Balance: common.Big1},
				address: {Balance: funds},
			},
			BaseFee: big.NewInt(params.InitialBaseFee),
		}
		signer  = types.LatestSigner(gspec.Config)
		nBlocks = 3
		nTx     = 3
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

			tx, err := types.SignTx(types.NewTransaction(block.TxNonce(address), pcAddr, common.Big0, 1_000_000, block.BaseFee(), preimage), signer, key)
			require.NoError(t, err)
			block.AddTx(tx)
		}
	})

	for _, rr := range receipts {
		for _, r := range rr {
			require.Equal(t, uint64(1), r.Status)
		}
	}

	root := blocks[nBlocks-1].Root()
	statedb, err := state.New(root, state.NewDatabase(db), nil)
	require.NoError(t, err)

	persistent := cc_api.NewCoreDatastore(cc_api.NewPersistentStorage(statedb, pcAddr))
	counter := lib.NewCounter(persistent.NewReference(runCounterKey))
	set := persistent.NewSet(hashSetKey)

	totalTxs := nBlocks * nTx

	require.Equal(t, new(big.Int).SetInt64(int64(totalTxs)), counter.Get())
	require.Equal(t, totalTxs, set.Size())

	for ii, hash := range hashes {
		preimage := preimages[ii]
		require.True(t, set.Has(hash))
		require.Equal(t, preimage, statedb.GetPersistentPreimage(hash))
	}
}

func TestNativePrecompile(t *testing.T) {
	address := common.BytesToAddress([]byte{128})
	pc := &lib.TypicalPrecompile{}
	err := contracts.AddPrecompile(address, pc)
	require.NoError(t, err)
	testPrecompile(t, address)
}

func TestWasmPrecompile(t *testing.T) {
	address := common.BytesToAddress([]byte{128})
	pc := wasm.NewWasmPrecompile(typicalWasm, address)
	err := contracts.AddPrecompile(address, pc)
	require.NoError(t, err)
	testPrecompile(t, address)
}
