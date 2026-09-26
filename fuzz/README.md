# Lithic/LithoVM fuzzing

The fuzz package is an independent workspace so nightly-only sanitizer tooling
does not enter production dependency resolution. It covers two security seams:

- `bytecode_decode`: arbitrary untrusted bytecode must never panic; every
  accepted program must re-encode and decode canonically.
- `compiler_runtime`: arbitrary UTF-8 source must never panic; every accepted
  source must compile deterministically and load in the native runtime.

Run on Linux or macOS with a nightly Rust toolchain, a C++ compiler and
`cargo-fuzz 0.13.2`:

```sh
cargo install cargo-fuzz --version 0.13.2 --locked
cargo +nightly fuzz run bytecode_decode --fuzz-dir fuzz -- -max_len=65536
cargo +nightly fuzz run compiler_runtime --fuzz-dir fuzz -- -max_len=65536
```

CI performs bounded smoke campaigns. Production release evidence requires a
long-running campaign with the retained corpus and any minimized regressions;
a passing smoke job is not a security review.
