# Changelog

## 1.5.1

- Align product copy with EchoEVM's exact-trace and bounded-evidence boundary.
- Ship the same frozen protocol and A-grade execution semantics on the
  responsibility-based Rust module layout.

## 1.5.0

- Align the extension with the EchoEVM 1.5.0 full Cancun-through-Osaka state
  test gate and public execution logs/state commitments.

## 1.4.0

- Keep the VS Code package aligned with the EchoEVM 1.4.0 proof-verified
  witness acquisition release.

## 1.3.0

- Keep the VS Code package aligned with the EchoEVM 1.3.0 execution engine and
  browser Contract Lens release.

## 1.2.0

- Replace the third-party EVM backend with EchoEVM's independent Rust opcode,
  gas, state-transition, call/create, transaction, and precompile implementation.
- Preserve the exact zero-skip Cancun, Prague, and Osaka official fixture gates.
- Expand the frozen EchoEVM opcode inventory from 154 to 170 names.

## 1.1.0

- Add exact Cancun, Prague, and Osaka bytecode conformance gates.
- Validate native and Wasm engines against the shared 15-vector compatibility matrix.
- Keep the frozen protocol v1 and EchoEVM CLI v1.0 minimum compatibility.

## 1.0.0

- Require the EchoEVM v1.0 Rust CLI and frozen Solidity protocol v1.
- Execute constructor deployment and function calls through the embedded Rust engine.
- Resolve the new native Rust release assets on macOS, Linux, and Windows.
- Remove legacy Go and Geth comparison framing from the active product path.

## 0.1.1

- Detect the nearest nested Foundry project for an open Solidity file instead of assuming the outermost VS Code workspace is the compiler root.
- Resolve Foundry remappings, libraries, project-local compilers, and pinned SVM compiler versions from that detected project root.
- Resolve configured include paths relative to the detected Solidity project and log the selected root for troubleshooting.

## 0.1.0

- Add Run and Compare CodeLens actions above Solidity function declarations.
- Show the latest status and gas result in the editor without replacing the source view.
- Add an Execution Evidence view for status, gas, storage, Geth comparison, key opcodes, and source navigation.
- Map runtime program counters and ABI functions back to Solidity source ranges when used with a source-aware EchoEVM CLI.
- Report concrete execution reverts and faults in VS Code Problems; no static security findings are inferred.
- Add settings to disable CodeLens or inline execution results independently.

## 0.0.6

- Auto-detect Foundry remappings, optimizer settings, optimizer runs, and `via_ir` through the EchoEVM CLI.
- Use a Foundry project's installed SVM compiler when `solc_version` or `solc` pins an available semantic version.
- Add explicit remapping, optimizer-runs, and via-IR settings for non-Foundry workspaces.
- Require EchoEVM CLI v0.0.37 and offer a guided verified update when an older CLI is detected.

## 0.0.5

- Read Standard JSON from stdin asynchronously so VS Code's Electron runtime cannot fail with `EAGAIN` on a temporarily empty non-blocking pipe.
- Add regression coverage for delayed, multi-chunk compiler input.

## 0.0.4

- Install or update EchoEVM through `smallyunet/tap/echoevm` on macOS.
- Resolve the Homebrew-installed executable by absolute path so VS Code does not depend on its GUI launch environment's `PATH`.
- Keep verified GitHub Release downloads for Linux and Windows extension runtimes.

## 0.0.3

- Download CLI binaries and checksums through GitHub's API-free latest-release URLs.
- Avoid anonymous GitHub API rate limits during first-run setup.

## 0.0.2

- Add toolchain health status and guided setup.
- Download the matching EchoEVM CLI from GitHub Releases and verify its SHA-256 checksum.
- Discover workspace-local `solc` and `solcjs` installations.
- Bundle `solc-js 0.8.30` as a no-install compiler fallback.
- Add a ready-to-run Solidity getting-started example.
- Add the EchoEVM Marketplace icon.

## 0.0.1

- Run a selected Solidity ABI function through EchoEVM.
- Compare return data, gas, storage, and opcode traces with embedded Geth.
- Display execution summaries and an on-demand trace panel.
