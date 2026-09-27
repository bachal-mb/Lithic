# Makalu limits capture and isolated Lithic benchmark nomination

Status 2026-09-28: **proposed VPS role, not provisioned; live-node capture pending
operator access.** This document does not authorize chain configuration changes,
gateway registration/activation, deployment or public load tests.

## What was checked read-only

- Repo inventory identifies Makalu testnet (`chain_id=700777`) primary validator
  `lithod-mtest-val-02` on the shared `srv02` host, home
  `/var/lib/litho-mtest-val-02`, CometBFT RPC port 26757 and EVM RPC port 8645.
  These are inventory claims, **not observed running configuration**.
- The [Foundation network-access page](https://access.litho.ai/network) lists
  `https://rpc.litho.ai` for Makalu testnet (EVM chain 700777).
  Public `https://rpc.litho.ai/status` and `/consensus_params` timed out from
  this workspace. The documented public front `vps1` was reachable by SSH;
  its live nginx routes Makalu to `31.97.39.138:26857` with backup
  `72.60.177.106:26857`. Both upstream read-only requests timed out from vps1.
- SSH from this workspace and from vps1 to the documented primary host was
  refused for the available public key. SSH to the backup host worked, but
  `lithod-makalu-sentry` was inactive. No validator key, app secret, full TOML,
  process environment or private network payload was read.
- Repo Ansible declares `consensus_max_gas: 100000000`; that value cannot be
  represented as live Makalu truth without a running-node capture. This is
  distinct from the 9005 mainnet's height-pinned observation in the gas review.

## Operator-side capture (read-only)

The operator of the **currently running** Makalu validator should first confirm
its actual service name, home and local ports. On that host, run
[`scripts/capture-makalu-effective-limits.sh`](../scripts/capture-makalu-effective-limits.sh)
with those parameters. It prints only service state, binary path/hash/version,
hashes of the config/genesis files, public consensus and EVM block limits, and
selected non-secret app options. It does **not** dump validator/node keys or full
configuration/environment. If `jq` is unavailable, use an operator-approved
equivalent read-only capture; do not install packages or restart the validator
for this request.

Example for the documented service (operator must verify before running):

```sh
bash scripts/capture-makalu-effective-limits.sh \
  lithod-mtest-val-02 /var/lib/litho-mtest-val-02 \
  http://127.0.0.1:26757 http://127.0.0.1:8645
```

If the documented host is no longer active, capture from the real active node
and name its service/path in the evidence. Share the command output and timestamp,
not credentials or key files. Foundation should confirm any enforced app/block
limit not exposed by `consensus_params` or the EVM header. No public traffic is
required for this capture.

## Nominated fresh VPS: `lithic-makalu-bench-01`

Requested role: new dedicated **8 vCPU / 32 GiB RAM / 500 GB NVMe** VPS, billed
and owned by Lithosphere Foundation. This is a procurement nomination, not an
existing host assignment or purchase request. Do **not** repurpose Makalu's
bonded validator, public sentries, explorer, mainnet nodes or MultX equipment.

The Foundation operator should choose the provider/region and return the private
host identifier, CPU model, actual usable RAM/NVMe size, filesystem/DB backend,
OS/kernel, and assigned benchmark operator. Provisioning/payment requires their
normal authorization; no provider SKU, IP or account access has been assumed.
No secrets should be sent in chat.

Acceptance before measurements:

1. Verify the running Makalu binary SHA-256; install that exact binary on the
   isolated VPS, not merely a similarly named release. Record both hashes.
2. Reproduce the execution-relevant Makalu genesis/consensus, app/EVM, gas and
   pruning/storage configuration. Record source/destination hashes and a
   redacted comparison of effective settings. Network identity, local ports,
   peer lists and signing settings **must differ** to preserve isolation; this
   exception is not an execution-config drift waiver.
3. Never copy `priv_validator_key.json`, `priv_validator_state.json`, `node_key.json`,
   wallet keys, KMS config or seed phrases. No live peers, signing duties,
   public RPC, faucet/bridge/MultX traffic or funded transactions. Private
   access for approved operators only. Use synthetic/replayed workloads and
   durable local state on the NVMe disk.
4. Run the Amir workload matrix in
   [his recorded review](AMIR_GAS_REVIEW_RESPONSE.md): 4/16/32/64 KiB deployment;
   1k/10k/100k holder and allowance states (after scalable persistence);
   read/write/transfer/approval/mint/burn, OOG/revert, max logs and full blocks.
   Measure p50/p95/p99 execution and commit time, CPU/RSS, DB I/O and persistent
   growth at multiple utilization levels. Do not assume a finite 100M Makalu
   ceiling from Ansible; obtain the **live** effective limit and agree a finite
   isolated-test envelope with Foundation first.

This nominated VPS is **not yet an available benchmark resource**. The current
WSL2/in-memory results do not satisfy the validator-class acceptance gate.
