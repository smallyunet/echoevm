# EchoEVM Roadmap

This document outlines the development roadmap for EchoEVM, a minimal Ethereum Virtual Machine implementation in Go.

**Current Version**: v0.0.18

---

## 📍 Development Phases

### Phase 1: Foundation (v0.0.1 - v0.0.6) ✅

Initial EVM implementation with core execution capabilities.

| Version | Highlights |
|---------|------------|
| v0.0.1 | Basic EVM interpreter, stack operations |
| v0.0.2 | Memory operations, arithmetic opcodes |
| v0.0.3 | Storage operations, SHA3 |
| v0.0.4 | Control flow (JUMP, JUMPI, JUMPDEST) |
| v0.0.5 | ABI encoding, contract calls |
| v0.0.6 | Interactive REPL, block execution, debug mode |

**Key Features Delivered:**
- Core opcode execution (arithmetic, bitwise, comparison)
- Stack and memory management
- Basic ABI encoding for function calls
- `deploy`, `call`, `trace`, `run` CLI commands
- Interactive REPL mode

---

### Phase 2: EVM Core Completion (v0.0.7 - v0.0.12) ✅

Expanded opcode support, EIP compliance, and testing infrastructure.

| Version | Highlights |
|---------|------------|
| v0.0.7 | Missing opcodes implementation |
| v0.0.8 | Project structure refactoring |
| v0.0.9 | Testing infrastructure simplification |
| v0.0.10 | Stability improvements, test coverage |
| v0.0.11 | Disassembly command, ABI parsing enhancements |
| v0.0.12 | EIP-1153 (Transient Storage), EIP-5656 (MCOPY) |
| v0.0.13 | Precompiled contracts (0x01-0x04), Tuple ABI support |
| v0.0.14 | Bug fixes, stability improvements |
| v0.0.15 | EIP-4844 Cancun opcodes (BLOBHASH, BLOBBASEFEE) |
| v0.0.16 | Web debugger UI stabilization, compliance coverage updates |
| v0.0.17 | Web debugger run control, origin allowlist, docs alignment |
| v0.0.18 | Merkle Patricia Trie (MPT), StateDB Integration |

**Key Features Delivered:**
- EIP-1153: TLOAD/TSTORE (Transient Storage)
- EIP-5656: MCOPY (Memory Copy)
- EIP-2929 compatible gas metering
- `disasm` command for bytecode disassembly
- Array support in ABI encoding (`uint256[]`, `address[]`, etc.)
- State journaling for snapshot/revert
- Comprehensive testing suite (unit, integration, compliance)
- Structured logging with zerolog

---

### Phase 3: Advanced Features (v0.0.13 - v0.0.18) ✅

Enhanced ABI support and fork-specific opcode behavior.

**Planned Features:**

- [x] **Tuple ABI Support** - Encode/decode struct types
- [x] **Nested Array Support** - Multi-dimensional arrays (`uint256[][]`)
- [x] **Fork-Specific Behavior** - Pre/post merge opcode differences
- [x] **Cancun Opcodes** - BLOBHASH, BLOBBASEFEE (EIP-4844)
- [x] **State Trie** - Merkle Patricia Trie implementation
- [x] **Expanded Compliance** - Increase Ethereum test suite coverage
- [x] **Precompiled Contracts** - 0x01-0x09 (ECRECOVER..BLAKE2F)

---

### Phase 4: Developer Experience (v0.0.19 - v0.0.24) 📋

Tools and integrations for enhanced developer productivity.

**Planned Features:**

- [x] Web Debugger UI - Browser-based EVM execution visualizer
- [ ] **VS Code Extension** - Inline bytecode visualization
- [ ] **Step-by-Step Debugging** - Breakpoints and watch expressions
- [ ] **Gas Profiler** - Per-opcode gas consumption analysis
- [ ] **Contract Analyzer** - Security pattern detection
- [ ] **Diff Mode** - Compare execution traces between EVMs
- [ ] **Export Formats** - Trace export to JSON, CSV, CallGraph

---

### Phase 5: Production Readiness (v0.0.25+) 📋

Full compliance and ecosystem integration.

**Planned Features:**

- [ ] **100% Test Compliance** - Pass all Ethereum GeneralStateTests
- [ ] **Performance Optimization** - Interpreter speed improvements
- [ ] **Library API** - Embeddable Go package for programmatic use
- [ ] **Plugin System** - Custom opcode handlers
- [ ] **Documentation Site** - Comprehensive API and usage docs
- [ ] **Community Guidelines** - Contributing, code of conduct

---

## 🎯 Current Focus

**v0.0.19 Priorities:**
1. VS Code Extension prototype
2. Gas Profiler
3. Further compliance refinements

---

## 📊 Feature Status Legend

| Symbol | Status |
|--------|--------|
| ✅ | Completed |
| 📋 | Planned |
| 🚧 | In Progress |
| ❌ | Blocked/Deferred |

---

## 🤝 Contributing

Have ideas for the roadmap? Open an issue or discussion on [GitHub](https://github.com/smallyunet/echoevm).

Feature requests and pull requests are welcome!
