# EchoEVM Documentation

Central index for EchoEVM reference, guides, and examples.

**Current Version**: v0.0.17

## 📚 Index

### Core
- [Main README](../README.md) – Overview, features, CLI, roadmap
- [Roadmap](../ROADMAP.md) – Development phases and version history
- [Configuration Guide](guides/configuration.md) – Environment variables & defaults
- [Logging Guide](guides/logging.md) – Levels, formats, structured fields

### Testing
- [Testing Guide](guides/testing.md) – Make targets & testing guide

### CLI Commands

| Command | Description |
|---------|-------------|
| `run` | Execute raw bytecode with optional debug tracing |
| `deploy` | Run constructor, extract runtime |
| `call` | Execute runtime bytecode with ABI encoding |
| `trace` | JSON line trace of opcode execution |
| `disasm` | Human readable disassembly |
| `repl` | Interactive EVM shell |
| `web` | Browser-based visual debugger |
| `version` | Display build metadata |

## 🚀 Quick Start Snippets

Run bytecode:
```bash
echoevm run 6001600201
echoevm run --debug 6001600201
```

Deploy (print runtime):
```bash
echoevm deploy -a ./artifacts/Add.json --print
```

Disassemble:
```bash
echoevm disasm 6001600201
echoevm disasm -a ./artifacts/Add.json --runtime
```

Call (ABI encode add):
```bash
echoevm call -a ./artifacts/Add.json -f add(uint256,uint256) -A 7,11
```

Trace (limit steps):
```bash
echoevm trace -a ./artifacts/Add.json -f add(uint256,uint256) -A 1,2 --limit 25 | jq .
```

## ⚙ Configuration & Logging

See [Configuration Guide](guides/configuration.md) and [Logging Guide](guides/logging.md). Examples:

```bash
export ECHOEVM_LOG_LEVEL=debug
echoevm call -a ./artifacts/Fact.json -f fact(uint256) -A 5

export ECHOEVM_LOG_LEVEL=trace
echoevm trace -a ./artifacts/Add.json -f add(uint256,uint256) -A 1,2 --limit 10
```

## 🧪 Testing

```bash
make test        # Run all tests
make test-unit   # Go unit tests only
make coverage    # Generate coverage report
make setup-tests # Initialize test fixtures submodule
```

More detail: [Testing Guide](guides/testing.md).

## 🗺 Roadmap

See [ROADMAP.md](../ROADMAP.md) for:
- Completed features by version (v0.0.1 - v0.0.17)
- Upcoming features (Phase 3-5)
- Development timeline

## 📝 Contribution Guidelines (Docs)

When editing docs:
1. Keep examples executable (copy/paste friendly)
2. Update cross-links if filenames move
3. Prefer present-tense, imperative style
4. Include context (what + why) for complex snippets
5. Validate new commands locally before publishing

## 📝 Style

- English, concise, technical
- Use fenced code blocks with language hints
- Avoid duplicating large code – link instead
- Prefer relative links within repo

---
Additions welcome. Open a PR if you introduce a new top-level document so it can be linked here.
