import assert from "node:assert/strict";
import test from "node:test";
import { buildEvidenceModel, terminalSourceLocation } from "../src/insightModel";
import type { RunResult } from "../src/protocol";

const result: RunResult = {
  schemaVersion: 1,
  source: "/workspace/Vault.sol",
  contract: "Vault",
  function: "withdraw(uint256)",
  compiler: { executable: "solc", version: "0.8.30" },
  durationMs: 14,
  execution: {
    engine: "EchoEVM", engineVersion: "v0.0.44", status: "revert", returnData: "0x", gasUsed: 31_204,
    storage: { "0x01": "0x32" }, error: "execution reverted",
    trace: [
      { index: 0, depth: 0, pc: 0, opcode: "0x54", opcodeName: "SLOAD", gasBefore: 100_000, gasAfter: 97_900, stackBefore: [] },
      { index: 1, depth: 0, pc: 2, opcode: "0xfd", opcodeName: "REVERT", gasBefore: 97_900, gasAfter: 68_796, stackBefore: [], haltClass: "revert" },
    ],
  },
  sourceMap: { locations: [
    { pc: 0, file: "Vault.sol", start: 40, length: 10 },
    { pc: 2, file: "Vault.sol", start: 80, length: 28 },
  ] },
};

test("buildEvidenceModel keeps conclusions first and opcode detail bounded", () => {
  const model = buildEvidenceModel(result);
  assert.equal(model[0]?.label, "Execution revert");
  assert.equal(model[1]?.label, "Gas used");
  assert.equal(model.find((node) => node.label === "Storage after execution")?.children?.[0]?.label, "0x01");
  assert.equal(model.find((node) => node.label === "Key execution steps")?.children?.length, 2);
  assert.equal(model.at(-1)?.action, "trace");
});

test("terminalSourceLocation maps the terminal pc to Solidity", () => {
  assert.deepEqual(terminalSourceLocation(result), { pc: 2, file: "Vault.sol", start: 80, length: 28 });
});
