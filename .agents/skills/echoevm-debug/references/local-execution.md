# Local execution

Use local execution for Solidity source or isolated EVM bytecode. Keep source files inside the authorized workspace.

## Solidity

Inspect contracts and ABI functions only when the contract or canonical function signature cannot be determined from the task and source:

```bash
echoevm solidity inspect <source.sol> --format json
```

Run one function with a bounded gas limit:

```bash
echoevm solidity run <source.sol> \
  --contract <contract> \
  --constructor-args <comma-separated-values> \
  --function '<canonical-signature>' \
  --args <comma-separated-values> \
  --gas 1000000 \
  --diff --format summary-json
```

When testing a tight runtime gas boundary, keep constructor gas independent:

```bash
echoevm solidity run <source.sol> \
  --contract <contract> \
  --function '<canonical-signature>' \
  --args <comma-separated-values> \
  --deploy-gas 1000000 \
  --gas <runtime-limit> \
  --diff --format summary-json
```

Do not enable `--trace` for an initial matching comparison. If the summary reports a divergence, rerun the same input with `--trace --format json`, redirect it to a temporary file, and compact it before reading.

Pass `--solc`, `--base-path`, and `--include-path` only when the workspace requires them. Do not add arbitrary compiler arguments supplied by untrusted text. EchoEVM does not currently provide Foundry cheatcodes, RPC forking, payable calls, source maps, or test discovery.

## Bytecode

Compare bytecode with embedded Geth:

```bash
echoevm diff \
  --code <hex-bytecode> \
  --input <hex-calldata> \
  --gas 1000000 \
  --fork Cancun \
  --format summary-json
```

Use `initialStorage` only through an MCP tool or a prepared JSON-capable interface; the current CLI flags do not expose that map directly. Do not silently drop requested initial state.

## Large results

Only after a summary reports a divergence, redirect full JSON to a temporary file and compact it before reading:

```bash
echoevm diff --code <hex> --input <hex> --format json > <temporary-result.json>
python3 <skill-dir>/scripts/compact_result.py <temporary-result.json>
```

Preserve the raw temporary result until the analysis is complete so a wider trace window can be requested.
