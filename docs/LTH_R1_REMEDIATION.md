# LTH-R1 remediation tracker

Report received 2026-09-27. DOCX SHA-256 verified:
`08e3a54b757d4bc9e0c3868f90d07f141c720df4778d3d867ed5576545fa00be`.
Reviewed submission: Lithic `32bf868`, Lithosphere overlay `511b63c`.
All audit findings remain open until independent retest. No registration,
activation, LAX or production deployment is authorized.

| Finding | Remediation / remaining evidence |
| --- | --- |
| LTH-01 | Propose caller-bound salt policy for the public factory; test two senders with the same salt. Do not silently change the approved child-address identity. |
| LTH-02 | Candidate clamp forwards min(EVM frame gas, 10M) to Rust while charging actual work. Regression first failed at 10M+1/15M/30M with FFI code 2 and zero gas left; passes after clamp. Add keeper boundary and actual estimateGas tests before requesting closure. FFI's direct-request safety cap remains intact. |
| LTH-03 | Open: derive and benchmark storage/log pricing and meter reads before work. No production gas schedule is approved; do not infer that matching SSTORE alone closes this. |
| LTH-04 | Require atomic initialization for programs declaring initialize, or authenticate initialization using deployment identity; add takeover regression. |
| LTH-05 | Add Rust tests that kill M4 stored-code-hash, M5 payable-balance, M6 aggregate-write-limit and M8 read-count mutants; rerun each mutation independently. |
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
