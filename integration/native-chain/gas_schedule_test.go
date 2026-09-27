//go:build lithovm_chain_lab

package nativechain

import (
	storetypes "cosmossdk.io/store/types"
	"encoding/json"
	"errors"
	"github.com/cosmos/cosmos-sdk/testutil"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/evmos/evmos/v20/x/evm/core/vm"
	"github.com/evmos/evmos/v20/x/evm/statedb"
	"math"
	"strconv"
	"strings"
	"testing"
)

func TestNewStatePaysKVAndGrowthFloor(t *testing.T) {
	key := storetypes.NewKVStoreKey("gas-pricing")
	ctx := testutil.DefaultContext(key, storetypes.NewTransientStoreKey("gas-pricing-transient"))
	db := statedb.New(ctx, keeper{}, statedb.NewEmptyTxConfig(common.Hash{}))
	f := frame(10_000_000)
	result, err := ExecuteFrame(db, key, f, false, Environment{ChainID: 700777, Height: 1, Timestamp: 2, Nonce: 1}, request(t))
	if err != nil || !result.Success {
		t.Fatalf("deploy: %v %+v", err, result)
	}
	writes, err := result.DecodedWrites()
	if err != nil {
		t.Fatal(err)
	}
	var floor uint64
	cfg := storetypes.KVGasConfig()
	for k, v := range writes {
		size := uint64(len(k) + len(v))
		floor += cfg.WriteCostFlat + cfg.WriteCostPerByte*size
		// Candidate persistent-growth floor: rounded 32-byte allocation units.
		floor += ((size + 31) / 32) * params.SstoreSetGasEIP2200
	}
	if used := uint64(10_000_000) - f.Gas; used < floor {
		t.Fatalf("new state underpriced: charged=%d minimum-write-and-growth=%d", used, floor)
	}
}

func TestCandidateRatesMatchPinnedBaselinesAndDoNotOverflow(t *testing.T) {
	cfg := storetypes.KVGasConfig()
	if candidateReadFlat != cfg.ReadCostFlat || candidateReadByte != cfg.ReadCostPerByte || candidateWriteFlat != cfg.WriteCostFlat || candidateWriteByte != cfg.WriteCostPerByte || candidateLogFlat != params.LogGas+params.LogTopicGas || candidateLogByte != params.LogDataGas || candidateGrowthUnit != params.SstoreSetGasEIP2200 {
		t.Fatal("dependency baseline changed; explicit schedule review required")
	}
	if gasCost(1, math.MaxUint64, 2) != math.MaxUint64 || gasSum(math.MaxUint64, 1) != math.MaxUint64 {
		t.Fatal("gas arithmetic wrapped")
	}
	for _, c := range []struct{ old, next, want uint64 }{{0, 1, 20000}, {0, 32, 20000}, {32, 33, 20000}, {33, 63, 0}, {64, 1, 0}, {32, 32, 0}} {
		if got := growthGas(c.old, c.next); got != c.want {
			t.Fatalf("growth %d -> %d: %d != %d", c.old, c.next, got, c.want)
		}
	}
}

func TestExactDeploymentGasBoundaryAndNoEffectsOnExhaustion(t *testing.T) {
	req := request(t)
	var exact uint64
	for step := 0; step < 4; step++ {
		// First measure, then exact budget, then one gas below, then no lookup budget.
		budget := uint64(10_000_000)
		switch step {
		case 1:
			budget = exact
		case 2:
			budget = exact - 1
		case 3:
			budget = 1
		}
		key := storetypes.NewKVStoreKey("boundary")
		ctx := testutil.DefaultContext(key, storetypes.NewTransientStoreKey("boundary-t"))
		db := statedb.New(ctx, keeper{}, statedb.NewEmptyTxConfig(common.Hash{}))
		f := frame(budget)
		result, err := ExecuteFrame(db, key, f, false, Environment{ChainID: 700777, Height: 1, Timestamp: 2, Nonce: 1}, req)
		if step == 0 {
			if err != nil {
				t.Fatal(err)
			}
			exact = budget - f.Gas
			continue
		}
		if step == 1 {
			if err != nil || !result.Success || f.Gas != 0 {
				t.Fatalf("exact budget failed: %v gas=%d", err, f.Gas)
			}
			continue
		}
		if !errors.Is(err, vm.ErrOutOfGas) || f.Gas != 0 || result.Success || len(db.Logs()) != 0 {
			t.Fatalf("OOG leaked or misclassified: %v %+v", err, result)
		}
		cache, _ := db.GetCacheContext()
		it := cache.KVStore(key).Iterator(nil, nil)
		exists := it.Valid()
		it.Close()
		if exists {
			t.Fatal("OOG retained storage")
		}
	}
}

func TestGetterPaysReadsButNoWriteOrEmptyLog(t *testing.T) {
	key := storetypes.NewKVStoreKey("getter-pricing")
	ctx := testutil.DefaultContext(key, storetypes.NewTransientStoreKey("getter-t"))
	db := statedb.New(ctx, keeper{}, statedb.NewEmptyTxConfig(common.Hash{}))
	req := request(t)
	env := Environment{ChainID: 700777, Height: 1, Timestamp: 2, Nonce: 1}
	result, err := ExecuteFrame(db, key, frame(10_000_000), false, env, req)
	if err != nil {
		t.Fatal(err)
	}
	writes, _ := result.DecodedWrites()
	var size uint64
	for k, v := range writes {
		size += uint64(len(k) + len(v))
	}
	if err := db.Commit(); err != nil {
		t.Fatal(err)
	}
	contract := returnedAddress(t, result)
	req.Operation = "call"
	req.Bytecode = nil
	req.Contract = &contract
	req.Function = ptr("get")
	req.Arguments = args()
	db = statedb.New(ctx, keeper{}, statedb.NewEmptyTxConfig(common.Hash{}))
	f := frame(10_000_000)
	result, err = ExecuteFrame(db, key, f, false, env, req)
	if err != nil {
		t.Fatal(err)
	}
	native, _ := strconv.ParseUint(result.GasUsed, 10, 64)
	// One logical record load uses two callbacks (size then data), each a Get.
	expected := native + 2*gasCost(candidateReadFlat, candidateReadByte, size)
	if used := uint64(10_000_000) - f.Gas; used != expected || len(db.Logs()) != 0 {
		t.Fatalf("getter gas=%d expected=%d logs=%d", used, expected, len(db.Logs()))
	}
	req.Function = ptr("set")
	req.Arguments = args(7, 1) // unchanged value, but emits a log
	db = statedb.New(ctx, keeper{}, statedb.NewEmptyTxConfig(common.Hash{}))
	f = frame(10_000_000)
	result, err = ExecuteFrame(db, key, f, false, env, req)
	if err != nil {
		t.Fatal(err)
	}
	native, _ = strconv.ParseUint(result.GasUsed, 10, 64)
	logs, _ := json.Marshal(struct{ Deployments, Events []json.RawMessage }{result.Deployments, result.Events})
	expected = native + 2*gasCost(candidateReadFlat, candidateReadByte, size) + gasCost(candidateLogFlat, candidateLogByte, uint64(len(logs)))
	if used := uint64(10_000_000) - f.Gas; used != expected || len(db.Logs()) != 1 {
		t.Fatalf("log pricing gas=%d expected=%d", used, expected)
	}
}

func TestNativeKeysAreCanonicalAndNamespaceRestricted(t *testing.T) {
	for _, key := range []string{"", "other/" + strings.Repeat("0", 40), "lithovm/v1/contracts/" + strings.Repeat("A", 40), "lithovm/v1/contracts/" + strings.Repeat("z", 40), "lithovm/v1/contracts/00"} {
		if validStateKey(key) {
			t.Fatalf("accepted %q", key)
		}
	}
	if !validStateKey("lithovm/v1/contracts/" + strings.Repeat("a", 40)) {
		t.Fatal("rejected canonical key")
	}
}
