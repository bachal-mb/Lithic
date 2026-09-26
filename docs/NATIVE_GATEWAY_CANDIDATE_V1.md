# Native EVM gateway candidate v1

Status: local executable candidate, not an activated chain interface or an
approved consensus gas/ABI schedule. On 2026-09-26 the client confirmed the
requested first-profile choices: EVM gateway, salted child creation,
fail-whole native transaction semantics and direct-wallet-only top-level
deployment. Formal chain/security approval and Makalu activation remain
separate gates.
On 2026-09-27 the client acknowledged the detailed review request and agreed
to proceed with a disabled integration candidate; this is not evidence of an
independent security review or activation vote.

## Interface and trust

The first candidate exposes one nonpayable EVM precompile address only in an
ephemeral test registry. Two canonical Solidity call signatures are exercised:

```solidity
function deploy(bytes bytecode, bytes32 initializerSelector, bytes nativeArgs)
    external returns (bytes memory nativeResult);
function call(address nativeContract, bytes32 functionSelector, bytes nativeArgs)
    external returns (bytes memory nativeResult);
```

Selectors inside the payload are full Keccak-256 digests of canonical native
function signatures; the EVM method IDs are the usual first four digest bytes.
`nativeArgs` and `nativeResult` use the existing versioned LithoVM value codec.
A zero initializer selector means no initializer and requires empty native
arguments. A zero call selector is rejected. ABI decoding is strict: re-encode
must equal the entire submitted input, so aliases/trailing bytes are rejected.
The Rust FFI JSON request is internal to the Go/Rust seam, not a wallet ABI.

The immediate EVM frame supplies caller and gas; the block context supplies
height/time. The test-only keeper seam receives chain ID, sender and deployment
nonce from the Evmos message through a `GatewayTransaction` input, never calldata.
It rejects a sender/origin mismatch, invalid chain ID and fake simulation
messages when commit is requested. Fake `eth_call`/estimate messages are allowed
only in discard mode. The precompile checks the input sender against the EVM
transaction origin.
For deployment only, it additionally requires the immediate caller to be that
sender with no EVM code. Native caller identity still comes from the frame,
not `tx.origin`. EVM wrappers may call existing native contracts but cannot
top-level deploy in this profile. The precompile also rejects STATICCALL,
DELEGATECALL/CALLCODE and nonzero value.

The pinned Evmos EVM transaction context does not expose the transaction nonce
to a precompile, while its keeper has `msg.Nonce()` during state
transition. The host's top-level address currently depends on deployer, nonce,
code hash and chain ID. Thus two same-code deployments by an EVM wrapper in
one transaction would collide under a single transaction nonce. The confirmed
direct-wallet restriction avoids that first-profile ambiguity. The live keeper
must still call the tested message seam from the validated state transition,
preserve discard-only simulation and prove replay behavior before app registration.

## Failure, gas and receipts

Native child failure aborts the whole native operation and returns EVM revert;
no partial native writes, logs or child code are applied. Successful writes are
staged in Evmos StateDB and revert with an enclosing EVM snapshot. Top-level
creation still uses the host's chain-domain/nonce address rule; child creation
uses the confirmed creator/salt/code-hash rule. Salt reservation/front-running
policy and collisions with all EVM/module identities remain to be reviewed.

The lab's deliberately provisional gas formula is `500 + 4 × ABI input bytes`
precharged before decode, plus native VM gas, state read bytes, write key/value
bytes, and encoded log bytes. Input is bounded to 2 MiB; bytecode to 64 KiB;
the Rust request gas cap is 10 million. This is a testable DoS guard, **not**
the approved consensus schedule. Calldata/intrinsic charging, memory expansion,
failed-write pricing, refunds and EVM forwarding need chain-owner review and
benchmark evidence before registration. The test-only log is a candidate JSON
deployment/event envelope, not a final indexed receipt schema.

The [lab evidence](NATIVE_CHAIN_LAB_EVIDENCE_2026_09_26.md) covers canonical
codec rejection, EVM.Call deploy/call, caller derivation, failure and outer
rollback. It does not cover a signed transaction, RPC, Makalu contract,
native-to-EVM call, payable balance bridge or production frontend deployment.
