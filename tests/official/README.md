# Official Ethereum execution fixtures

This directory pins the official Ethereum Execution Spec Tests (EEST) mainnet
release `tests@v20.0.1`. Its `fixtures.tar.gz` contains all fixture formats and
mainnet forks through Osaka/BPO2. The 423,237,039-byte asset is not committed to
Git; `manifest.json` records its immutable URL and SHA-256 digest.

Download, verify, and atomically install the release under
`tests/official/fixtures`:

```sh
make setup-official-fixtures
```

Audit every JSON fixture file and case in the release, then execute the fixed
Prague/Osaka core-EIP corpus against EchoEVM:

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
state rather than test cases. The executable corpus currently covers 26 pinned
state-test files for Prague/Osaka core EIPs, with exact transaction and skip
counts printed by the test. It covers EIP-2537, EIP-7594, EIP-7623, EIP-7702,
EIP-7823, EIP-7825, EIP-7883, EIP-7939, and EIP-7951 with zero skipped cases.
It does not claim that EchoEVM executes all 239,839 indexed fixtures yet.
