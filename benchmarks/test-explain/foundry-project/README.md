# Foundry preparation fixture

This dependency-free project is the source fixture for the direct
`echoevm explain foundry` workflow. It exercises constructor deployment,
ABI-visible `setUp()`, storage capture, independent final-call replay, and
storage-to-output provenance.

The normal zero-dependency gate uses the compact artifacts in the sibling
`foundry/` directory. Maintainers can additionally verify the current local
Foundry output without committing generated `out/` or `cache/` directories:

```bash
forge build --root benchmarks/test-explain/foundry-project
echoevm explain foundry \
  benchmarks/test-explain/foundry-project/out/SetUpTest.sol/SetUpTest.json \
  --test 'testReadsSetup()' --expect-return 0x2a --format json
```
