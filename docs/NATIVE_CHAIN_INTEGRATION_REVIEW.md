# Native chain integration proposal for owner review

Status: proposal, not consensus approval or activation authorization.

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

## Proposed first native integration

1. Add a dedicated Cosmos module with authenticated deploy/call messages.
   Derive caller and nonce from the chain transaction, never an RPC-provided
   assertion. Keep existing EVM transactions unchanged.
2. Embed the Rust runtime in-process behind a versioned C ABI and explicit
   ownership/error boundaries. Do not spawn a compiler, runtime subprocess or
   network service during consensus execution. Compile off-chain.
3. Implement TransactionalState against an isolated Cosmos cache context.
   Bank balances, code, scalar/map storage and committed events share one
   atomic transaction; failing initialization or nested execution discards it.
   Preserve prefunded deployment addresses and reject overflow.
4. Pin native bytecode/ABI, code-size limits and a byte-dependent gas schedule
   at the upgrade boundary. The candidate's current charges are test values.
5. Provide read-only native code/receipt/simulation APIs for the frontend and
   explorer. The explorer obtains code at a pinned block and invokes the
   offline verification worker with an approved executable digest.
6. Keep native-to-EVM calls disabled until their separate balance, revert,
   reentrancy and gas rules have executable conformance vectors. Amir's first
   new-token factory does not require an existing ERC-20 transferFrom call.

The dedicated Cosmos message route requires wallet signing support; an EVM
precompile gateway is an alternative with materially different authentication,
gas and interoperability requirements. Neither is currently deployed here.

## Required external decision

The LithoVM/Chain owner should accept or amend the dedicated Cosmos-message
route, its in-process runtime boundary, and the first-release interoperability
scope. Final gas/ABI/upgrade acceptance requires the subsequent complete
implementation evidence. This proposal does not request authority to deploy.

## Independent work still outstanding

Executable string arguments/storage/returns/events and their gas schedule,
contract-level child creation, ordered synchronous calls/return data,
adversarial persistence tests, release reproducibility and independent review
remain unfinished. The UTF-8 codec and configurable token fixture are partial
building blocks. Full frontend build and wallet integration also require the
application repository; the supplied six-file package is not a full checkout.
