import assert from "node:assert/strict";
import test from "node:test";
import {
  buildInspectArguments,
  buildRunArguments,
  parseProtocolOutput,
  type InspectResult,
  type RunResult,
} from "../src/protocol";

test("buildInspectArguments keeps paths as individual process arguments", () => {
  const args = buildInspectArguments({
    source: "/workspace/My Contract.sol",
    solcPath: "/tools/solc",
    basePath: "/workspace",
    includePaths: ["/workspace/node_modules", "/shared/contracts"],
    optimize: true,
  });
  assert.deepEqual(args, [
    "solidity", "inspect", "/workspace/My Contract.sol",
    "--format", "json", "--solc", "/tools/solc", "--base-path", "/workspace",
    "--include-path", "/workspace/node_modules",
    "--include-path", "/shared/contracts",
    "--optimize",
  ]);
});

test("buildRunArguments includes selected ABI call and optional comparison flags", () => {
  const args = buildRunArguments({
    source: "/workspace/Counter.sol",
    solcPath: "solc",
    basePath: "/workspace",
    includePaths: [],
    optimize: false,
    contract: "Counter.sol:Counter",
    functionSignature: "add(uint256,uint256)",
    constructorArgs: "7",
    functionArgs: "2,40",
    gasLimit: 1_000_000,
    diff: true,
    trace: true,
  });
  assert.deepEqual(args, [
    "solidity", "run", "/workspace/Counter.sol",
    "--format", "json", "--solc", "solc", "--base-path", "/workspace",
    "--contract", "Counter.sol:Counter",
    "--function", "add(uint256,uint256)",
    "--gas", "1000000",
    "--constructor-args", "7",
    "--args", "2,40",
    "--diff", "--trace",
  ]);
});

test("parseProtocolOutput accepts version one results", () => {
  const result = parseProtocolOutput<InspectResult>(JSON.stringify({
    schemaVersion: 1,
    source: "Counter.sol",
    compiler: { executable: "solc", version: "0.8.30" },
    durationMs: 12,
    contracts: [],
  }));
  assert.equal(result.schemaVersion, 1);
  assert.equal(result.compiler.version, "0.8.30");
});

test("parseProtocolOutput rejects incompatible or malformed output", () => {
  assert.throws(() => parseProtocolOutput<RunResult>("not-json"), /invalid JSON/);
  assert.throws(() => parseProtocolOutput<RunResult>(JSON.stringify({ schemaVersion: 2 })), /protocol version: 2/);
});
