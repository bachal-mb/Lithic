# LithoVM artifact and source verification v1

Status: compiler and schema candidate; no LithoScan worker or chain deployment

`lithc --emit lithovm` emits a versioned JSON artifact containing the exact
compiler version, target, bytecode version, raw-source Keccak-256, bytecode
Keccak-256, ABI, canonical entrypoints and bytecode. JSON field order is not a
consensus property; the hash values and bytecode are.

The source hash is computed over the exact UTF-8 bytes supplied to the
compiler. Verification clients must not normalize line endings, Unicode or
trailing whitespace. Each selector is the full 32-byte Keccak-256 digest of:

```text
function_name(type_name,...)
```

For example, `transfer(address,u256)` is
`0xf61367304e4e32065cad538b10a44bb599f78e0771530108572d225c74122c1f`.

## Schemas

- `sdk/schemas/lithovm.artifact.schema.json` describes compiler output.
- `sdk/schemas/lithovm.verification.schema.json` packages exact source bytes,
  artifact, chain ID, canonical 20-byte address and deployed code hash.
- `sdk/schemas/lithovm.deployment-status.schema.json` is the shared
  litho.finance/LithoScan status envelope.

Sources must be sorted by path before submission. Native v11 currently accepts
one source, and its source hash must equal both the source entry hash and the
artifact `sourceHash`. A verification worker must select the exact released
compiler version, compile the exact content, validate the complete artifact,
and accept only when rebuilt bytecode and `codeHash` equal chain state. Merely
matching an ABI or contract name is insufficient.

The lifecycle is monotonic:

```text
prepared -> submitted -> included -> verified
                         |           |
                         v           v
                       failed   verification_failed
```

`failed` records transaction/deployment failure. `verification_failed` means a
contract was included but its submitted source package did not reproduce the
deployed code. Initializer success is part of atomic inclusion, not a separate
state.

The schemas do not define wallet authorization, broadcast, finality depth,
retry policy or database ownership. Those remain frontend, chain RPC and
LithoScan integration gates. Nothing in this interface authorizes deployment.
