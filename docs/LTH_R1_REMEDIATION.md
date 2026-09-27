# LTH-R1 remediation tracker

Report received 2026-09-27. DOCX SHA-256 verified:
`08e3a54b757d4bc9e0c3868f90d07f141c720df4778d3d867ed5576545fa00be`.
Reviewed submission: Lithic `32bf868`, Lithosphere overlay `511b63c`.
All audit findings remain open until independent retest. No registration,
activation, LAX or production deployment is authorized.

| Finding | Remediation / remaining evidence |
| --- | --- |
| LTH-01 | Propose caller-bound salt policy for the public factory; test two senders with the same salt. Do not silently change the approved child-address identity. |
| LTH-02 | Candidate clamp forwards min(EVM frame gas, 10M) to Rust while charging actual work. Regression first failed at 10M+1/15M/30M with FFI code 2 and zero gas left; passes after clamp. Keeper boundary and RPC-facing EstimateGas tests pass locally (details below); HTTP JSON-RPC/Makalu and independent retest remain outstanding. FFI's direct-request safety cap remains intact. |
| LTH-03 | Open: derive and benchmark storage/log pricing and meter reads before work. No production gas schedule is approved; do not infer that matching SSTORE alone closes this. |
| LTH-04 | Require atomic initialization for programs declaring initialize, or authenticate initialization using deployment identity; add takeover regression. |
| LTH-05 | Four Rust guard tests added; independently disabling M4, M5, M6 and M8 makes each corresponding test fail. Guards restored; independent auditor retest still required. |
| LTH-06 | After code fixes, freeze one Lithic commit and rerun FFI/Go/keeper evidence at that pin; update overlay and immutable handoff links together. |
| LTH-07 | Build next package on Linux with forward-slash paths and Git bundles; verify raw blobs and git fsck. Preserve R1 archive unchanged. |
| LTH-08 | Carry explicit wallet fixture into L1 release evidence and require full keeper suite in the next candidate's release gate; do not rewrite frozen r1 artifacts. |
| LTH-09 | Prefix-restrict reads; design separate native store and genesis/upgrade handling before registration. |

LTH-02 regression command (Linux, Go 1.22.12, existing pinned SDK/Evmos lab
replacements and built Rust FFI required):

```sh
cd integration/native-chain
go test -mod=mod -tags lithovm_chain_lab -run TestFrameGasAboveNativeCap -count=1 .
go test -mod=mod -tags lithovm_chain_lab -count=1 .
```

The added EVM.Call test compares return bytes and gas consumed across 10M-1,
10M, 10M+1, 15M and 30M budgets, using fresh transaction contexts. This is not
an RPC estimateGas or Makalu test. The reviewer could not reproduce Go/fuzz
campaigns in R1; vendor passes must remain distinguished from independent
evidence. The review also recommends a dedicated compiler/VM audit round.

## Keeper estimation follow-up

Local chain-lab commit `45d051dc1fe2f0e9e585b09be50c1df80ec99e24`
adds boundary and estimation checks to
`TestLithoVMCandidateSignedDeploySimulationAndCommit`. The focused tagged
keeper test and full tagged keeper suite pass with Lithic gas-clamp code at `22d9b78`:

- Signed core messages at 10M-1, 10M, 10M+1, 15M and 30M preserve successful
  output and do not burn the entire allowance.
- Keeper `EstimateGas` (the RPC-facing method) succeeds at caps 10M, 30M and
  50M, both with and without an explicit gas argument. Replaying its returned
  gas estimate succeeds, and estimation does not persist the contract.
- Keeper receipt gas includes Evmos's existing minimum-gas multiplier and is
  therefore not constant as the transaction limit changes. This was not
  modified. The bridge-level work charge remains constant for the same work.

This is direct keeper invocation, not HTTP JSON-RPC or a Makalu broadcast.
The chain overlay and final review bundle will be re-pinned together after
the remaining remediations; the immutable R1 submission remains unchanged.

## LTH-05 mutation evidence

Each mutation below was applied alone, its focused test was executed and failed
on `unwrap_err()` receiving success, and the original guard was restored before
the next case. This is vendor reproduction, not independent audit closure.

| Mutant | Regression test in lithovm-ffi | Result when guard disabled |
| --- | --- | --- |
| M4 stored code hash | stored_hash_guard_rejects_well_formed_record | FAILED: accepted valid JSON with wrong stored hash |
| M5 payable balance | payable_guard_rejects_nonzero_balance_without_commit | FAILED: accepted nonzero balance |
| M6 aggregate writes | aggregate_write_guard_accepts_limit_rejects_one_byte_over | FAILED: accepted one byte above aggregate cap |
| M8 read count | read_count_guard_rejects_257th_read | FAILED: accepted 257th read |

The aggregate-write test supplies staged serialized byte accounting directly;
it isolates the aggregate limit from record-count and per-record limits. It
does not claim end-to-end contract execution producing that batch. The hash
test starts from a valid stored contract and changes only its stored hash.

## Required chain-owner decisions before profile changes

1. Approve caller-bound public-factory salt derivation. Recommended policy:
   derive an effective salt from a domain tag, authenticated immediate caller
   and user salt, before the existing child-address derivation. This changes
   predicted factory addresses and requires matching SDK/frontend calculation.
2. Approve rejecting deployment without atomic initialization for contracts
   declaring `initialize`. This is the auditor's alternative to storing a
   deployer identity. It changes the currently allowed zero-initializer profile;
   templates declaring initialize must also be initialized atomically.
3. Name the chain owner to approve gas economics and state lifecycle: baseline
   native KV charges against the pinned SDK KVGasConfig, log charges against
   EVM LOG economics, and explicit persistent-state-growth charges. Numerical
   values must be backed by benchmarks and approved before registration.

These are requests to define the remediation candidate, not requests to deploy
or activate it. LTH-06 source repinning and LTH-07 replacement packaging should
follow the profile fixes, not freeze another incomplete audit submission.
