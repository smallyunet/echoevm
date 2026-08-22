# EchoEVM Chrome extension

The Manifest V3 extension enriches Ethereum Mainnet transaction pages on
Etherscan with standalone EchoEVM replay. The Rust execution engine is compiled
to WebAssembly and packaged inside the extension, so users do not install a CLI
and transaction execution never leaves the browser.

## Install a GitHub Release build

1. Download `echoevm-chrome-<version>.zip` from the matching EchoEVM GitHub
   Release and extract it.
2. Open `chrome://extensions`, enable **Developer mode**, choose **Load
   unpacked**, and select the extracted directory.
3. Open an `https://etherscan.io/tx/0x…` page. The EchoEVM launcher appears in
   the lower-right corner.
4. Select an `echoevm.replay-witness.v1` JSON document. The extension validates
   it, runs EchoEVM Wasm locally, verifies that its transaction hash matches the
   open Etherscan page, and renders execution and bounded causal evidence.

The ZIP is an unpacked/developer distribution. Chrome Web Store publication
and signing are separate release channels.

## Witness acquisition boundary

Etherscan's transaction page does not contain complete historical prestate.
The extension therefore accepts the same self-contained witness contract as
the CLI instead of depending on a remote executor. A trace-capable RPC may be
used explicitly as an acquisition adapter outside the extension:

```bash
echoevm witness import-debug 0xTRANSACTION --out transaction.witness.json
```

The resulting replay is independent of that RPC. See
[`docs/REPLAY_WITNESS.md`](../../docs/REPLAY_WITNESS.md) for the exact contract
and completeness boundary.

## Build and test

```bash
make test-chrome
make build-chrome
```

`build-chrome` creates both an unpacked directory under
`build/chrome-extension` and a versioned ZIP under `dist/`.
