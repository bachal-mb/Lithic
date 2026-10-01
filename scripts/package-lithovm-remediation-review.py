#!/usr/bin/env python3
"""Build a deterministic, source-pinned Lithic/LithoVM remediation review ZIP.

Only committed Git trees are archived. The known upstream Evmos scripts/.env
is excluded; no working-tree content, credentials or binaries are included.
This is an audit input, not a signed release or production approval.
"""

import argparse
import hashlib
import json
from pathlib import Path, PurePosixPath
import re
import subprocess
import tarfile
import zipfile


DATE_TIME = (2026, 10, 1, 0, 0, 0)
SOURCES = ("lithic", "evmos", "sdk")
KNOWN_EXCLUSION = ("evmos", "scripts/.env")
SENSITIVE_PATH = re.compile(r"(^|/)(\.env(?:$|\.)|id_rsa$|priv_validator_key\.json$|node_key\.json$)", re.I)


def git(repository: Path, *args: str) -> str:
    return subprocess.check_output(["git", "-C", str(repository), *args], text=True).strip()


def resolve_commits(repositories: dict[str, Path], refs: dict[str, str]) -> dict[str, str]:
    commits = {}
    for name in SOURCES:
        commit = git(repositories[name], "rev-parse", "--verify", f"{refs[name]}^{{commit}}")
        if not re.fullmatch(r"[0-9a-f]{40}", commit):
            raise RuntimeError(f"invalid {name} source commit")
        commits[name] = commit
    return commits


def archived_files(repository: Path, commit: str):
    process = subprocess.Popen(
        ["git", "-C", str(repository), "archive", "--format=tar", commit],
        stdout=subprocess.PIPE, stderr=subprocess.PIPE,
    )
    try:
        with tarfile.open(fileobj=process.stdout, mode="r|") as archive:
            for member in archive:
                if not member.isfile():
                    continue
                data = archive.extractfile(member).read()
                yield member.name.replace("\\", "/"), data
        if process.wait() != 0:
            raise RuntimeError(process.stderr.read().decode(errors="replace"))
    finally:
        if process.poll() is None:
            process.kill()
            process.wait()
        process.stdout.close()
        process.stderr.close()


def review_readme(commits: dict[str, str]) -> bytes:
    return f"""# Lithic/LithoVM LTH-R1 remediation source snapshot

Prepared 2026-10-01 for independent **retest** of the disabled candidate.
This is not a security sign-off, signed release, Makalu registration/activation,
or permission to deploy LAX or production contracts. MultX is out of scope.

Exact source commits (not mutable PR branch names):

- Lithic compiler/runtime and native-chain module: `{commits['lithic']}`.
- Evmos disabled chain lab: `{commits['evmos']}`; based on public upstream
  `eca13ef2521a9ef13c32e80b1b147230bdb155b5`.
- Cosmos SDK lab dependency: `{commits['sdk']}`.

`source/lithic`, `source/evmos` and `source/sdk` are Git-archive snapshots of
those commits. The upstream Evmos `scripts/.env` is intentionally excluded;
no working-tree modifications or binaries are included. All ZIP names use
forward slashes. Check `CHECKSUMS.sha256` and `SOURCE_COMMITS.json` before
review. The Lithic and Evmos source may be compared with the draft PRs, but
the listed commits, not current PR heads, define this snapshot.

Review LTH-01 through LTH-09 against the included
`source/lithic/docs/LTH_R1_REMEDIATION.md`. Local regression evidence covers
caller-bound salts, mandatory atomic initialization, >10M gas clamp,
FFI mutation guards, shared live fuel, scalable per-key state, and the
disabled native-store upgrade rehearsal. Reproduce Rust workspace tests and
tagged/ordinary Go command, app and keeper suites using the included source.
The build tag `lithovm_chain_lab` installs an **active lab-only gateway**;
never use that binary on a public network. The SDK snapshot is the pinned
commit, not the locally dirty SDK working tree.

Open gates: all findings await independent auditor retest; growth pricing
and the 10M cap await Foundation policy; the admitted 16/32/64-KiB contracts
cannot deploy under that provisional cap; full-block validator evidence,
networked store-upgrade rollback, Makalu deploy/call/failure/recovery,
one-pin release/overlay alignment and signed release evidence remain pending.
The isolated full-block workload requires the separate Foundation decision
in `source/lithic/docs/ISOLATED_FULL_BLOCK_BENCHMARK_APPROVAL.md`.

Please return finding-by-finding reproduction and disposition, exact source
commit pins, residual risks, and whether a further retest is required after
any policy or code changes. Do not interpret this review snapshot as an
activation or deployment request.
""".encode("utf-8")


def zip_info(name: str) -> zipfile.ZipInfo:
    info = zipfile.ZipInfo(name, DATE_TIME)
    info.compress_type = zipfile.ZIP_DEFLATED
    info.external_attr = 0o100644 << 16
    info.create_system = 3
    return info


def package(output: Path, repositories: dict[str, Path], refs: dict[str, str]) -> dict:
    if output.exists():
        raise RuntimeError(f"refusing to overwrite {output}")
    commits = resolve_commits(repositories, refs)
    entries = {"README.md": review_readme(commits)}
    omitted = []
    for source in SOURCES:
        for name, data in archived_files(repositories[source], commits[source]):
            path = PurePosixPath(name)
            if path.is_absolute() or ".." in path.parts or "\\" in name:
                raise RuntimeError(f"unsafe archived path: {name}")
            if (source, name) == KNOWN_EXCLUSION:
                omitted.append(f"{source}/{name}")
                continue
            if SENSITIVE_PATH.search(name):
                raise RuntimeError(f"unexpected sensitive tracked path: {source}/{name}")
            key = f"source/{source}/{name}"
            if key in entries:
                raise RuntimeError(f"duplicate archive entry: {key}")
            entries[key] = data
    if omitted != ["evmos/scripts/.env"]:
        raise RuntimeError(f"known Evmos exclusion missing or changed: {omitted}")
    entries["SOURCE_COMMITS.json"] = (json.dumps({
        "purpose": "independent disabled-candidate remediation retest",
        "commits": commits,
        "omitted_tracked_paths": omitted,
        "production_approval": False,
    }, indent=2, sort_keys=True) + "\n").encode()
    checksums = "".join(
        f"{hashlib.sha256(data).hexdigest()}  {name}\n"
        for name, data in sorted(entries.items())
    ).encode()
    entries["CHECKSUMS.sha256"] = checksums
    with zipfile.ZipFile(output, "x", compression=zipfile.ZIP_DEFLATED, compresslevel=9,
                         strict_timestamps=True) as archive:
        for name, data in sorted(entries.items()):
            archive.writestr(zip_info(name), data)
    return {"output": str(output), "sha256": hashlib.sha256(output.read_bytes()).hexdigest(),
            "entries": len(entries), "commits": commits, "omitted": omitted}


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, required=True)
    for source in SOURCES:
        parser.add_argument(f"--{source}-repo", type=Path, required=True)
        parser.add_argument(f"--{source}-ref", required=True)
    args = parser.parse_args()
    repositories = {source: getattr(args, f"{source}_repo") for source in SOURCES}
    refs = {source: getattr(args, f"{source}_ref") for source in SOURCES}
    print(json.dumps(package(args.output, repositories, refs), sort_keys=True))


if __name__ == "__main__":
    main()
