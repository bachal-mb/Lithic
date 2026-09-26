//go:build lithovm_chain_lab

package nativechain

import (
	"errors"

	storetypes "cosmossdk.io/store/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/evmos/evmos/v20/x/evm/core/vm"
	"github.com/evmos/evmos/v20/x/evm/statedb"
)

// LabPrecompile is deliberately registered only by tests. Its nonce, chain ID,
// gas pricing and receipt log are not approved consensus rules.
type LabPrecompile struct {
	Key         storetypes.StoreKey
	Environment Environment
}

var _ vm.PrecompiledContract = LabPrecompile{}

func (LabPrecompile) Address() common.Address { return LabAddress }

// Decode work is prepaid even for malformed input. ExecuteFrame separately
// charges native execution and state/log copy. This is not an approved schedule.
func (LabPrecompile) RequiredGas(input []byte) uint64 {
	return 500 + 4*uint64(len(input))
}

func (p LabPrecompile) Run(evm *vm.EVM, frame *vm.Contract, readOnly bool) ([]byte, error) {
	db, ok := evm.StateDB.(*statedb.StateDB)
	if !ok || evm.Context.BlockNumber == nil || evm.Context.Time == nil ||
		!evm.Context.BlockNumber.IsUint64() || !evm.Context.Time.IsUint64() {
		return nil, errors.New("unsupported lab EVM context")
	}
	request, err := DecodeGatewayInput(frame.Input)
	if err != nil {
		return nil, errors.New("invalid lab payload")
	}
	env := p.Environment
	env.Height = evm.Context.BlockNumber.Uint64()
	env.Timestamp = evm.Context.Time.Uint64()
	response, err := ExecuteFrame(db, p.Key, frame, readOnly, env, request)
	if err != nil {
		return nil, err
	}
	return EncodeGatewayResult(response.Result)
}

// RegisterLabPrecompile modifies only the supplied ephemeral EVM instance.
// It is never called from the chain app or production precompile registry.
func RegisterLabPrecompile(evm *vm.EVM, key storetypes.StoreKey, env Environment) {
	p := LabPrecompile{Key: key, Environment: env}
	evm.WithPrecompiles(map[common.Address]vm.PrecompiledContract{LabAddress: p}, []common.Address{LabAddress})
}
