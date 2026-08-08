# Official Ethereum execution fixtures

This directory pins the official Ethereum Execution Spec Tests (EEST) mainnet
release `tests@v20.0.0`. Its `fixtures.tar.gz` contains all fixture formats and
mainnet forks through Osaka/BPO2. The 399,656,884-byte asset is not committed to
Git; `manifest.json` records its immutable URL and SHA-256 digest.

Download, verify, and atomically install the release under
`tests/official/fixtures`:

```sh
make setup-official-fixtures
```

Audit every JSON fixture file and case in the release:

```sh
make test-official-fixtures
```

Large CI jobs can split the audit deterministically by fixture file:

```sh
ECHOEVM_FIXTURE_SHARD=0/16 \
ECHOEVM_OFFICIAL_FIXTURES="$PWD/tests/official/fixtures" \
go test -v ./tests/official/...
```

The full-release audit verifies acquisition integrity, `.meta/index.json`, JSON
decoding, the index's declared case count, and bidirectional coverage between
indexed paths and fixture files. Engine-X `pre_alloc` JSON files are decoded and
reported separately because the official index treats them as shared auxiliary
state rather than test cases. It does not claim
that EchoEVM executes every fixture yet. Executed official cases remain in
`tests/compliance`; new raw EEST state/blockchain runners should report executed,
unsupported, and failed counts separately instead of silently skipping cases.
