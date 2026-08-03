import assert from "node:assert/strict";
import test from "node:test";
import { parseFoundrySolcVersion } from "../src/foundry";

test("parseFoundrySolcVersion reads the requested profile and falls back to default", () => {
  const contents = `
[profile.default]
solc_version = "0.8.33"

[profile.ci]
solc = '0.8.30' # pinned in CI
`;
  assert.equal(parseFoundrySolcVersion(contents), "0.8.33");
  assert.equal(parseFoundrySolcVersion(contents, "ci"), "0.8.30");
  assert.equal(parseFoundrySolcVersion(contents, "missing"), "0.8.33");
});

test("parseFoundrySolcVersion ignores unrelated TOML sections", () => {
  assert.equal(parseFoundrySolcVersion('[fmt]\nsolc = "0.8.99"\n'), undefined);
});
