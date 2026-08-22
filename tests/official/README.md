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

Audit every JSON fixture file and case in the release, then execute every
official state fixture authored for Prague or Osaka under current Osaka rules:

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
state rather than test cases. The executable current-mainnet corpus covers 187
Prague- and Osaka-authored state-test files and 3,461 cases in the pinned
release: 3,244 accepted transactions, 217 consensus-invalid rejections, and
zero skipped cases. It covers EIP-7702 alongside EIP-7594, EIP-7823, EIP-7825,
EIP-7883, EIP-7939, and EIP-7951.
It does not claim that EchoEVM executes all 239,839 indexed fixtures yet.
