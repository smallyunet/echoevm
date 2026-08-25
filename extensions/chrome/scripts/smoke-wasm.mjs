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
assert.equal(typeof module.executeContract, "function", "Wasm bridge must export executeContract");
assert.equal(typeof module.inferBehavior, "function", "Wasm bridge must export inferBehavior");

const behavior = JSON.parse(module.inferBehavior(JSON.stringify({
  bytecode: "600035631122334414600d57005b60043560015500",
  abi: []
})));
assert.equal(behavior.ok, true);
assert.equal(behavior.result.schema, "echoevm.behavior.v1");
assert.equal(behavior.result.coverage.recognizedSelectors, 1);
assert.equal(behavior.result.functions[0].selector, "0x11223344");
assert.equal(behavior.result.functions[0].effects[0].kind, "storage-write");
assert.equal(behavior.result.functions[0].effects[0].inputs.value, "calldata.arg0");
console.log("EchoEVM Wasm Behavioral ABI smoke test passed (selector and storage effect inferred)");

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

const contract = JSON.parse(module.executeContract(JSON.stringify({
  bytecode: "6004356024350160005260206000f3",
  function: {
    type: "function",
    name: "add",
    inputs: [{ name: "a", type: "uint256" }, { name: "b", type: "uint256" }],
    outputs: [{ name: "", type: "uint256" }],
    stateMutability: "pure"
  },
  args: ["2", "3"],
  fork: "Osaka",
  profile: "arithmetic"
})));
assert.equal(contract.ok, true);
assert.equal(contract.result.execution.status, "success");
assert.equal(contract.result.execution.decodedOutput[0], "Uint(5, 256)");
assert.match(contract.result.calldata, /^0x771602f7/);
assert(contract.result.evidence.events.some((event) => event.op === "ADD"));
console.log("EchoEVM Wasm Contract Lens smoke test passed (add(uint256,uint256) = 5)");

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
