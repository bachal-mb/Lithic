# Candidate dynamic value envelope

Implementation: `lithovm_bytecode::values::{Value, encode, decode}`.
This is a standalone encoding prerequisite for native string metadata. It is
not yet a v11 call ABI, compiler string lowering, persistent storage encoding,
or RPC interface. Existing bytecode and contract behavior are unchanged.

The envelope is `LVAL` followed by version byte 1 and a big-endian u16 value
count. Each value starts with a one-byte tag. Tags 1 through 5 retain the
existing scalar types and their canonical 32-byte words. Tag 6 denotes a
string: big-endian u16 byte length followed by exactly that many UTF-8 bytes.
No padding, trailing bytes, unknown tags or alternative versions are accepted.
Strings preserve embedded NUL, combining marks and all valid UTF-8 without
normalization or truncation. Empty strings are permitted.

Candidate limits are 64 values, 4096 UTF-8 bytes per string, and 65536 bytes
per complete envelope. The aggregate limit includes headers. Encoding validates
limits before allocating output. Decoding checks total size before parsing and
checks count/string bounds before allocation. These are implementation limits
for review, not a consensus-approved schedule or Solidity compatibility claim.

Independent example: two strings `Aé` and empty encode as
`4c56414c01000206000341c3a9060000`.

Tests cover this vector, exact round trips, Unicode preservation, maximum string
length, total/count limits, malformed/truncated payloads, invalid UTF-8, trailing
data and noncanonical scalar words. The existing bytecode-decoder fuzz target
also checks canonical re-encoding of every accepted value envelope.

Remaining work before Amir's token factory can use strings: introduce a
versioned executable string type; carry dynamic values through parameters,
locals, storage, returns and events; charge deterministic byte-dependent gas;
add compiler string parsing/lowering and rollback tests; connect the host and
RPC. The envelope alone does not close string support or factory deployment.
