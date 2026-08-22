import assert from "node:assert/strict";
import { readFile, stat } from "node:fs/promises";
import path from "node:path";
import process from "node:process";

const root = process.argv[2];
assert(root, "extension build directory is required");

const manifest = JSON.parse(await readFile(path.join(root, "manifest.json"), "utf8"));
assert.equal(manifest.manifest_version, 3);
assert.match(manifest.version, /^\d+\.\d+\.\d+$/);
assert.match(manifest.content_security_policy.extension_pages, /wasm-unsafe-eval/);
assert.deepEqual(manifest.permissions, undefined, "the first release should not request broad permissions");

const required = [
  "background.js", "content.js", "content.css", "lib.js", "popup.html", "popup.js", "popup.css",
  "LICENSE", "THIRD_PARTY_NOTICES.md",
  "wasm/engine.js", "wasm/engine_bg.wasm",
  "icons/icon-16.png", "icons/icon-32.png", "icons/icon-48.png", "icons/icon-128.png"
];
for (const relative of required) {
  const info = await stat(path.join(root, relative));
  assert(info.isFile(), `${relative} must be a file`);
}

const wasm = await readFile(path.join(root, "wasm/engine_bg.wasm"));
assert.deepEqual([...wasm.subarray(0, 4)], [0x00, 0x61, 0x73, 0x6d], "engine.wasm must have the Wasm magic header");
assert(wasm.byteLength > 100_000, "engine.wasm is unexpectedly small");

for (const relative of ["background.js", "content.js", "popup.js", "wasm/engine.js"]) {
  const source = await readFile(path.join(root, relative), "utf8");
  assert(!/<script[^>]+src=["']https?:/i.test(source), `${relative} must not load remote code`);
}

console.log(`Validated EchoEVM Chrome extension v${manifest.version} (${(wasm.byteLength / 1024 / 1024).toFixed(1)} MiB Wasm)`);
