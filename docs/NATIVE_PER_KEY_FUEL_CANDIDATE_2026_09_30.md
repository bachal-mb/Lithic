# Per-key native state and live fuel candidate — 2026-09-30

Status: **disabled lab candidate only**. This work does not approve gateway
registration, Makalu deployment, production pricing, LAX custody or activation.
It was tested from a separate working copy based on Lithic `df2d4fe`, Evmos
lab `45d051d` and SDK lab `f2e6295`; the Evmos native-store edit is now a
local commit at `24e674de1bd4150025562aa6e393a672736feb65`. The combined candidate is not yet a frozen,
independently retested release artifact.

## State layout and rollback

- The existing `lithovm/v1/contracts/<20-byte-address-hex>` key now holds a
  version-2 record with code hash, code and **scalar-only** storage. The key
  name is retained to keep the code lookup stable, but the record version is
  consensus-significant.
- Each map word is stored at
  `lithovm/v2/maps/<address>/<field>/<concatenated-32-byte-keys-hex>` and has
  exactly 32 value bytes. The FFI and Go bridge both validate canonical key
  shape. The VM lazily resolves an absent map entry only when an instruction
  reads it; writes stage only touched entries. All staged writes are emitted
  as one batch and applied only after validation and charging. Nested native
  calls see staged map writes through the transaction-scoped map overlay.
- Identifier and nested-key limits match validated bytecode (255 name bytes,
  up to 64 typed 32-byte keys). The callback caps encoded key bytes at 8192.
- Version-1 inline-map records are rejected; no implicit migration occurs on
  a getter or transfer. A separately reviewed migration would be required if
  any such state ever existed on-chain. There is no registered live native
  state to migrate today.
- A zero-valued map write emits a map-key deletion in the same atomic batch.
  Go rejects deletion of contract metadata and rejects non-deletion map
  values that are not exactly 32 bytes. There is no gas refund; deletion pays
  the provisional changed-write rate with no additional growth charge.
  The disabled Evmos lab now mounts a separate native KV store and has
  versioned genesis export/import checks. The store-upgrade path on an
  existing chain and production-scale genesis benchmarks are **not finished**.

## Live fuel boundary

The new FFI v2 entrypoint takes a charge callback. Each VM charge, including
initial call fuel and child-creation fuel, debits the same Go `frameMeter` that
charges physical KV reads, changed writes, state growth and emitted logs.
Synchronous child gas is added to the parent's local total without a second
external debit. A failed debit exhausts the EVM frame and discards the staged
batch. The existing FFI v1 entrypoint remains for isolated unmetered
conformance use; `ExecuteFrame` uses v2. The 10M aggregate envelope and
growth rate remain provisional, not production economics.

## Local conformance evidence

On WSL/Linux, `cargo test --workspace --locked --quiet` passed after the
storage/fuel changes; focused Rust tests cover scoped fuel, record integrity
and refusal of legacy inline records. The isolated VPS working copy passed
the tagged native-chain Go suite, including exact combined KV/VM fuel
boundaries, a 1,000-holder per-key state check, flat gas across new holders,
and an OOG write with no persisted holder. A separate adversarial test
corrupts a map value, verifies failure without writes, restores it and
verifies read/overwrite recovery; it then clears the word and verifies the
persistent key is removed and future reads return zero. The tagged full
Evmos keeper suite and targeted keeper race test also passed against that
  working copy after the deletion change. A separate cross-frame test writes
  a map word in one synchronous child call, reads it in a subsequent child
  call in the same parent transaction, and confirms a parent revert discards
  it. The native genesis codec and tagged/ordinary Evmos app/keeper suites
  also pass; see [native-store lifecycle evidence](NATIVE_STORE_LAB_LIFECYCLE_2026_09_30.md).
  `cargo fmt --check`, workspace
clippy with `-D warnings`, and `gofmt -l` also passed. These
are in-memory/isolated correctness checks, **not** durable full-block
benchmarks or evidence of 10k/100k-holder affordability.

Before a benchmark or release candidate: rerun all suites after final source
pinning; design and test a chain-approved store upgrade; measure
1k/10k/100k holders and allowances on durable
validator-class state;
obtain Foundation acceptance of the 350 GiB storage variance and final
pricing envelope; then seek independent security retesting and Makalu
end-to-end approval. No production or LAX deployment is authorized.
