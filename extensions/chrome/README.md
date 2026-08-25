# EchoEVM Chrome extension

The Manifest V3 extension enriches contract and Ethereum Mainnet
transaction pages on Etherscan. The Rust execution engine is compiled to
WebAssembly and packaged inside the extension, so users do not install a CLI
and execution never leaves the browser.

## Install a GitHub Release build

1. Download `echoevm-chrome-<version>.zip` from the matching EchoEVM GitHub
   Release and extract it.
2. Open `chrome://extensions`, enable **Developer mode**, choose **Load
   unpacked**, and select the extracted directory.
3. Open a `https://etherscan.io/address/0x…#code` page. Behavior Lens reads the
   deployed bytecode already rendered by Etherscan and automatically infers
   selectors, reachable effects, value origins, and coverage locally. When a
   verified ABI is present it labels inferred selectors and lists `pure`
   functions that can run in EchoEVM's local empty-state sandbox.
4. Or open an `https://etherscan.io/tx/0x…` page and select an
   `echoevm.replay-witness.v1` JSON document. The extension validates
   it, runs EchoEVM Wasm locally, verifies that its transaction hash matches the
   open Etherscan page, and renders execution and bounded evidence selected
   from the complete trace.

The ZIP is an unpacked/developer distribution. Chrome Web Store publication
and signing remain separate release channels.

## Behavior Lens boundary

Behavior Lens uses deployed bytecode displayed on the current Etherscan page.
It does not download or execute remote JavaScript and does not recompile
displayed Solidity source. The optional verified ABI supplies labels only.

Behavior inference uses bounded abstract execution and emits
`echoevm.behavior.v1`. It is not a concrete Mainnet simulation, decompilation,
security audit, or formal proof. Dynamic jumps, fallback dispatch, and unknown
value origins remain explicit coverage limits. See
[`docs/BEHAVIOR_ABI.md`](../../docs/BEHAVIOR_ABI.md).

Only ABI functions marked `pure` are runnable in this release. They execute with
empty storage, zero call value, and no external contract state. Proxy execution
is disabled because the implementation bytecode and proxy storage context are
different inputs. The result is a local sandbox trace, not a Mainnet simulation,
security audit, or formal proof. Stateful execution requires an explicit
self-contained witness.

## Transaction witness boundary

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
