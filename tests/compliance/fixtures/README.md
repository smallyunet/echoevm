# Curated Ethereum compliance fixtures

This directory contains a small, deterministic subset adapted from the official
Ethereum execution tests. Each fixture records the upstream repository, commit,
and source file. The cases retain the upstream transaction and bytecode while
expressing only the post-state fields that EchoEVM can currently verify.

The baseline is intentionally small. Adding a fixture requires at least one
executed case and an explicit post-state assertion; an empty fixture directory
or fixture file fails the compliance test.
