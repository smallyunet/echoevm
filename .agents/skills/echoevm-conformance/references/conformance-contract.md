# Conformance contract

EchoEVM uses two complementary baselines:

1. Pinned official Ethereum fixtures, including the executable Prague/Osaka core-EIP corpus.
2. Purpose-built independent execution regression vectors.

The suites must fail when the number of executed cases falls below their minimum, required metadata or behavior categories disappear, or any case is skipped. Read the current test output for exact totals; do not copy historical counts into a release report.

For each relevant official fixture, validate the fields supplied by the pinned state-test contract:

- canonical signed transaction bytes and recovered sender;
- exact accepted or normalized rejected category;
- receipt status and cumulative gas;
- ordered logs commitment;
- post-state accounts and state root.

Keep unsupported behavior explicit. Cancun through Osaka are declared for transaction/interpreter execution and the accepted single-block fixture gate. Pre-Cancun rules, rejected or multi-block blockchain fixtures, consensus and networking, and unexecuted fixture families are outside that claim. Proof-backed witness acquisition derives later transaction positions by proving parent state and locally replaying the preceding block prefix. Official fixtures are the release oracle; no foreign execution engine is a backend for standalone replay or block execution.
