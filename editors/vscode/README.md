# EchoEVM for VS Code

Compile, deploy, run, trace, and differentially compare a Solidity function
without starting a local JSON-RPC node.

## Requirements

Install `echoevm` and `solc` in the environment where the VS Code workspace is
running. For Remote SSH, Dev Containers, and WSL, install them on the remote
host. Configure custom paths with `echoevm.executablePath` and
`echoevm.solcPath`.

## Usage

1. Open a trusted workspace and a `.sol` file.
2. Run **EchoEVM: Run Solidity Function** or **EchoEVM: Run and Compare with Geth**.
3. Select a deployable contract and ABI function.
4. Enter constructor and function arguments when prompted.
5. Review the result in the EchoEVM output channel. Run **EchoEVM: Show Last Trace** for the opcode table.

This MVP targets Cancun and does not provide Foundry cheatcodes, RPC forking,
payable calls, source-level breakpoints, or automatic compiler installation.
