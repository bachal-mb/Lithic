# Lithic/LithoVM preview.3 to production gap matrix

Verified: 2026-09-26

Baseline: annotated tag `lithic-v0.2.0-preview.3` at
`b5547d408d7f461d285402f6329090940d54eeae`. The tag itself has no Git
signature. GitHub Actions run `35652881357` succeeded and its Linux archive
has valid GitHub/Sigstore build provenance bound to that commit. All three
published archive hashes match their adjacent SHA-256 files:

| Platform | SHA-256 |
| --- | --- |
| Linux x86_64 | `3eda35ff5bab5da445d76e00272e591792986539305920d39b00df8bb57b4ed2` |
| Windows x86_64 | `54f63577a6aa9fddb58bf891e39a9ef07df171668472a3ce524fccbae3577fc6` |
| macOS arm64 | `a20695801b9fe964943a725b1f93d64d5aa8929d6ebb93f586a9ecff0b5be2fe` |

At verification time KaJLabs/Lithic had no open pull requests and no open
issues. The preview.3 source passed all 82 workspace tests on Linux. Windows
source tests could not link locally because MSVC Build Tools are absent.

Status meanings: **implemented** means exercised through the public
compiler/runtime seam; **partial** means useful implementation exists but the
production requirement is not closed; **missing** means no executable path was
found. Documentation and pseudocode are not counted as implementation.

| Production capability | preview.3 actual state | Current candidate | Exact production gap / acceptance evidence |
| --- | --- | --- | --- |
| Typed expressions and control flow | Partial: `u64` checked arithmetic/comparison/equality, typed constants/locals, `if/else`, bounded `repeat`; versions 2, 8 and 9 | v10 adds failures; v11 adds checked `u256` add/subtract and `>=` needed by token balances | Add remaining approved operations, boolean operators, function calls, approved recursion policy or explicit rejection; differential/property fuzzing and limits evidence |
| State and storage | Partial: deterministic scalar fields, zero initialization, staged clone-and-commit | v11 adds typed single/nested maps flattened to deterministic multi-key entries with atomic rollback | Contract namespaces, persistent chain adapter, layout/version migration rules, general collections and adversarial persistence tests |
| Events | Partial: typed schema and ordered staged records | Same; failure tests prove records are not returned on revert | Consensus event commit, RPC/indexer mapping, topic/indexing rules, Makalu observation tests |
| Contract calls | Partial: typed target/selector/value intents, balance/depth checks | Transactional host v1 resolves registered zero-argument entrypoints, forwards remaining gas, propagates structured failures, atomically executes nested frames and rejects reentrancy | Add ordered effects, calldata/arguments and return data; integrate the approved policy with the Lithosphere adapter and run nested Makalu vectors |
| Native transfers | Partial: balance-checked staged intents | Host conformance adapter atomically debits contracts, credits native accounts and rolls back transfers on nested failure | Consensus bank/EVM adapter, address/denom rules and Makalu success/failure tests |
| Rollback and recovery | Partial: storage clones roll back runtime and out-of-gas errors; effects exist only in successful results | Structured VM/host failure receipts include gas; conformance tests prove full call-tree rollback and recovery | Crash/restart recovery against the persistent Lithosphere adapter and Makalu rehearsal |
| ABI and deployment | Partial: deterministic native artifact/call ABI; CLI emits JSON/hex | v11 documented with v1-v10 compatibility; compiler emits canonical full-hash selectors and reproducible source/code metadata; host derives registrations from bytecode and adds domain-separated address derivation plus atomic value/code/initializer commit | Authenticated chain nonce, deployment gas/code limits, RPC, upgrade policy, signing/broadcast and `lithdev` implementation |
| Gas semantics | Partial: deterministic instruction/statement schedule, host-intent charges and OOG rejection | v11 adds deterministic candidate map read/write charges; VM failures report consumed gas; host forwards remaining gas and aggregates nested frames | Consensus-approved schedule, memory/event/calldata pricing, refund decision, chain integration and DoS benchmarks |
| LEP100 token | Missing executable implementation in preview.3 | v11 compiles and executes the LAX candidate with exact initial supply, metadata getters, balances, nested allowances, transfers, approvals, delegated transfers, burn, events and negative rollback vectors | Approve the executable profile, atomic deploy/init ABI, zero-address and allowance policy, distribution input, more portable conformance vectors, chain integration and audit |
| LEP100-15 | Missing: the standard is Draft and no account implementation or vectors exist | Failure primitives are groundwork only | Resolve normative signing/domain values; implement nonce/replay, unique signers/threshold, revocation, atomic batch, self-authorized changes, contract signatures, custody/recovery, reentrancy and portable vectors |
| Makalu | Missing native chain integration: library runtime only; no native RPC/module identified | Unchanged | Land compiler/runtime host adapter in the L1, deterministic deploy/call RPC, real contracts, failure/recovery/upgrade/DoS tests and recorded receipts |
| Fuzzing/security | Missing production evidence: bounded unit tests exist; no fuzz target, corpus or current toolchain security review | v10 adds decoder/compiler-runtime fuzzing; v11 adds a bounded stateful map/model target and CI job | Run and retain the new campaign, add broader execution differentials, review coverage and dependencies/SBOM, complete threat model, independent review and remediation closure |
| Reproducible signed release | Partial: locked Rust build, pinned Actions, checksums and build provenance; archives are produced independently per hosted runner | Unchanged | Rebuild comparison or documented reproducibility level, SBOMs, malware/dependency scan, signed annotated release tag or approved keyless tag policy, installation/conformance tests |
| litho.finance deployment UI | Missing from this repository; no production deployment interface exists to integrate | Versioned artifact and deployment-status schemas now define the portable compiler/status boundary | Identify/approve site source and wallet contract, implement simulation/fee/network checks, gated signing/broadcast, status/retry UX and end-to-end Makalu evidence |
| LithoScan verification/status | Missing native source-verification pipeline | Compiler artifacts and source/status schemas exist; offline `lithverify` rebuilds exact source, validates complete artifact metadata and matches independently supplied chain bytecode with negative tests | Pin reviewed compiler executable/provenance, wire trusted native RPC observations and worker resource limits, implement persisted status/finality and UI |
| LAX reference token | No production Lithic contract or deployment authorization | Source candidate compiles under v11, initializes exactly `10_000_000_000 * 10^18`, and passes atomic host deploy/init; no deployment performed | Approved LITHO allocation input, chain preservation of deploy/init atomicity, executable-profile approval, audit, Makalu rehearsal, verification package and explicit deployment approval |

## Ordered closure path

### Local string milestone, 2026-09-26

The current candidate now supports compiler-to-VM string parameters, locals,
scalar storage, events, equality and returns under bytecode v12, with bounded
UTF-8 values, byte-dependent gas and transactional rollback tests. Scalar
programs still emit v11. See [exact supported behavior and
limits](LITHOVM_DYNAMIC_VALUES_V1.md). This advances the language, storage,
events and ABI rows; it does not close any production row. The host now has
explicit dynamic-value deploy/call APIs alongside the scalar APIs. Dynamic
initialization is atomic; nested failures roll back strings and value transfers;
transaction-wide event limits are enforced. A separate v12 finance-token fixture
adds name/symbol and passes atomic initialization/getter tests for all 16 feature
profiles. Source literals, factories, ordered calls and consensus integration
remain necessary.

### Closure sequence

1. Merge and review v10 explicit failure semantics.
2. Define the persistent host interface, structured failure/gas receipt and
   nested call atomicity; implement it with an in-memory conformance adapter.
3. Add collection storage and the integer/authorization primitives required by
   an executable LEP100 token, then build conformance vectors.
4. Implement the LEP100-15 core only after its unresolved signing decisions
   are approved.
5. Integrate the native host into Lithosphere and validate deployments,
   calls, rollback, recovery and limits on Makalu.
6. Integrate the approved deploy/status interfaces with litho.finance and
   LithoScan.
7. Complete fuzzing, security review, reproducibility/SBOM evidence and
   documentation; request production and LAX deployment approvals.

No row in this matrix authorizes a production or LAX deployment.
