# CLI Commands

EchoEVM provides a set of CLI commands to interact with the EVM.

## Overview

| Command | Description |
|---------|-------------|
| `run` | Execute raw bytecode with optional debug tracing |
| `diff` | Compare EchoEVM and embedded Geth results and traces |
| `deploy` | Run constructor and extract runtime bytecode |
| `call` | Execute runtime bytecode with ABI encoding |
| `trace` | JSON line trace of opcode execution |
| `disasm` | Disassemble bytecode to human-readable opcodes |
| `repl` | Interactive EVM shell |
| `web` | Browser-based visual debugger |
| `version` | Display build metadata |

### Global Flags

```bash
--log-level, -L   Log level (trace|debug|info|warn|error)
--output, -o      Output format (plain|json)
--config, -c      Config file path (reserved)
--rpc-url         Ethereum RPC endpoint
```

## Command Details

### `diff`

Run the same bytecode and calldata through EchoEVM and the module's embedded
go-ethereum dependency, then compare halt class, return/revert data, gas,
observed persistent storage, and the normalized top-level opcode trace.

```bash
echoevm diff \
  --code 60026003015f5260205ff3 \
  --input 0x \
  --gas 1000000 \
  --format text

echoevm diff --code 00 --format json
echoevm diff --web --addr :8080
```

Only Cancun and isolated in-memory state are supported. The Explorer API does
not accept RPC URLs. Inputs are bounded to 24,576 bytes of bytecode, 128 KiB of
calldata, 30 million gas, and 2,000 trace steps. The web service accepts one
comparison at a time and applies a five-second request timeout.

Normalized steps use pre-op PC/opcode/gas/stack. Post-op gas and non-terminal
stack are compared when both tracers can align them at the next top-level
opcode. Terminal stack and memory are deliberately excluded rather than
reported with false precision.

### `deploy`

Run constructor bytecode and return the runtime bytecode.

```bash
# From artifact
echoevm deploy -a ./artifacts/Add.json --print

# From raw bytecode file
echoevm deploy -b ./constructor.bin --out-file runtime.bin
```

### `call`

Execute runtime bytecode.

```bash
# With ABI encoding (using artifact)
echoevm call -a ./artifacts/Add.json -f "add(uint256,uint256)" -A "2,40"

# With raw calldata and bytecode binary
echoevm call -r ./runtime.bin -d 771602f70000...
```

### `trace`

Execute with granular tracing enabled.

```bash
# Trace first 40 steps
echoevm trace -a ./artifacts/Add.json -f "add(uint256,uint256)" -A "1,2" --limit 40 | jq .

# Trace with full stack/memory state
echoevm trace -a ./artifacts/Loops.json -f "forLoop(uint256)" -A 5 --full | jq .
```

### `disasm`

Disassemble bytecode into readable opcodes.

```bash
# From hex string
echoevm disasm 6001600201

# Output:
# 0000: PUSH1 01
# 0002: PUSH1 02
# 0004: ADD
```

### `repl`

Start the interactive Read-Eval-Print Loop.

```bash
echoevm repl
> PUSH1 10 PUSH1 20 ADD
```

### `web`

Start the web debugger UI.

```bash
echoevm web --code "6001600201"
# Open http://localhost:8080 and click "Run Trace"
```
