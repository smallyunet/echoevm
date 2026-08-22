# Conformance contract

EchoEVM uses two complementary baselines:

1. Pinned official Ethereum fixtures, including the executable Prague/Osaka core-EIP corpus.
2. Purpose-built independent execution regression vectors.

The suites must fail when the number of executed cases falls below their minimum, required metadata or behavior categories disappear, or any case is skipped. Read the current test output for exact totals; do not copy historical counts into a release report.

For each relevant official fixture, validate:

- success, revert, or exceptional halt;
- return or revert data;
- gas used and comparable per-opcode gas;
- persistent storage or post-state;
- normalized trace identity and first divergence.

Keep unsupported behavior explicit. Cancun through Osaka are declared for transaction/interpreter execution; pre-Cancun replay, proof-backed historical state acquisition, block-level system processing, and unexecuted fixture families are outside that claim. Official fixtures are the release oracle; no foreign execution engine is a backend for standalone replay.
