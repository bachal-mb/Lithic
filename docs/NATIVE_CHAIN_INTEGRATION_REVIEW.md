# Native chain integration proposal for owner review

Status: technically reviewed recommendation, not consensus approval or activation authorization.

## Review outcome (2026-09-26)

### Current critical-path decision

The local compiler/runtime now supports v12 strings, v13 ordered typed calls and
v14 atomic child creation, with a finance-factory fixture and host-generated
[deployment records](LITHOSCAN_NATIVE_RECEIPTS_V1.md). These supersede the older
implementation-gap statements below, not the chain integration requirements.

Before fixing the production-facing keeper/gateway interface, the chain owner
must resolve two mismatches between this proposal and the candidate:

- Creation identity: this proposal called for per-creator counters; v14 currently
  uses a factory-scoped salt and code hash. Confirm salted identity plus the public
  factory's anti-front-running/reservation policy, or require counter-based identity.
- Failure semantics: this proposal's acceptance table includes a caught child
  failure; v13/v14 deliberately abort the entire native transaction. Confirm this
  fail-whole-transaction profile for the first gateway, or require catchable frames.

Also confirm EVM-gateway-first as the route to implement for the production
integration. The recommended first harness remains nonpayable, with no native-to-EVM
callbacks, no live registration, and no claim of ordinary ERC-20 compatibility.
These are chain-owner decisions, not new requests for Amir's frontend repository.
Local experiments remain possible; they do not freeze interoperable consensus
semantics or authorize activation. The isolated, test-only StateDB/EVM harness
is [documented separately](NATIVE_CHAIN_LAB_EVIDENCE_2026_09_26.md); do not
register it in the chain app or expose a frontend/RPC path before the decision.

Recommend an EVM precompile gateway over a shared native keeper/execution
core for the first Litho Finance integration. This supersedes the earlier
Cosmos-message-first recommendation below. Amir's supplied frontend uses
viem/wagmi writeContract and EVM transaction receipts. A gateway preserves
that transport and signing workflow, although it requires a new ABI, native
payload encoder, receipt decoder and service adapter.

Use isolated candidate development with no live gateway registration. Defer
separate public Cosmos messages until a concrete consumer requires them.
The gateway has more cross-runtime rollback risk, so this recommendation is
conditional on the acceptance tests below. It is not a claim of easier or
completed integration.

### Source evidence

- The pinned Evmos source commit is
  `eca13ef2521a9ef13c32e80b1b147230bdb155b5`. Its
  `precompiles/common/precompile.go` provides RunSetup, GetCacheContext,
  MultiStoreSnapshot, readOnly checks, gas metering and AddJournalEntries.
  `x/evm/statedb/journal.go` implements precompileCallChange.Revert;
  `precompiles/staking/staking.go` demonstrates charging gas and journalling.
- The isolated lab is assembled from pinned Evmos/SDK sources and the six
  hash-verified release patches. It is not the running Makalu or mainnet app.
- The Rust TransactionalState seam commits an entire native call tree. The
  v13/v14 path supports synchronous typed calls and child return data; native
  to EVM calls, payable bridging and catchable child failure remain open.
- Amir's supplied factory creates tokens; it does not pull existing ERC-20s.
  Its frontend needs an actual transaction hash and TokenCreated receipt.

The review originally preceded the isolated FFI/StateDB lab. The dated lab
evidence now proves local EVM-frame rollback with a test-only precompile; it
does not prove live RPC, wallet signing, Makalu operation, production gas or
security correctness.

### Route comparison

| Concern | Cosmos messages | EVM gateway |
| --- | --- | --- |
| Current frontend | New message/signing/receipt integration; inspect wallet support | Retains EVM transport/signing, but changes ABI and service adapter |
| Caller | Authenticated message signer | Immediate EVM frame caller, not tx.origin |
| Rollback | SDK transaction/cache | Native cache plus enclosing EVM frame snapshots |
| Gas | SDK and native costs | EVM envelope plus native/keeper work, charged once |
| Receipts | Cosmos events need translation | Explicit native-to-EVM log encoding |
| EVM composition | Separate entry point does not enable Solidity calls | Entry into native execution possible; callbacks remain separate |
| ERC-20 compatibility | Not automatic | Not automatic: gateway access does not put ERC-20 runtime at a native address |

### Corrections required before implementation

1. Native adapter commit must merge into the current EVM frame's reversible
   state, not durable chain state. Code, storage, balances, creation counters
   and logs must revert if an enclosing EVM call later fails. Reuse and test
   StateDB cache/journal behavior with the actual chain patch set.
2. Derive native caller from the immediate EVM frame. Never authenticate using
   tx.origin or a user-supplied creator. A wrapper is itself the caller, so
   preserving user identity through a wrapper needs an explicit delegation
   rule. A native factory passes its authenticated caller to its child init.
3. Reject DELEGATECALL/CALLCODE into the first gateway. STATICCALL may use
   supported queries only: no writes, creation, value or logs. eth_call is
   distinct: it can simulate writes but must discard state afterward.
4. Start the harness nonpayable. Later payable support must transfer value
   exactly once across EVM/bank/native balances, preserve prefunding, reject
   overflow and retain patched module-account protections. Never expose the
   conformance adapter's arbitrary balance setters as bank authority.
5. EVM transaction nonce is insufficient for multiple native creations in one
   transaction. Define a journalled per-native-creator counter, versioned
   address derivation and chain domain. Check native/EVM code and reserved
   precompile/module identities. Specify failed-creation counter behavior.
6. Wrap native payload bytes in a Solidity gateway ABI; retain full bytes32
   native selectors inside the payload, distinct from EVM four-byte selectors.
   Specify log topics/data and emitter identity. Frontend validation must use
   that emitter, not assume the old Solidity factory emitted native events.
7. Bound decode/copy/output/storage/event/deployment work and meter it on
   success and failure. Integrate remaining gas with EVM forwarding rules.
   Define FFI buffer ownership, free functions, error codes and reentry limits;
   no Rust unwind or Go panic may cross FFI. No network/subprocess consensus
   execution. Pin both toolchains and test deterministic builds/results.
8. Native tokens remain gateway-managed initially. Ordinary ERC-20 calls to
   their addresses, wallet token discovery and EVM token consumers require a
   facade or routing design. Native-to-EVM transferFrom additionally requires
   ordered calls, calldata, return/revert data and reentrancy semantics. Do not
   present the first gateway as drop-in ERC-20 interoperability.

### First milestone and implementation order

1. Complete executable strings and metered factory child creation in the
   existing compiler/runtime; the UTF-8 codec and FinanceToken fixture alone
   do not meet Amir's factory acceptance case.
2. Assemble clean pinned L1 source. Add a versioned in-process Rust C ABI,
   native keeper and test-only gateway registry, disabled in ordinary builds.
3. Prove a small native stateful contract through the real EVM StateDB and
   Go/Rust seam, then run the factory. Keep nonzero value and native-to-EVM
   callbacks disabled for the first proof and document those limits.
4. Connect local EVM JSON-RPC, simulation, signing and real receipt logs.
   Integrate native code-at-height queries with the offline verification worker
   pinned to an approved executable digest. Source match is not finality.
5. Present measured evidence, final ABI/gas/upgrade/rollback plan for chain
   and security review before Makalu activation. Production and LAX retain
   their separate approval gates.

### Mandatory acceptance tests

| Case | Required result |
| --- | --- |
| Native creation succeeds, outer EVM reverts or runs out of gas | No native code/storage/log/balance/counter effects survive; normal transaction fee/nonce effects follow chain rules |
| Failed child under first fail-whole profile | Entire native transaction rolls back with deterministic remaining gas; catchable frames require separate approval |
| Two successful creates in one transaction | Distinct addresses; correct creator and counter |
| Spoofed creator, wrapper, delegate and static call modes | No tx.origin escalation; correct caller and write restrictions |
| Prefunding, balance overflow, reserved/code collisions | Preserve funds and reject invalid creation atomically |
| Malformed payload, oversized strings/events, FFI failure | Bounded deterministic failure without process crash or partial state |
| OOG during copy/write/log/init | Correct gas charge and complete rollback |
| Simulation versus submission | Simulation never persists; submitted success has real hash and matching log |
| All 16 token feature combinations | Correct creator, metadata, exact base-unit supply, flags, allowances and ownership |
| Restart/replay and explorer verification | Committed state survives; pinned-height code matches; failed/unverified statuses are retained |
| Ordinary ERC-20 call to native address | Explicitly unsupported until facade/routing exists; no false success |

### Approval timing

No additional client decision is needed to prepare or test this isolated
recommendation under the current task authorization. Chain/security owners
must accept final execution/storage/ABI/gas rules before activation. Full
frontend access is now available: the local safety branch passes application
build and service tests. Browser-wallet tests and independent compiler/runtime
work remain unfinished.

## Confirmed source boundary

The local KaJLabs/Lithosphere checkout contains the authoritative assembly
scripts and patch set under `infra/litho-mainnet-9005/bin/`, including an
immutable release manifest. The Evmos lab checkout is not the authoritative
release definition. Explorer/API/indexer sources are under `Makalu/` in the
same repository. The standalone compiler/runtime continues in KaJLabs/Lithic.

RFC 0002 in the inspected Lithosphere checkout records assigned KaJ owners but
pending E-01/E-02, V-01/V-02 and A-01 execution/storage/VM/gas/ABI decisions.
Its declaration-only implementation summary is historical, superseded by the
compiler candidate. Local development is not proof that those decisions have
been accepted by chain governance.

## Required external decision

The LithoVM/Chain owner should review the gateway recommendation and measured
evidence before activation. The earlier Cosmos-message route remains an
alternative, not a required first step. Final gas/ABI/upgrade acceptance
requires complete implementation evidence. This proposal does not authorize
deployment or block isolated technical development on another client message.

## Independent work still outstanding

The local v12-v14 candidate now executes strings, typed synchronous native
calls and atomic child creation. Bounded persisted storage and a versioned
Go/Rust FFI have isolated StateDB and EVM-call tests. Production ABI/gas rules,
release reproducibility, independent review, signed deployment, live RPC and
Makalu recovery remain unfinished. The full frontend is checked out and its
local safety branch passes build/typecheck/lint and service tests. Native
wallet integration and on-chain tests remain outstanding.
