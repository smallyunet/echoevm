# EchoEVM Documentation

Central index for EchoEVM reference, guides, and examples.

## 📚 Index

### Core
- [Main README](../README.md) – Overview, features, CLI, roadmap
- [Configuration Guide](CONFIGURATION.md) – Environment variables & defaults
- [Logging Guide](LOGGING_GUIDE.md) – Levels, formats, structured fields

### Testing
- [Testing Quick Start](TESTING_QUICK.md) – Make targets & script usage
- [Test Directory Overview](../test/README.md) – Integration test layout

### CLI Usage (Current Commands)
- `deploy` – Run constructor, extract runtime
- `call` – Execute runtime bytecode with ABI encoding
- `trace` – JSON line trace of opcode execution

### Planned / Roadmap
- `disasm` – Human readable disassembly
- `block` / `range` – Block replay for analysis

## 🚀 Quick Start Snippets

Deploy (print runtime):
```bash
echoevm deploy -a ./test/contract/artifacts/contracts/01-data-types/Add.sol/Add.json --print
```

Call (ABI encode add):
```bash
echoevm call -a ./test/contract/artifacts/contracts/01-data-types/Add.sol/Add.json -f add(uint256,uint256) -A 7,11
```

Trace (limit steps):
```bash
echoevm trace -a ./test/contract/artifacts/contracts/01-data-types/Add.sol/Add.json -f add(uint256,uint256) -A 1,2 --limit 25 | jq .
```

Trace with pre/post states:
```bash
echoevm trace -a ./test/contract/artifacts/contracts/03-control-flow/Loops.sol/Loops.json -f forLoop(uint256) -A 5 --full | head -n 40
```

## ⚙ Configuration & Logging
See [Configuration Guide](CONFIGURATION.md) and [Logging Guide](LOGGING_GUIDE.md). Examples:
```bash
export ECHOEVM_LOG_LEVEL=debug
echoevm call -a ./test/contract/artifacts/contracts/01-data-types/Fact.sol/Fact.json -f fact(uint256) -A 5

export ECHOEVM_LOG_LEVEL=trace
echoevm trace -a ./test/contract/artifacts/contracts/01-data-types/Add.sol/Add.json -f add(uint256,uint256) -A 1,2 --limit 10
```

## 🧪 Testing
Fast path:
```bash
./test/test.sh          # all integration tests
./test/test.sh --binary # only binary tests
make test-unit          # Go unit tests
```
More detail: [Testing Quick Start](TESTING_QUICK.md).

## � Contribution Guidelines (Docs)
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
