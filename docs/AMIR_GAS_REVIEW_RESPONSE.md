# Amir gas review: receipt, ancestry and remaining acceptance work

Recorded 2026-09-27 from the review relayed by the user. This records Foundation
technical decisions, not an independent security verdict. Registration, deployment,
activation and production custody/use remain unauthorized.

## Source ancestry confirmed

- Implementation: `a64703faf831736afa21a1fea2a92808b9ee478c`.
- Reviewed packet source: `fcb2f47aaecea1546b8a20ba6dc596a111d2719e`.
- `fcb2f47^` equals the full implementation hash above: it is the immediate parent.
- `git merge-base --is-ancestor a64703f fcb2f47` returned exit 0.
- `integration/native-chain` tree ID at **both** commits:
  `ac6a9e1c6ab296bbc4e3c282c73fc12ba9946e33`.
- The only intervening changes are an added `docs/AMIR_GAS_REVIEW_HANDOFF.md`
  and edits to `docs/LTH_R1_GAS_SCHEDULE_PROPOSAL.md` and
  `docs/LTH_R1_REMEDIATION.md`. No implementation, tests or benchmark output changed.

Thus the reviewed source contains the `a64703f` implementation unchanged.
This resolves this ancestry question, **not final source-pin/audit closure**;
additional required engineering will produce a new candidate to pin and retest.

Original reviewed ZIP remains unchanged:
`LITHIC_GAS_REVIEW_FOR_AMIR_fcb2f47.zip`, 84,701 bytes, SHA-256
`5e38b43fc545fb9ab18477b23777cc131a7db3fe73d288896a248cbab3c0b188`.
All 21 file entries were verified against the committed Git blobs during packaging;
the ZIP digest was checked again when recording this response.

## Decisions recorded

| Item | Amir's decision | Scope |
| --- | --- | --- |
| Physical reads: 1,000 + 3 per key/value byte | Accept | Disabled remediation candidate only |
| Changed writes: 2,000 + 30 per key/new-value byte | Accept | Disabled remediation candidate only |
| Actual logs: 750 + 8 per emitted data byte | Accept | Disabled remediation candidate only |
| Growth: 20,000 per additional rounded 32-byte unit | Defer final policy | May remain conservative disabled-test candidate |
| Aggregate native cap: 10M | Defer final policy | May remain current safety/test envelope |
| Whole-record persistence | Not accepted as production Finance/LAX layout | Implement scalable per-key persistence |

No numerical change is necessary merely to record this review. Neither accepted
KV/log rates nor deferred growth/cap decisions authorize production use.

## Live configuration observation: checked-in 100M is not the live ceiling

Read-only public RPC observations on 2026-09-27:

- `/status` at 12:59:50 UTC identified `lithosphere_9005-1`, latest height
  10,109,487, latest block time 12:59:48.797 UTC, `catching_up=false`.
- At 13:00:12 UTC,
  [height-pinned consensus parameters](https://rpc-mainnet.litho.ai/consensus_params?height=10109488)
  returned block height **10,109,488**, `max_gas="-1"`,
  `max_bytes="22020096"`.
- At the same height, `eth_getBlockByNumber("0x9a4230", false)` at
  `https://rpc-mainnet.litho.ai` returned `gasLimit="0xffffffff"`
  (4,294,967,295) and block hash
  `0xebcd35c64d95fdcf46b8f5e3893963f46bfb15060856ea589ddd0e192d75179d`.

The CometBFT response does not advertise a finite consensus gas ceiling. Its
`-1` and the EVM header's gasLimit are different fields; the EVM field is not
proof of a finite consensus enforcement limit. These are single-endpoint RPC
observations, not independently verified consensus proofs or fleet configuration.
The earlier checked-in 100M figure must not be used as a live benchmark ceiling.
Foundation/operator confirmation of enforced application/consensus limits and
a finite isolated-test load envelope is now required. Do not change live limits
or load-test the public/mainnet endpoint to obtain this evidence.

## Required production evidence matrix

All rows below remain pending; WSL2/in-memory measurements do not satisfy them.

| Coverage | Required cases | Prerequisite/owner |
| --- | --- | --- |
| Deployment | 4, 16, 32 KiB and 64 KiB admission boundary; gas failure/recovery | Engineering fixtures; Foundation final limits |
| Persistent token state | 1k, 10k, 100k holders; equivalent allowance sets; mixed state | Engineering per-key persistence first |
| Operations | deploy, read, transfer/write, overwrite, growth, approve/allowance, mint/burn where applicable, maximum-size transitions, OOG/revert | Engineering correctness and workload harness |
| Logs | Single/multi-event transactions; small and 4 KiB payloads; maximum allowed transaction/log envelope | Engineering boundary fixtures |
| Full blocks | Multiple utilization levels through an agreed enforced ceiling; warm/cold state, durable commits, long-lived state, maximum logs and worst-case failures | Foundation isolated validator-class environment and finite test envelope |
| Measurements | p50/p95/p99 execution and durable commit time; CPU/RSS; DB reads/writes; persistent growth | Engineering instrumentation and operator runs |
| Acceptance | Execution-time and persistent-growth budgets derived from validator evidence | Foundation technical approver |

Amir supplied the recommended hardware class: 8 CPU cores / 32 GB RAM / 1 TB
NVMe / 1 Gbps, or identify the actual production validator class. This is a
review requirement, not a claim that our available WSL environment meets it.

## Next work and external inputs

Engineering can continue shared live Go/Rust fuel accounting and scalable
per-key/native-store lifecycle design without asking Amir to approve the same
KV/log rates again. Independent retesting and final source alignment remain ours
to prepare. The reviewed packet is historical evidence, not the final release pin.

Request from Foundation/operator: confirm effective live block limits and nominate
an isolated validator-class benchmark environment/operator. Agree a finite test
load envelope given the observed `max_gas=-1`; derive final time/growth budgets
from measured results rather than inventing them. No production load test, chain
configuration change, gateway activation or contract deployment is authorized.
