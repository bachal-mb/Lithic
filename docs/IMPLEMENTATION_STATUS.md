# Lithic implementation status

This repository contains a preview compiler backend plus development scaffolds. Its design documents describe intended capabilities; they do not establish production availability.

| Tool | Current implementation |
|---|---|
| lithc | Parses/checks declarations and emits ABI plus executable EVM or versioned native LithoVM bytecode for the documented subset. Native v11 adds typed single/nested map storage and checked `u256` token arithmetic to the v10 executable core. Unsupported semantics fail the complete build. |
| lithovm-host | Transactional chain-state seam plus in-memory conformance adapter. Produces structured failure/gas outcomes, atomically executes call trees, and rejects reentrancy. Deployment v1 adds code hashes, deterministic addresses, value transfer and optional initializer execution in one transaction. No Lithosphere persistence/RPC adapter is included. |
| lithfmt | Parse-checked, literal-preserving whitespace normalization; supports --check. |
| lithlint | Declaration-level naming and AI-budget rules; supports --deny-warnings. Not a security analyzer. |
| lithdev | Placeholder shell entrypoint. No deployment execution. |
| lithls, lithtest, lithsec, lithpkg | Specification-only targets. No usable implementations here. |

The SDK compiler, formatter and linter wrappers invoke the Rust commands from this checkout. lithdev remains a placeholder. The native VM strictly decodes and executes [Native LithoVM ABI v11](LITHOVM_ABI_V11.md) and versions 1 through 10. Version 11 adds typed map storage and checked `u256` add/subtract/comparison. Version 10 adds explicit transactional `require` and `revert`. Versions 9 through 3 add bounded loops, mutable locals, staged calls, transfers, events, context and scalar storage. The [transactional host v1](LITHOVM_HOST_V1.md) resolves registered zero-argument child calls synchronously at the host seam; calldata, return data, general collections, persisted consensus receipts and zk authorization are not implemented. The EVM target remains documented in [EVM backend v1](EVM_BACKEND_V1.md).

## Local development

Install Rust and the platform C/C++ linker, then run from this repository:

```sh
cargo build --workspace
cargo test --workspace
cargo run -p lithc -- --help
```

These commands build and test the toolchain. Passing them validates only the capability matrix above. No production installation, signing or deployment command is included.

Reviewed tags can produce draft cross-platform archives and checksums through
the [preview release process](RELEASE_PROCESS.md).

The independent `fuzz` workspace contains libFuzzer targets for strict
bytecode decoding and the compiler-to-runtime artifact seam. CI runs bounded
smoke campaigns; see the dated fuzz evidence for the latest local run. These
campaigns do not replace a production security review.

## Lithosphere integration

A [local execution experiment](LITHOVM_EXECUTION_LAB.md) remains historical.
`lithc --emit lithovm` now emits the versioned native artifact described in
[Native LithoVM ABI v11](LITHOVM_ABI_V11.md), while retaining v1 through v10 decoding compatibility. This establishes the combined
compiler/runtime boundary; it does not yet establish production readiness.

The tested front end from `KaJLabs/Lithosphere/toolchain` has been imported here. See [import provenance and command changes](FRONTEND_IMPORT.md).

The EVM backend targets the deployed EVM interface on LITHO. The native
compiler, interpreter and transactional host are currently library-level
components and do not
claim that a `lithic_*` RPC namespace or on-chain native module is deployed.

## LEP100-15

LEP100-15 is a draft multisignature smart-account standard. See [the supplied draft](../packages/standards/lep100/LEP100-15.md). Its example contracts, SDK calls and deployment configurations are illustrative. No implementation or mainnet address is certified by inclusion in these docs.

Interoperable signing also requires the unresolved normative values listed in
[LEP100-15 implementation decisions](LEP100_15_IMPLEMENTATION_DECISIONS.md).

Core acceptance requires deterministic signing vectors, domain and nonce replay protection, threshold and unique-signer checks, revocation, atomic execution, reentrancy protection, authorized signer changes, custody tests and contract-signature tests against the supported runtime. Recovery and optional AI/agent profiles require their own tests. Partial implementation must identify unsupported features.
