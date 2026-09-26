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
| Typed expressions and control flow | Partial: `u64` checked arithmetic/comparison/equality, typed constants/locals, `if/else`, bounded `repeat`; versions 2, 8 and 9 | Same, plus v10 typed `require(bool)` and terminal `revert()` | Add required integer widths and operations, boolean operators, function calls, approved recursion policy or explicit rejection; differential/property fuzzing and limits evidence |
| State and storage | Partial: deterministic scalar fields, zero initialization, staged clone-and-commit | Same | Maps/collections needed by LEP100 and LEP100-15, contract namespaces, persistent host adapter, layout/version migration rules, adversarial persistence tests |
| Events | Partial: typed schema and ordered staged records | Same; failure tests prove records are not returned on revert | Consensus event commit, RPC/indexer mapping, topic/indexing rules, Makalu observation tests |
| Contract calls | Partial: typed target/selector/value intents, balance/depth checks | Same; intents are discarded on explicit failure | Calldata/arguments, synchronous host execution, return data/failure propagation, reentrancy policy/guard, nested atomicity and gas forwarding |
| Native transfers | Partial: balance-checked staged intents | Same; intents are discarded on explicit failure | Consensus bank/EVM adapter, address/denom rules, atomic host commit and Makalu success/failure tests |
| Rollback and recovery | Partial: storage clones roll back runtime and out-of-gas errors; effects exist only in successful results | v10 implements explicit `require`/`revert` and compile-runtime recovery tests | Structured failure receipt including gas used, nested-call rollback at the host seam, crash/restart recovery and Makalu rehearsal |
| ABI and deployment | Partial: deterministic native artifact/call ABI; CLI emits JSON/hex | v10 documented; v1-v9 decode compatibility retained | Constructor/init ABI, deployment transaction/RPC, code hash/address derivation, upgrade/compatibility policy, signing/broadcast and `lithdev` implementation |
| Gas semantics | Partial: deterministic instruction/statement schedule, host-intent charges and OOG rejection | v10 charges `require` expression and `revert` statement | Consensus-approved schedule, memory/storage/event/calldata pricing, nested-call forwarding/refunds, structured failed-call gas accounting and DoS benchmarks |
| LEP100 token | Missing executable implementation: `sdk/contracts/standards/lep100_ft.lithic` is illustrative syntax and fails `lithc --emit lithovm`; tests contain only one receipt-digest test | Unchanged | Finalize executable token profile; maps, `u128`/approved amount type, initialization, authorization, allowance semantics, metadata, conformance/negative vectors and audit |
| LEP100-15 | Missing: the standard is Draft and no account implementation or vectors exist | Failure primitives are groundwork only | Resolve normative signing/domain values; implement nonce/replay, unique signers/threshold, revocation, atomic batch, self-authorized changes, contract signatures, custody/recovery, reentrancy and portable vectors |
| Makalu | Missing native chain integration: library runtime only; no native RPC/module identified | Unchanged | Land compiler/runtime host adapter in the L1, deterministic deploy/call RPC, real contracts, failure/recovery/upgrade/DoS tests and recorded receipts |
| Fuzzing/security | Missing production evidence: bounded unit tests exist; no fuzz target, corpus or current toolchain security review | v10 adds decoder and compiler→runtime libFuzzer targets, reviewed seeds, bounded CI and a successful local smoke campaign | Long retained-corpus campaigns, execution/stateful differential targets, coverage review, dependency/SBOM review, threat model, independent review and remediation closure |
| Reproducible signed release | Partial: locked Rust build, pinned Actions, checksums and build provenance; archives are produced independently per hosted runner | Unchanged | Rebuild comparison or documented reproducibility level, SBOMs, malware/dependency scan, signed annotated release tag or approved keyless tag policy, installation/conformance tests |
| litho.finance deployment UI | Missing from this repository; no production deployment interface exists to integrate | Unchanged | Identify/approve site source and wallet contract, consume versioned compiler artifact, simulation/fee/network checks, gated signing/broadcast, status/retry UX and end-to-end Makalu evidence |
| LithoScan verification/status | Missing native source-verification pipeline | Unchanged | Canonical compiler settings/metadata, source bundle and artifact hash schema, deterministic rebuild worker, deployment lifecycle/status model, indexer/RPC integration and UI |
| LAX reference token | No production Lithic contract or deployment authorization | Unchanged; no deployment performed | Approved LITHO allocation input, executable audited LEP100 contract, exact `10_000_000_000 * 10^18` supply invariant, Makalu rehearsal, verification package and explicit deployment approval |

## Ordered closure path

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
