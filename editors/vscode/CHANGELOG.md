# Changelog

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
