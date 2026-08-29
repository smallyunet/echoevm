const test = require("node:test");
const assert = require("node:assert/strict");
const helpers = require("../lib.js");

test("extractTransactionHash accepts only an Etherscan transaction path", () => {
  const hash = `0x${"ab".repeat(32)}`;
  assert.equal(helpers.extractTransactionHash(`https://etherscan.io/tx/${hash}`), hash);
  assert.equal(helpers.extractTransactionHash(`https://etherscan.io/address/${hash}`), "");
  assert.equal(helpers.extractTransactionHash("not a URL"), "");
});

test("extractContractAddress accepts Etherscan address pages and hashes", () => {
  const address = `0x${"12".repeat(20)}`;
  assert.equal(helpers.extractContractAddress(`https://etherscan.io/address/${address}#code`), address);
  assert.equal(helpers.extractContractAddress(`https://etherscan.io/tx/${address}`), "");
});

test("contract helpers validate bytecode and describe ABI functions", () => {
  assert.equal(helpers.normalizeBytecode(" 60 00 f3 "), "0x6000f3");
  assert.equal(helpers.normalizeBytecode("0xxyz"), "");
  const abi = helpers.parseContractAbi('[{"type":"function","name":"add","inputs":[{"name":"a","type":"uint256"}],"stateMutability":"pure"},{"type":"event","name":"Added"}]');
  assert.equal(helpers.abiFunctions(abi).length, 1);
  assert.equal(helpers.functionSignature(abi[0]), "add(uint256)");
  assert.equal(helpers.inputHint(abi[0].inputs[0]), "integer");
  assert.throws(() => helpers.parseContractAbi("{}"), /not an array/);
});

test("validateWitnessText enforces the versioned replay contract", () => {
  assert.deepEqual(helpers.validateWitnessText('{"schema":"echoevm.replay-witness.v1"}'), { schema: "echoevm.replay-witness.v1" });
  assert.throws(() => helpers.validateWitnessText("{}"), /Expected an echoevm\.replay-witness\.v1/);
  assert.throws(() => helpers.validateWitnessText("{"), /not valid JSON/);
});

test("display helpers keep transaction data compact", () => {
  assert.equal(helpers.shortHex("0x1234567890abcdef", 6, 4), "0x1234…cdef");
  assert.equal(helpers.formatInteger(1234567), "1,234,567");
  assert.deepEqual(helpers.evidenceEvents({ evidence: { events: [{ step: 1 }] } }), [{ step: 1 }]);
});

test("pollForValue tolerates delayed Etherscan contract markup", async () => {
  let attempts = 0;
  const value = await helpers.pollForValue(() => {
    attempts += 1;
    if (attempts === 1) throw new Error("tab replaced while rendering");
    return attempts === 3 ? "runtime-bytecode" : null;
  }, 4, 0);
  assert.equal(value, "runtime-bytecode");
  assert.equal(attempts, 3);
  assert.equal(await helpers.pollForValue(() => null, 2, 0), null);
  await assert.rejects(helpers.pollForValue(() => {
    throw new Error("broken contract markup reader");
  }, 2, 0), /broken contract markup reader/);
});
