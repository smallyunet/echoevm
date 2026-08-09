import assert from "node:assert/strict";
import { execFile } from "node:child_process";
import { mkdtemp, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import { promisify } from "node:util";

const execute = promisify(execFile);
const echoevm = process.env.ECHOEVM_TEST_BINARY;
const solc = process.env.ECHOEVM_TEST_SOLC ?? process.execPath;
const solcArgs = process.env.ECHOEVM_TEST_SOLC ? [] : [path.resolve("dist/solcjs.cjs")];
if (!echoevm) {
  throw new Error("ECHOEVM_TEST_BINARY is required");
}

const directory = await mkdtemp(path.join(tmpdir(), "echoevm-vscode-smoke-"));
const source = path.join(directory, "Counter.sol");
await writeFile(path.join(directory, "Math.sol"), `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;
library Math { function increment(uint256 value) internal pure returns (uint256) { return value + 1; } }
`);
await writeFile(source, `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;
import "./Math.sol";
contract Counter {
  uint256 private value = 41;
  function increment() public returns (uint256) { value = Math.increment(value); return value; }
}
`);

const common = ["--format", "json", "--solc", solc];
for (const argument of solcArgs) {
  common.push("--solc-arg", argument);
}
common.push("--base-path", directory);
const inspection = JSON.parse((await execute(echoevm, ["solidity", "inspect", source, ...common])).stdout);
assert.equal(inspection.schemaVersion, 1);
const counter = inspection.contracts.find((contract) => contract.name === "Counter");
assert.ok(counter);
assert.equal(counter.functions[0].signature, "increment()");
assert.equal(counter.functions[0].sourceLocation.file, "Counter.sol");
assert.ok(counter.functions[0].sourceLocation.length > 0);

const result = JSON.parse((await execute(echoevm, [
  "solidity", "run", source, ...common,
  "--contract", counter.key,
  "--function", "increment()",
  "--trace",
])).stdout);
assert.equal(result.schemaVersion, 1);
assert.equal(result.execution.status, "success");
assert.ok(result.execution.trace.length > 0);
assert.ok(result.sourceMap.locations.length > 0);
assert.ok(result.sourceMap.locations.some((location) => location.file === "Counter.sol"));
console.log(`EchoEVM client smoke passed with ${result.execution.trace.length} trace steps.`);
