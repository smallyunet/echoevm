# EchoEVM for VS Code

Compile, deploy, run, trace, and differentially compare a Solidity function
without starting a local JSON-RPC node.

## Two-minute start

1. Install the extension.
2. Run **EchoEVM: Setup and Diagnose Toolchain**. The extension can download the
   matching EchoEVM CLI from GitHub Releases and verifies it against the
   published `SHA256SUMS` file.
3. Run **EchoEVM: Open Getting Started Example**.
4. Choose **Run Example**, then **Show Trace**.

The EchoEVM status-bar item shows whether the CLI and compiler are ready. Click
it at any time to diagnose or repair the toolchain.

## Requirements

The extension can install a release-matched EchoEVM CLI into its private global
storage. It discovers workspace-local `node_modules/.bin/solc` and `solcjs` on
macOS and Linux, then falls back to its bundled `solc-js 0.8.30` compiler. On
Windows, npm `.cmd` launchers are skipped and the bundled compiler is used. You
can select a native compiler through the setup command or configure
`echoevm.executablePath` and `echoevm.solcPath`.

For Remote SSH, Dev Containers, and WSL, tools are resolved and managed in the
remote extension runtime rather than on the local desktop.

## Usage

1. Open a trusted workspace and a `.sol` file.
2. Run **EchoEVM: Run Solidity Function** or **EchoEVM: Run and Compare with Geth**.
3. Select a deployable contract and ABI function.
4. Enter constructor and function arguments when prompted.
5. Review the result in the EchoEVM output channel. Run **EchoEVM: Show Last Trace** for the opcode table.

## Trust and downloads

EchoEVM only downloads CLI assets from `smallyunet/echoevm` GitHub Releases and
rejects a binary unless its SHA-256 digest matches that release's checksum
manifest. The bundled compiler does not modify the workspace or global package
manager. Projects that pin another compiler can select it explicitly.

This release targets Cancun and does not provide Foundry cheatcodes, RPC
forking, payable calls, or source-level breakpoints.
