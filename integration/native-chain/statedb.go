//go:build lithovm_chain_lab

package nativechain

import (
	"encoding/json"
	"errors"
	"math/bits"
	"sort"
	"strconv"
	"strings"

	storetypes "cosmossdk.io/store/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/evmos/evmos/v20/x/evm/core/vm"
	"github.com/evmos/evmos/v20/x/evm/statedb"
)

// LabAddress is never registered with the EVM precompile map.
var LabAddress = common.HexToAddress("0x000000000000000000000000000000000000f000")

type Environment struct{ ChainID, Height, Timestamp, Nonce uint64 }

// ExecuteFrame uses real Evmos cache/snapshot journalling. The environment must
// come from consensus, not the request; this package provides no public RPC.
func ExecuteFrame(db *statedb.StateDB, key storetypes.StoreKey, frame *vm.Contract, readOnly bool, env Environment, request Request) (response Response, err error) {
	if readOnly || frame.Address() != LabAddress || frame.Value().Sign() != 0 {
		return Response{}, errors.New("unsupported static/delegate/payable frame")
	}
	snapshot := db.Snapshot()
	defer func() {
		if p := recover(); p != nil {
			db.RevertToSnapshot(snapshot)
			response = Response{}
			err = errors.New("native frame panic")
			_ = frame.UseGas(frame.Gas)
		}
	}()
	ctx, err := db.GetCacheContext()
	if err != nil {
		return Response{}, err
	}
	oldStore := db.MultiStoreSnapshot()
	oldEvents := ctx.EventManager().Events()
	request.Caller = strings.ToLower(frame.Caller().Hex())
	request.GasLimit = frame.Gas
	request.Nonce = env.Nonce
	request.ChainID = env.ChainID
	request.BlockHeight = env.Height
	request.BlockTimestamp = env.Timestamp
	var readBytes uint64
	response, err = Execute(request, func(k string) ([]byte, error) {
		value := ctx.KVStore(key).Get([]byte(k))
		readBytes += uint64(len(value))
		return value, nil
	})
	if err != nil {
		_ = frame.UseGas(frame.Gas)
		return Response{}, err
	}
	gas, _ := strconv.ParseUint(response.GasUsed, 10, 64)
	writes, err := response.DecodedWrites()
	if err != nil {
		return Response{}, err
	}
	// Experimental copy pricing only, not a production consensus schedule.
	var carry uint64
	gas, carry = bits.Add64(gas, readBytes, 0)
	if carry != 0 {
		_ = frame.UseGas(frame.Gas)
		return Response{}, vm.ErrOutOfGas
	}
	for k, v := range writes {
		gas, carry = bits.Add64(gas, uint64(len(k)+len(v)), 0)
		if carry != 0 {
			_ = frame.UseGas(frame.Gas)
			return Response{}, vm.ErrOutOfGas
		}
	}
	logs, err := json.Marshal(struct{ Deployments, Events []json.RawMessage }{response.Deployments, response.Events})
	if err != nil {
		return Response{}, err
	}
	gas, carry = bits.Add64(gas, uint64(len(logs)), 0)
	if carry != 0 {
		_ = frame.UseGas(frame.Gas)
		return Response{}, vm.ErrOutOfGas
	}
	if !frame.UseGas(gas) {
		_ = frame.UseGas(frame.Gas)
		return Response{}, vm.ErrOutOfGas
	}
	if !response.Success {
		return response, vm.ErrExecutionReverted
	}
	if err := db.AddPrecompileFn(LabAddress, oldStore, oldEvents); err != nil {
		db.RevertToSnapshot(snapshot)
		return Response{}, err
	}
	keys := make([]string, 0, len(writes))
	for k := range writes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		ctx.KVStore(key).Set([]byte(k), writes[k])
	}
	if len(response.Events) > 0 || len(response.Deployments) > 0 {
		db.AddLog(&ethtypes.Log{Address: LabAddress, Topics: []common.Hash{crypto.Keccak256Hash([]byte("LithoNativeLab(bytes)"))}, Data: logs})
	}
	return response, nil
}
