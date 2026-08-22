# Official Ethereum execution fixtures

This directory pins the official Ethereum Execution Spec Tests release
`tests@v20.0.1`. `manifest.json` records the immutable 423,237,039-byte archive,
URL, and SHA-256 digest; generated fixtures are not committed.

```bash
make setup-official-fixtures
make test-official-fixtures
```

The Rust gate executes every Osaka post-state case in the Prague- and
Osaka-authored state-test suites. It compares accepted post-state accounts,
balances, nonce, code, and storage exactly, and requires consensus-invalid
transactions to be rejected. The wrapper asserts the exact release summary:

```text
files=187 transactions=3461 accepted=3244 rejected=217 fork=Osaka skipped=0
```

No allowlist, per-case skip, or Go/client differential output participates in
this semantic gate. The claim is limited to this pinned corpus.
