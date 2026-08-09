import assert from "node:assert/strict";
import test from "node:test";
import { scanSolidityFunctions } from "../src/solidityScanner";

test("scanSolidityFunctions finds multiline declarations and ignores comments and strings", () => {
  const source = `
// function ignored(uint256 value) external {}
contract Vault {
  string constant LABEL = "function alsoIgnored()";
  function deposit(
    uint256 amount
  ) external {}
  /* function hidden() external {} */
  function withdraw(uint256 amount) external {}
}`;
  assert.deepEqual(scanSolidityFunctions(source).map((item) => item.name), ["deposit", "withdraw"]);
  for (const declaration of scanSolidityFunctions(source)) {
    assert.equal(source.slice(declaration.offset, declaration.offset + declaration.name.length), declaration.name);
  }
});

test("scanSolidityFunctions does not treat constructors or fallback handlers as ABI functions", () => {
  const source = `contract Example {
    constructor() {}
    fallback() external {}
    receive() external payable {}
    function run() external {}
  }`;
  assert.deepEqual(scanSolidityFunctions(source).map((item) => item.name), ["run"]);
});
