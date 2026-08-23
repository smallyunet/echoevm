import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import path from "node:path";
import { pathToFileURL } from "node:url";
import process from "node:process";

const root = process.argv[2];
const fixturePath = process.argv[3];
const conformancePath = process.argv[4];
assert(root, "extension build directory is required");
assert(fixturePath, "valid replay witness fixture is required");
assert(conformancePath, "bytecode conformance matrix is required");
const module = await import(pathToFileURL(path.join(root, "wasm/engine.js")));
const bytes = await readFile(path.join(root, "wasm/engine_bg.wasm"));
await module.default({ module_or_path: bytes });
assert.equal(typeof module.replay, "function", "Wasm bridge must export replay");
assert.equal(typeof module.executeBytecode, "function", "Wasm bridge must export executeBytecode");

const conformance = JSON.parse(await readFile(conformancePath, "utf8"));
assert.equal(conformance.schema, "echoevm.bytecode-conformance.v1");
assert.equal(conformance.vectors.length, 15, "Wasm vector count must not shrink");
const categories = new Set();
const forks = new Set();
for (const vector of conformance.vectors) {
  categories.add(vector.category);
  forks.add(vector.fork);
  const result = JSON.parse(module.executeBytecode(JSON.stringify({
    bytecode: vector.bytecode,
    calldata: "",
    gasLimit: 15_000_000,
    fork: vector.fork,
  })));
  assert.equal(result.status, vector.status, `${vector.name} status`);
  assert.equal(result.returnData, vector.returnData, `${vector.name} return data`);
  assert.equal(result.gasUsed, vector.gasUsed, `${vector.name} gas`);
  assert.equal(result.error ?? null, vector.error ?? null, `${vector.name} error`);
}
assert.deepEqual([...forks].sort(), ["Cancun", "Osaka", "Prague"]);
assert.equal(categories.size, 11, "Wasm conformance category count must not shrink");
console.log(`EchoEVM Wasm bytecode conformance passed (${conformance.vectors.length} vectors, ${categories.size} categories, ${forks.size} forks)`);

const response = JSON.parse(module.replay("{}", JSON.stringify({ profile: "auto", limit: 40, maxMemoryBytes: 256 })));
assert.equal(response.ok, false);
assert.match(response.error, /schema/);
const witness = await readFile(fixturePath, "utf8");
const replay = JSON.parse(module.replay(witness, JSON.stringify({ profile: "auto", limit: 40, maxMemoryBytes: 256 })));
assert.equal(replay.ok, true, replay.error);
assert.equal(replay.result.execution.status, "success");
assert(replay.result.execution.gasUsed >= 21_000);
assert.match(replay.result.transaction.hash, /^0x[0-9a-fA-F]{64}$/);
assert.equal(replay.result.witness.schema, "echoevm.replay-witness.v1");
console.log(`EchoEVM Wasm replay smoke test passed (${replay.result.transaction.hash})`);
process.exit(0);
