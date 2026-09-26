//go:build lithovm_chain_lab

package nativechain

import (
	"encoding/json"
	"errors"

	storetypes "cosmossdk.io/store/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/evmos/evmos/v20/x/evm/core/vm"
	"github.com/evmos/evmos/v20/x/evm/statedb"
)

// LabPrecompile is deliberately registered only by tests. Its JSON envelope,
// nonce and chain ID are not a production transaction ABI or consensus rule.
type LabPrecompile struct {
	Key         storetypes.StoreKey
	Environment Environment
}

var _ vm.PrecompiledContract = LabPrecompile{}

func (LabPrecompile) Address() common.Address { return LabAddress }

// The experimental dynamic charge is applied by ExecuteFrame, including copy
// costs. Production requires a reviewed gas schedule before registration.
func (LabPrecompile) RequiredGas([]byte) uint64 { return 0 }

func (p LabPrecompile) Run(evm *vm.EVM, frame *vm.Contract, readOnly bool) ([]byte, error) {
	db, ok := evm.StateDB.(*statedb.StateDB)
	if !ok || evm.Context.BlockNumber == nil || evm.Context.Time == nil ||
		!evm.Context.BlockNumber.IsUint64() || !evm.Context.Time.IsUint64() {
		return nil, errors.New("unsupported lab EVM context")
	}
	var request Request
	if len(frame.Input) > MaxBytes || json.Unmarshal(frame.Input, &request) != nil {
		return nil, errors.New("invalid lab payload")
	}
	env := p.Environment
	env.Height = evm.Context.BlockNumber.Uint64()
	env.Timestamp = evm.Context.Time.Uint64()
	response, err := ExecuteFrame(db, p.Key, frame, readOnly, env, request)
	if err != nil {
		return nil, err
	}
	return json.Marshal(response)
}

// RegisterLabPrecompile modifies only the supplied ephemeral EVM instance.
// It is never called from the chain app or production precompile registry.
func RegisterLabPrecompile(evm *vm.EVM, key storetypes.StoreKey, env Environment) {
	p := LabPrecompile{Key: key, Environment: env}
	evm.WithPrecompiles(map[common.Address]vm.PrecompiledContract{LabAddress: p}, []common.Address{LabAddress})
}
