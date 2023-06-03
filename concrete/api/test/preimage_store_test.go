package test

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	cc_api "github.com/ethereum/go-ethereum/concrete/api"
	"github.com/ethereum/go-ethereum/concrete/crypto"
	"github.com/stretchr/testify/require"
)

func TestBigPreimageStore(t *testing.T) {
	var (
		r   = require.New(t)
		sdb = NewMockStateDB()
		evm = NewMockEVM(sdb)
		api = cc_api.New(evm, common.Address{})
	)
	var (
		radixCases    = []int{2, 4, 8, 16}
		leafSizeCases = []int{32, 128, 512}
		pi0           = []byte("hello world")
		pi1           = crypto.Keccak256(pi0)
		pi2           = make([]byte, 999)
		preimageCases = [][]byte{pi0, pi1, pi2}
	)

	for ii := 0; ii < len(pi2)/2; ii++ {
		pi2[2*ii] = byte(ii / 256)
		pi2[2*ii+1] = byte(ii % 256)
	}

	for _, radix := range radixCases {
		for _, leafSize := range leafSizeCases {
			for i, preimage := range preimageCases {
				t.Run(fmt.Sprint("r", radix, "/l", leafSize, "/pi", i), func(t *testing.T) {
					store := cc_api.NewPersistentBigPreimageStore(api, radix, leafSize)
					root := store.AddPreimage(preimage)
					retrivedPreimage := store.GetPreimage(root)
					r.True(store.HasPreimage(root))
					r.Equal(len(preimage), len(retrivedPreimage))
					r.Equal(preimage, retrivedPreimage)
				})
			}
		}
	}
}
