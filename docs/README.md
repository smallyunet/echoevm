# EchoEVM documentation

Start with the workflow you need, then use the protocol documents to understand
exactly what EchoEVM observes, derives, and does not claim.

| Goal | Document |
|---|---|
| Replay a transaction from complete historical state | [Replay witnesses](REPLAY_WITNESS.md) |
| Integrate trace or evidence output | [Trace protocol](TRACE_PROTOCOL.md) |
| Understand supported bytecode semantics | [Bytecode compatibility](BYTECODE_COMPATIBILITY.md) |
| Audit fixture counts and release-grade criteria | [Conformance contract](CONFORMANCE.md) |
| Understand crate and module responsibilities | [Architecture](ARCHITECTURE.md) |
| Build against the frozen wire contract | [Protocol v1](../protocol/v1/README.md) |
| Embed EchoEVM in VS Code | [VS Code onboarding validation](VSCode_ONBOARDING_VALIDATION.md) |

## Recommended path

1. Run the [README quick start](../README.md#quick-start).
2. Open the [static evidence playground](https://smallyunet.github.io/echoevm/).
3. Choose the protocol or compatibility document for your integration.
4. Reproduce the committed case locally before relying on an explanation.

## Trust boundary

EchoEVM performs execution in its Rust engine. RPC and Geth adapters may acquire
inputs or provide independent comparison data, but they are not the execution
result or semantic oracle. See [replay witnesses](REPLAY_WITNESS.md) for the
proof-backed and trace-backed acquisition boundaries.

Exact traces are primary execution output. Bounded evidence is selected from a
completed trace for presentation and diagnosis; it is not a formal proof,
security finding, or inferred causal graph.
