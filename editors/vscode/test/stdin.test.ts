import assert from "node:assert/strict";
import { PassThrough } from "node:stream";
import test from "node:test";
import { readUtf8 } from "../src/stdin";

test("readUtf8 waits for delayed stdin chunks and EOF", async () => {
  const stdin = new PassThrough();
  const contents = readUtf8(stdin);

  await new Promise<void>((resolve) => setImmediate(resolve));
  stdin.write('{"language":"Solidity",');
  await new Promise<void>((resolve) => setImmediate(resolve));
  stdin.end('"sources":{}}');

  assert.equal(await contents, '{"language":"Solidity","sources":{}}');
});
