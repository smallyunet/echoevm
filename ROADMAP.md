# EchoEVM Roadmap

This document outlines the development roadmap for EchoEVM, a minimal Ethereum Virtual Machine implementation in Go.

**Current Version**: v0.0.45

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
| v0.0.19 | MVP reliability: unified execution semantics, restored Web CLI, Trie prefix fix |
| v0.0.20 | Transaction correctness, top-level precompiles, pinned compliance baseline |
| v0.0.21 | Geth differential conformance, visible CI reports, complete BLAKE2F |
| v0.0.22 | RPC-backed transaction replay, Etherscan input, nested call-frame tracing |
| v0.0.23 | Lint-clean immutable patch release for transaction replay |
| v0.0.24 | Cache-safe Explorer assets for reliable transaction replay controls |
| v0.0.25 | Ethereum Mainnet-only transaction recognition and RPC validation |
| v0.0.26 | Trace-aware readiness, credential-safe atomic deployment bundles, and typed replay failures |
| v0.0.27 | Correct SSTORE gas replay, normalized Geth opcodes, recent-transaction shortcuts, and a light Explorer UI |
| v0.0.28 | Reliable Caddy container readiness using its local administration endpoint |
| v0.0.29 | Reproducible production version metadata from the release tag |
| v0.0.30 | Comparable per-op gas diagnostics without nested-call false positives |
| v0.0.31 | STATICCALL write protection and Geth-matched nested call/create rollback and gas semantics |
| v0.0.32 | Solidity source runner and VS Code extension MVP |
| v0.0.33 | Correct VS Code Marketplace publisher identity |
| v0.0.34 | Verified CLI release assets and zero-terminal VS Code onboarding |
| v0.0.35 | Portable Codex, Gemini CLI, and Claude Code Skills with bounded trace evidence |
| v0.0.36 | Reliable bundled solc-js stdin handling in VS Code's Electron runtime |
| v0.0.37 | Foundry remappings, optimizer/via-IR settings, and pinned SVM compiler discovery |
| v0.0.38 | Agent-summary JSON, separate deployment/call gas limits, and token-efficient EchoEVM skill routing |
| v0.0.39 | Explainable opcode-event protocol, top-level rollback conformance, and auditable trace-value benchmark |
| v0.0.40 | Compact causal evidence profiles with benchmarked long-trace token savings |
| v0.0.41 | Solidity source-run causal evidence, nested frame/value links, and a formal compiled-Solidity benchmark |
| v0.0.42 | Fork conformance baseline and official EEST fixture audit infrastructure |
| v0.0.43 | CI-clean compatibility release |
| v0.0.44 | Source-aware Solidity inspection and runtime PC mapping for editor execution evidence |
| v0.0.45 | Mainnet replay causal evidence and shareable transaction explanation flow |

**Key Features Delivered:**
- EIP-1153: TLOAD/TSTORE (Transient Storage)
- EIP-5656: MCOPY (Memory Copy)
- EIP-2929 compatible gas metering
- `disasm` command for bytecode disassembly
- Array support in ABI encoding (`uint256[]`, `address[]`, etc.)
- State journaling for snapshot/revert
- Testing suite covering unit, integration, E2E, and a curated compliance baseline
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
- [x] **Compliance Baseline** - Run pinned official Ethereum vectors without optional downloads
- [x] **Complete Precompiled Contracts** - 0x01-0x09 execute, including EIP-152 BLAKE2F

---

### Phase 4: Developer Experience (v0.0.19 - v0.0.24) 📋

Tools and integrations for enhanced developer productivity.

**Planned Features:**

- [x] Web Debugger UI - Browser-based EVM execution visualizer
- [x] **Solidity Source Runner** - Compile, deploy, call, trace, and differentially compare one contract function
- [x] **VS Code Extension MVP** - Inspect, run, compare, and display opcode traces from `.sol` files
- [x] **Source-Aware VS Code Evidence** - Function CodeLens, inline execution results, Problems diagnostics, evidence tree, and PC-to-source navigation
- [x] **Agent Skills** - Debug contracts and transactions or validate conformance from Codex, Gemini CLI, and Claude Code
- [ ] **Source-Level Debug Controls** - Breakpoints and step controls remain deferred; source-mapped execution evidence is available
- [ ] **Step-by-Step Human Debugging** - Deferred; bounded agent trace windows take priority
- [ ] **Gas Profiler** - Deferred in favor of per-opcode semantic gas explanations
- [ ] **Contract Analyzer** - Deferred; EchoEVM provides execution evidence, not scanner claims
- [x] **Differential Explorer** - Reusable Cancun EchoEVM/Geth engine, CLI, JSON API, and local trace UI
- [x] **Transaction Replay** - Hash/Etherscan input, RPC prestate hydration, and full call-frame trace comparison
- [x] **Replay Causal Evidence** - Question-routed, bounded transaction evidence with comparison confidence and shareable Explorer URLs
- [ ] **Export Formats** - Trace export to JSON, CSV, CallGraph

---

### Phase 5: Production Readiness (v0.0.25+) 📋

Full compliance and ecosystem integration.

**Planned Features:**

- [ ] **100% Test Compliance** - Pass all Ethereum GeneralStateTests
- [ ] **Performance Optimization** - Interpreter speed improvements
- [ ] **Library API** - Embeddable Go package for programmatic use
- [ ] **Plugin System** - Custom opcode handlers
- [x] **Unified Hosted Demo** - Use `https://r.dark20.xyz/` as the single public web entry point
- [ ] **Community Guidelines** - Contributing, code of conduct

---

## 🎯 Current Focus

**Next Release Priorities:**
1. Freeze reproducible public-transaction witnesses for a Mainnet evidence benchmark
2. Validate real-transaction diagnosis accuracy and fresh-token use against broad traces
3. Add richer account, log, return-data, and semantic dynamic-gas evidence when benchmark cases require it
4. Expand the compiled-Solidity benchmark across contracts, compiler settings, and held-out failure classes
5. Preserve `echoevm.trace.v1` as the full diagnostic contract behind bounded evidence

The product is AI-first. The VS Code extension and hosted Explorer remain useful
demonstration and inspection surfaces, but human onboarding, CodeLens, source-level
breakpoints, and broad debugger UI expansion no longer gate the roadmap.
Embedded Geth stays in conformance tests and optional comparison commands; the
primary product path explains EchoEVM execution without requiring a comparison.

The replay engine intentionally requires a trace-capable RPC and does not
approximate transaction prestate from the parent block. Cancun remains the only
fully declared EchoEVM ruleset; other fork eras are labeled instead of silently
claiming exact compatibility.

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
