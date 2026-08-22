const test = require("node:test");
const assert = require("node:assert/strict");
const helpers = require("../lib.js");

test("extractTransactionHash accepts only an Etherscan transaction path", () => {
  const hash = `0x${"ab".repeat(32)}`;
  assert.equal(helpers.extractTransactionHash(`https://etherscan.io/tx/${hash}`), hash);
  assert.equal(helpers.extractTransactionHash(`https://etherscan.io/address/${hash}`), "");
  assert.equal(helpers.extractTransactionHash("not a URL"), "");
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
