//go:build lithovm_chain_lab

package nativechain

import (
	storetypes "cosmossdk.io/store/types"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/evmos/evmos/v20/x/evm/core/vm"
	"github.com/evmos/evmos/v20/x/evm/statedb"
	"math/big"
	"strings"
	"testing"
)

// Only EVM account backing is minimal; SDK KV cache and StateDB journals are real.
type keeper struct{}

func (keeper) GetAccount(sdk.Context, common.Address) *statedb.Account {
	return statedb.NewEmptyAccount()
}

func TestRealStateDBFactoryChildAtomicRollback(t *testing.T) {
	key := storetypes.NewKVStoreKey("lithovm-factory-lab")
	ctx := testutil.DefaultContext(key, storetypes.NewTransientStoreKey("transient-factory"))
	db := statedb.New(ctx, keeper{}, statedb.NewEmptyTxConfig(common.Hash{}))
	env := Environment{ChainID: 700777, Height: 1, Timestamp: 2, Nonce: 1}
	deployCode := func(source string, nonce uint64) string {
		req := request(t)
		req.Bytecode = ptr(fixture(t, source))
		req.Function = nil
		req.Arguments = args()
		env.Nonce = nonce
		result, err := ExecuteFrame(db, key, frame(1000000), false, env, req)
		if err != nil || !result.Success {
			t.Fatalf("deploy %s: %v %+v", source, err, result)
		}
		return returnedAddress(t, result)
	}
	template := deployCode("testdata/child.lithic", 1)
	factory := deployCode("testdata/factory.lithic", 2)
	selector := crypto.Keccak256Hash([]byte("initialize(u64)"))
	templateBytes, _ := hex.DecodeString(strings.TrimPrefix(template, "0x"))
	makeArgs := func(accept bool) string {
		flag := 0
		if accept {
			flag = 1
		}
		return fmt.Sprintf("0x4c56414c01000504%064x05%064x05%x01%064x03%064x", templateBytes, uint64(1), selector[:], uint64(42), flag)
	}
	req := request(t)
	req.Operation = "call"
	req.Bytecode = nil
	req.Contract = &factory
	req.Function = ptr("create")
	req.Arguments = makeArgs(false)
	outer := db.Snapshot()
	failed, err := ExecuteFrame(db, key, frame(1000000), false, env, req)
	if err == nil || failed.Success || len(failed.Writes) != 0 {
		t.Fatal("parent failure leaked child", err, failed)
	}
	db.RevertToSnapshot(outer)
	req.Arguments = makeArgs(true)
	outer = db.Snapshot()
	success, err := ExecuteFrame(db, key, frame(1000000), false, env, req)
	if err != nil || !success.Success || len(success.Deployments) != 1 || len(success.Events) != 2 {
		t.Fatal("creation failed", err, success)
	}
	child := returnedAddress(t, success)
	childKey := []byte("lithovm/v1/contracts/" + strings.TrimPrefix(child, "0x"))
	cache, _ := db.GetCacheContext()
	if !cache.KVStore(key).Has(childKey) {
		t.Fatal("missing staged child")
	}
	db.RevertToSnapshot(outer)
	cache, _ = db.GetCacheContext()
	if cache.KVStore(key).Has(childKey) {
		t.Fatal("outer revert retained child")
	}
	if len(db.Logs()) != 2 {
		t.Fatalf("expected two template/factory logs after revert, got %d", len(db.Logs()))
	}
	success, err = ExecuteFrame(db, key, frame(1000000), false, env, req)
	if err != nil || !success.Success {
		t.Fatal("retry failed", err)
	}
	if err := db.Commit(); err != nil {
		t.Fatal(err)
	}
	if !ctx.KVStore(key).Has(childKey) {
		t.Fatal("child not durable after commit")
	}
	db = statedb.New(ctx, keeper{}, statedb.NewEmptyTxConfig(common.Hash{}))
	get := request(t)
	get.Operation = "call"
	get.Bytecode = nil
	get.Contract = &child
	get.Function = ptr("get")
	get.Arguments = args()
	observed, err := ExecuteFrame(db, key, frame(1000000), false, env, get)
	if err != nil || !strings.HasSuffix(observed.Result, fmt.Sprintf("%064x", 42)) {
		t.Fatal("child reload", err, observed)
	}
}
func (keeper) GetState(sdk.Context, common.Address, common.Hash) common.Hash                   { return common.Hash{} }
func (keeper) GetCode(sdk.Context, common.Hash) []byte                                         { return nil }
func (keeper) ForEachStorage(sdk.Context, common.Address, func(common.Hash, common.Hash) bool) {}
func (keeper) SetAccount(sdk.Context, common.Address, statedb.Account) error                   { return nil }
func (keeper) SetState(sdk.Context, common.Address, common.Hash, []byte)                       {}
func (keeper) SetCode(sdk.Context, []byte, []byte)                                             {}
func (keeper) DeleteAccount(sdk.Context, common.Address) error                                 { return nil }
func frame(gas uint64) *vm.Contract {
	return vm.NewContract(vm.AccountRef(common.HexToAddress("0x0000000000000000000000000000000000000009")), vm.AccountRef(LabAddress), big.NewInt(0), gas)
}

func TestEphemeralEVMPrecompileCallAndOuterRevert(t *testing.T) {
	key := storetypes.NewKVStoreKey("lithovm-evm-call-lab")
	ctx := testutil.DefaultContext(key, storetypes.NewTransientStoreKey("transient-evm-call"))
	db := statedb.New(ctx, keeper{}, statedb.NewEmptyTxConfig(common.Hash{}))
	evm := vm.NewEVM(vm.BlockContext{
		CanTransfer: func(vm.StateDB, common.Address, *big.Int) bool { return true },
		Transfer:    func(vm.StateDB, common.Address, common.Address, *big.Int) {},
		BlockNumber: big.NewInt(11), Time: big.NewInt(22),
	}, vm.TxContext{Origin: common.HexToAddress("0xdead")}, db, params.AllEthashProtocolChanges, vm.Config{})
	RegisterLabPrecompile(evm, key, Environment{ChainID: 700777, Nonce: 1})
	request := request(t)
	request.Caller = "0xdead" // The EVM frame, not this field, authenticates the caller.
	request.BlockHeight = 999
	request.BlockTimestamp = 999
	payload, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	caller := vm.AccountRef(common.HexToAddress("0x0000000000000000000000000000000000000009"))
	outer := db.Snapshot()
	output, left, err := evm.Call(caller, LabAddress, payload, 1000000, big.NewInt(0))
	if err != nil || left >= 1000000 {
		t.Fatalf("EVM precompile call: %v gas=%d", err, left)
	}
	var result Response
	if err := json.Unmarshal(output, &result); err != nil || !result.Success {
		t.Fatalf("EVM result: %v %+v", err, result)
	}
	address := returnedAddress(t, result)
	stateKey := []byte("lithovm/v1/contracts/" + strings.TrimPrefix(address, "0x"))
	cache, _ := db.GetCacheContext()
	if !cache.KVStore(key).Has(stateKey) || len(db.Logs()) != 1 {
		t.Fatal("EVM call did not stage native state and log")
	}
	db.RevertToSnapshot(outer)
	cache, _ = db.GetCacheContext()
	if cache.KVStore(key).Has(stateKey) || len(db.Logs()) != 0 {
		t.Fatal("EVM outer revert retained native effects")
	}
}

func TestRealStateDBOuterRollbackCommitAndGuards(t *testing.T) {
	key := storetypes.NewKVStoreKey("lithovm-lab")
	ctx := testutil.DefaultContext(key, storetypes.NewTransientStoreKey("transient"))
	db := statedb.New(ctx, keeper{}, statedb.NewEmptyTxConfig(common.Hash{}))
	env := Environment{ChainID: 700777, Height: 1, Timestamp: 2, Nonce: 1}
	req := request(t)
	// Native success must not survive an enclosing EVM snapshot revert.
	outer := db.Snapshot()
	response, err := ExecuteFrame(db, key, frame(1000000), false, env, req)
	if err != nil {
		t.Fatal(err)
	}
	address := returnedAddress(t, response)
	stateKey := []byte("lithovm/v1/contracts/" + strings.TrimPrefix(address, "0x"))
	cache, _ := db.GetCacheContext()
	if !cache.KVStore(key).Has(stateKey) || len(db.Logs()) != 1 {
		t.Fatal("missing staged state/log")
	}
	if ctx.KVStore(key).Has(stateKey) {
		t.Fatal("premature durable state")
	}
	db.RevertToSnapshot(outer)
	cache, _ = db.GetCacheContext()
	if cache.KVStore(key).Has(stateKey) || len(db.Logs()) != 0 {
		t.Fatal("outer rollback leaked state/log")
	}
	response, err = ExecuteFrame(db, key, frame(1000000), false, env, req)
	if err != nil || !response.Success {
		t.Fatal("retry", err)
	}
	if err := db.Commit(); err != nil {
		t.Fatal(err)
	}
	if !ctx.KVStore(key).Has(stateKey) {
		t.Fatal("missing committed state")
	}
	// Reload in a fresh StateDB and read persisted Rust storage.
	db = statedb.New(ctx, keeper{}, statedb.NewEmptyTxConfig(common.Hash{}))
	req.Operation = "call"
	req.Bytecode = nil
	req.Contract = &address
	req.Function = ptr("get")
	req.Arguments = args()
	read, err := ExecuteFrame(db, key, frame(1000000), false, env, req)
	if err != nil || !strings.HasSuffix(read.Result, "0007") {
		t.Fatal("reload", err, read)
	}
	if _, err := ExecuteFrame(db, key, frame(1000000), true, env, req); err == nil {
		t.Fatal("static accepted")
	}
	paid := vm.NewContract(vm.AccountRef(common.Address{}), vm.AccountRef(LabAddress), big.NewInt(1), 1000000)
	if _, err := ExecuteFrame(db, key, paid, false, env, req); err == nil {
		t.Fatal("payable accepted")
	}
	delegate := vm.NewContract(vm.AccountRef(common.Address{}), vm.AccountRef(common.HexToAddress("0x10")), big.NewInt(0), 1000000)
	if _, err := ExecuteFrame(db, key, delegate, false, env, req); err == nil {
		t.Fatal("delegate accepted")
	}
	before := string(ctx.KVStore(key).Get(stateKey))
	req.Function = ptr("set")
	req.Arguments = args(99, 1)
	if _, err := ExecuteFrame(db, key, frame(1), false, env, req); err == nil {
		t.Fatal("OOG accepted")
	}
	if string(ctx.KVStore(key).Get(stateKey)) != before {
		t.Fatal("OOG changed committed state")
	}
}
