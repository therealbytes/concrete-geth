package precompiles

import (
	_ "embed"
	"errors"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/concrete/api"
	"github.com/ethereum/go-ethereum/concrete/lib"
)

//go:embed sol/abi/PreimageRegistry.abi
var preimageRegistryABI string

var (
	PreimageRegistryMetadata = PrecompileMetadata{
		Name:        "PreimageRegistry",
		Version:     "0.1.0",
		Author:      "The concrete-geth Authors",
		Description: "A registry of stored preimages indexed by their hash.",
		Source:      "https://github.com/therealbytes/concrete-geth/tree/concrete/concrete/precompiles/preimage_registry.go",
	}
	BigPreimageRegistryMetadata = PrecompileMetadata{
		Name:        "BigPreimageRegistry",
		Version:     "0.1.0",
		Author:      "The concrete-geth Authors",
		Description: "A registry of stored preimage merkle trees indexed by their root hash.",
		Source:      "https://github.com/therealbytes/concrete-geth/tree/concrete/concrete/precompiles/pi_registry.go",
	}
)

func init() {
	abiReader := strings.NewReader(preimageRegistryABI)
	ABI, err := abi.JSON(abiReader)
	if err != nil {
		panic(err)
	}

	preimageRegistry := NewPreimageRegistry(
		ABI,
		func(concrete api.API) api.PreimageStore {
			return concrete.Persistent()
		},
		preimageRegistryGasTable{
			addPreimageGas:        10,
			addPreimagePerByteGas: 10,
			hasPreimageGas:        10,
			getPreimageSizeGas:    10,
			getPreimageGas:        10,
			getPreimagePerByteGas: 10,
		},
		false,
	)
	AddPrecompile(api.PreimageRegistryAddress, preimageRegistry, PreimageRegistryMetadata)

	bigPreimageRegistry := NewPreimageRegistry(
		ABI,
		func(concrete api.API) api.PreimageStore {
			return api.NewPersistentBigPreimageStore(concrete, -1, -1)
		},
		preimageRegistryGasTable{
			addPreimageGas:        10,
			addPreimagePerByteGas: 10,
			hasPreimageGas:        10,
			getPreimageSizeGas:    10,
			getPreimageGas:        10,
			getPreimagePerByteGas: 10,
		},
		false,
	)
	AddPrecompile(api.BigPreimageRegistryAddress, bigPreimageRegistry, BigPreimageRegistryMetadata)
}

type storeGetter func(api.API) api.PreimageStore

type preimageRegistryGasTable struct {
	addPreimageGas        uint64
	addPreimagePerByteGas uint64
	hasPreimageGas        uint64
	getPreimageSizeGas    uint64
	getPreimageGas        uint64
	getPreimagePerByteGas uint64
}

func NewPreimageRegistry(ABI abi.ABI, getStore storeGetter, gasTable preimageRegistryGasTable, enableWrites bool) api.Precompile {
	return lib.NewPrecompileWithABI(ABI, map[string]lib.MethodPrecompile{
		"addPreimage": &addPreimage{
			getStore: getStore,
			gasTable: &gasTable,
			enabled:  enableWrites,
		},
		"hasPreimage": &hasPreimage{
			getStore: getStore,
			gasTable: &gasTable,
		},
		"getPreimageSize": &getPreimageSize{
			getStore: getStore,
			gasTable: &gasTable,
		},
		"getPreimage": &getPreimage{
			getStore: getStore,
			gasTable: &gasTable,
		},
	})
}

type addPreimage struct {
	lib.BlankMethodPrecompile
	getStore storeGetter
	gasTable *preimageRegistryGasTable
	enabled  bool
}

func (p *addPreimage) RequiredGas(input []byte) uint64 {
	if len(input) < 64 {
		return 0
	}
	return p.gasTable.addPreimageGas + p.gasTable.addPreimagePerByteGas*uint64(len(input)-64)
}

func (p *addPreimage) Run(concrete api.API, input []byte) ([]byte, error) {
	if !p.enabled {
		return nil, errors.New("writes to registry are disabled")
	}
	return p.CallRunWithArgs(func(concrete api.API, args []interface{}) ([]interface{}, error) {
		preimage := args[0].([]byte)
		hash := p.getStore(concrete).AddPreimage(preimage)
		return []interface{}{hash}, nil
	}, concrete, input)
}

type hasPreimage struct {
	lib.BlankMethodPrecompile
	getStore storeGetter
	gasTable *preimageRegistryGasTable
}

func (p *hasPreimage) RequiredGas(input []byte) uint64 {
	return p.gasTable.hasPreimageGas
}

func (p *hasPreimage) Run(concrete api.API, input []byte) ([]byte, error) {
	return p.CallRunWithArgs(func(concrete api.API, args []interface{}) ([]interface{}, error) {
		hash := common.Hash(args[0].([32]byte))
		has := p.getStore(concrete).HasPreimage(hash)
		return []interface{}{has}, nil
	}, concrete, input)
}

type getPreimageSize struct {
	lib.BlankMethodPrecompile
	getStore storeGetter
	gasTable *preimageRegistryGasTable
}

func (p *getPreimageSize) RequiredGas(input []byte) uint64 {
	return p.gasTable.getPreimageSizeGas
}

func (p *getPreimageSize) Run(concrete api.API, input []byte) ([]byte, error) {
	return p.CallRunWithArgs(func(concrete api.API, args []interface{}) ([]interface{}, error) {
		hash := common.Hash(args[0].([32]byte))
		size := p.getStore(concrete).GetPreimageSize(hash)
		if size < 0 {
			size = 0
		}
		return []interface{}{big.NewInt(int64(size))}, nil
	}, concrete, input)
}

type getPreimage struct {
	lib.BlankMethodPrecompile
	getStore storeGetter
	gasTable *preimageRegistryGasTable
}

func (p *getPreimage) RequiredGas(input []byte) uint64 {
	return p.CallRequiredGasWithArgs(func(args []interface{}) uint64 {
		size := args[0].(*big.Int)
		return p.gasTable.getPreimageGas + p.gasTable.getPreimagePerByteGas*uint64(size.Int64())
	}, input)
}

func (p *getPreimage) Run(concrete api.API, input []byte) ([]byte, error) {
	return p.CallRunWithArgs(func(concrete api.API, args []interface{}) ([]interface{}, error) {
		size := int(args[0].(*big.Int).Int64())
		hash := common.Hash(args[1].([32]byte))
		store := p.getStore(concrete)
		if !store.HasPreimage(hash) {
			return []interface{}{[]byte{}}, errors.New("preimage not found")
		}
		realSize := store.GetPreimageSize(hash)
		if size < realSize {
			return []interface{}{[]byte{}}, errors.New("provided preimage size too small")
		}
		preimage := store.GetPreimage(hash)
		return []interface{}{preimage}, nil
	}, concrete, input)
}
