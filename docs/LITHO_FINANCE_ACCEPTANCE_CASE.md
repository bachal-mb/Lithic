# Litho Finance token factory acceptance case

Inspected 2026-09-26. Client package:
`client-work/litho-finance-lithic-integration-case.zip` (in the infrastructure workspace).
SHA-256: `28aadfb91d749fec0ce67b98ad5f88c94ddb25f002d3acbba215733ec8272bac`.

The six-file package contains two Solidity contracts, two frontend TypeScript
files, ten Hardhat test cases and a README. It provides a representative
integration case, not a standalone buildable frontend: package manifests,
lockfiles, Hardhat configuration, generated types and frontend imports are absent.
No client tests were run during this source inspection.

The README lists EVM factory addresses for chains 9005 and 700777. These are
client-supplied references; inspection of this archive does not verify live
code, deployment provenance or native LithoVM availability.

## Required behavior and current gaps

| Client behavior | Current native candidate | Required acceptance |
| --- | --- | --- |
| Balances and allowances | Typed single/nested maps exist | Port ERC-20 behavior, including allowance edge cases, against the pinned OpenZeppelin 5.0.2 reference |
| Dynamic name and symbol | Fixed `bytes32` metadata in LAX fixture; standalone bounded UTF-8 value codec implemented as a prerequisite | Executable string ABI/storage/events and byte-dependent gas; do not silently truncate or replace client strings |
| Configurable decimals and supply | `u64`/`u256` values exist | Preserve `uint8` decimals range and exact base-unit supply; test non-18 decimals |
| Factory creates a child token | Top-level host deploy/init exists | Contract-level creation with deterministic child identity, metering and transaction-wide rollback |
| Supply and ownership assigned to original caller | Caller context exists | Pass creator explicitly to child initializer; factory must not receive supply or ownership |
| Mint/burn/pause/ownership flags | Not implemented by the LAX fixture | Port four flags and authorization semantics, including burnFrom and ownership renunciation |
| TokenCreated receipt | Native typed events exist | Address plus string metadata, indexed event policy and committed receipt decoding |
| Wallet submission and confirmation | No native chain adapter | Authenticated transaction, simulation, real hash, inclusion/failure and explorer status |

This factory only creates a new token. It does not pull existing ERC-20s.
Native-to-EVM `transferFrom` interoperability is a separate later acceptance
case for the other factories. Factory creation itself still requires ordered
child execution and the newly created address to be available to the parent.

The LAX fixture is not a drop-in replacement: it has fixed metadata and supply,
and does not implement configurable minting, pausing or ownership. Its approved
distribution and deployment gates remain independent of this generic factory.

## Frontend findings

`TokenCreationService.ts` currently simulates a Lithic deployment with a delay,
random hash and random address, and invokes `onSubmitted`. Replace that path
with an explicit unavailable error until a real native adapter is available.
The production UI must not present a generated identifier as a submitted transaction.
Whether this supplied branch is active in the live UI has not been verified.

The supplied Solidity receipt path also needs explicit success-status checking,
event-emitter matching against the factory, and rejection of a missing or
ambiguous TokenCreated event. Checking creator and submitted metadata further
binds the selected event to the user's request. Preserve this EVM workflow
while introducing an independently tested native adapter.

## Test plan

Retain all ten supplied tests as the initial behavioral baseline. Add tests for
factory initialization failure and retry, no partial child/storage/event commit,
creator-versus-factory identity, all feature combinations, burnFrom allowance
handling, paused mint/burn/transfer behavior, ownership changes, overflow,
out-of-gas, malformed receipts and duplicate/wrong-emitter events.

The replacement service and six passing boundary regression tests are now in
`sdk/integrations/litho-finance/`. Native requests reject before submission;
the existing Solidity path validates receipt success, factory emitter, event
cardinality, metadata and token address. Tests execute the actual service with
mocked external dependencies; this is not a live frontend integration.

Next implementation step: develop child-creation and metadata support against
the above acceptance cases. Full frontend repository access is still needed to build,
typecheck and wire the actual application. No additional workflow description
is needed from Amir for this first case.
