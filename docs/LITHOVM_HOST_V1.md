# LithoVM transactional host v1

Status: production-candidate interface; in-memory conformance adapter only.

`lithovm-host` is the persistence and nested-execution seam between LithoVM
and a chain runtime. It does not deploy a native module or expose an RPC.

## Transaction contract

`TransactionalState::begin_transaction` returns a `StateTransaction` that
loads and stages contract code, storage and balances. The adapter contract is:

- `commit` publishes every staged write atomically;
- dropping without `commit` publishes no writes;
- a failed `commit` publishes no writes;
- reads within a transaction observe earlier staged writes.

`InMemoryState` is the executable conformance adapter. A Lithosphere adapter
must satisfy the same contract using the chain's cache/multistore transaction.

The host also exposes the deterministic, atomic
[deployment interface v1](LITHOVM_DEPLOYMENT_V1.md). Code, code hash, attached
value and optional initializer effects share one state transaction.

## Execution and receipts

`Vm::execute_transactionally` returns `ExecutionOutcome`, preserving failure
kind, message and gas consumed. Pre-execution request failures consume zero
gas. Out-of-gas consumes the supplied limit. Runtime traps and explicit
reverts report the gas charged before failure. Existing `anyhow::Result` VM
entrypoints remain available for source compatibility.

`TransactionalHost::execute` returns a `HostOutcome`. A successful receipt
contains the top-level VM result, total gas across all frames and committed
events annotated with the emitting contract. A failed receipt identifies the
failure kind, failing contract and total gas through the failing frame.

## Nested-call semantics

### Synchronous calls (local v13 candidate)

The typed host also supports v13 `invoke` with arguments, return data and
source-ordered effects. It shares this host's transaction and event journal;
failure in a child or resumed caller discards the complete transaction. See
[v13 semantics, compatibility and limits](LITHOVM_SYNCHRONOUS_V13.md).
The deferred behavior below continues to apply to legacy frames.

### Dynamic values (local v12 candidate)

`deploy_values(DeployRequest<Value>)` and `execute_values(CallRequest<Value>)`
carry bounded typed values through the same transactional implementation.
Scalar entry points remain source-compatible and reject dynamic bytecode.
Initialization strings, storage, events, balances and code commit together or
are all discarded. Successful top-level calls return a `Value`; emitted events
retain the originating contract and exact UTF-8 fields. Dynamic transactions
are capped at 65536 bytes of encoded event-value envelopes across all frames.
This does not change deferred child-call ordering or provide call arguments and
return data to contract code. See [string semantics](LITHOVM_DYNAMIC_VALUES_V1.md).

Tests cover atomic Unicode initialization, prefunding, every insufficient gas
budget of a representative deployment, initializer revert, collision, child
revert/OOG and cross-frame event-limit rollback followed by successful recovery.

### Deferred calls

Native v10 call instructions stage target, selector and value. Host v1:

1. executes and stages the caller frame;
2. applies native transfers in their emitted order;
3. resolves each call selector through the deployed contract's entrypoint map;
4. forwards the transaction's remaining gas and executes the zero-argument
   child entrypoint synchronously;
5. commits the complete call tree once, or discards it on any failure.

The caller's storage is visible before child execution. Direct and indirect
reentrancy are rejected while a contract is active. This is a deliberate v1
policy, not an incidental recursion limit.

Events are receipt data only after commit. Parent-frame events precede events
from its deferred child calls. A failed transaction returns no event list and
does not change storage, contract balances or external balances.

## Deliberate limits

- Call intents have no calldata or return-data channel; registered child
  entrypoints must take zero arguments.
- VM effects are grouped as events, transfers and calls, so host v1 defines
  calls as deferred until local execution succeeds. A future bytecode version
  must add an ordered effect stream before source-level synchronous return
  values can be supported.
- Top-level `CallRequest.value` describes the message value; the embedding
  chain must stage the originating account debit and contract credit before
  invoking the host transaction.
- The host gas schedule has no refund mechanism.
- No disk, RPC, consensus-bank or Makalu adapter is included here. Deployment
  nonce authentication remains the embedding chain's responsibility.

These limits keep the chain seam explicit and prevent the in-memory adapter
from being mistaken for production integration.
