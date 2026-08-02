# Conformance contract

EchoEVM uses two complementary baselines:

1. Pinned official Ethereum Cancun fixtures.
2. Purpose-built differential vectors executed by EchoEVM and embedded Geth.

The suites must fail when the number of executed cases falls below their minimum, required metadata or behavior categories disappear, or any case is skipped. Read the current test output for exact totals; do not copy historical counts into a release report.

For each relevant differential vector, compare:

- success, revert, or exceptional halt;
- return or revert data;
- gas used and comparable per-opcode gas;
- persistent storage or post-state;
- normalized trace identity and first divergence.

Keep unsupported fork behavior explicit. Cancun is the declared execution ruleset; a warning for another fork is not proof that those later or earlier semantics were executed correctly.
