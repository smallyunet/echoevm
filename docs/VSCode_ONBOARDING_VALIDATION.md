# VS Code onboarding validation

This checklist validates the product promise: a new user can install EchoEVM
and reach a Solidity opcode trace in two minutes without typing a terminal
command.

## Release gate

- GitHub Release contains five platform binaries, the VSIX, and `SHA256SUMS`.
- A clean VS Code profile shows **EchoEVM Setup** before tools are available.
- **Install or update verified EchoEVM CLI** downloads the correct platform asset and the
  installed `echoevm version --json` reports the release tag.
- A checksum mismatch fails closed and does not replace the previous binary.
- Workspace-local `node_modules/.bin/solc` or `solcjs` is detected on macOS and
  Linux; otherwise bundled `solc-js 0.8.30` compiles without modifying the user
  environment.
- Setup still allows an explicit native compiler and links to the official
  Solidity installation guide.
- **Open Getting Started Example** creates `.echoevm/Counter.sol` without
  overwriting an existing file.
- **Run Example** reaches a successful result and **Show Trace** displays at
  least one opcode.
- Remote SSH, Dev Container, Windows, macOS, and Linux paths are exercised.

## Pilot scorecard

Run the flow with 5–10 Solidity developers who did not implement it. Record
only aggregate counts; do not add extension telemetry for the pilot.

| Metric | Target |
|---|---:|
| Median install-to-first-trace time | <= 2 minutes |
| Participants reaching a trace | >= 70% |
| Participants needing a terminal command | 0 |
| Checksum or platform-selection failures | 0 |
| Actionable feedback reports | >= 3 |

For each failure, record the operating system, workspace type, compiler source,
failed step, and exact visible error. Fix repeated onboarding failures before
adding source maps, gas profiling, or debugger features.
