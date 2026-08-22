import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import path from "node:path";
import { pathToFileURL } from "node:url";
import process from "node:process";
import { webcrypto } from "node:crypto";

const root = process.argv[2];
const fixturePath = process.argv[3];
assert(root, "extension build directory is required");
assert(fixturePath, "valid replay witness fixture is required");
if (!globalThis.crypto) globalThis.crypto = webcrypto;

await import(pathToFileURL(path.join(root, "wasm/wasm_exec.js")));
assert.equal(typeof globalThis.Go, "function", "wasm_exec.js must expose Go");
const go = new globalThis.Go();
const bytes = await readFile(path.join(root, "wasm/engine.wasm"));
const { instance } = await WebAssembly.instantiate(bytes, go.importObject);
void go.run(instance);

for (let attempt = 0; attempt < 200 && !globalThis.echoevmWasmReady; attempt += 1) {
  await new Promise((resolve) => setTimeout(resolve, 10));
}
assert.equal(typeof globalThis.echoevmReplay, "function", "Wasm bridge must expose echoevmReplay");
const response = JSON.parse(globalThis.echoevmReplay("{}", JSON.stringify({ profile: "auto", limit: 40, maxMemoryBytes: 256 })));
assert.equal(response.ok, false);
assert.match(response.error, /witness schema/);
const witness = await readFile(fixturePath, "utf8");
const replay = JSON.parse(globalThis.echoevmReplay(witness, JSON.stringify({ profile: "auto", limit: 40, maxMemoryBytes: 256 })));
assert.equal(replay.ok, true, replay.error);
assert.equal(replay.result.execution.status, "success");
assert.equal(replay.result.execution.gasUsed, 21_000);
assert.match(replay.result.transaction.hash, /^0x[0-9a-fA-F]{64}$/);
assert.equal(replay.result.witness.schema, "echoevm.replay-witness.v1");
console.log(`EchoEVM Wasm replay smoke test passed (${replay.result.transaction.hash})`);
process.exit(0);
