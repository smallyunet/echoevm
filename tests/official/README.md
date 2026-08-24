# Official Ethereum execution fixtures

This directory pins the official Ethereum Execution Spec Tests release
`tests@v20.0.1`. `manifest.json` records the immutable 423,237,039-byte archive,
URL, and SHA-256 digest; generated fixtures are not committed.

```bash
make setup-official-fixtures
make test-official-fixtures
```

The Rust gate recursively executes every state-test fixture in the release's
`for_cancun`, `for_prague`, and `for_osaka` directories under its matching
declared ruleset. It validates canonical signed transaction bytes and recovered
senders, accepted post-state accounts and roots, receipt status and gas, ordered
logs commitments, and normalized rejection categories. The wrapper asserts all
three exact release summaries:

```text
files=2337 transactions=11554 accepted=10968 rejected=586 fork=Cancun skipped=0
files=2471 transactions=13851 accepted=13063 rejected=788 fork=Prague skipped=0
files=2408 transactions=14516 accepted=13708 rejected=808 fork=Osaka skipped=0
```

No allowlist, per-case skip, or Go/client differential output participates in
this semantic gate. The claim is limited to this pinned corpus.
