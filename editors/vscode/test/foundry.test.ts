import assert from "node:assert/strict";
import { mkdtemp, mkdir, writeFile } from "node:fs/promises";
import * as os from "node:os";
import * as path from "node:path";
import test from "node:test";
import { parseFoundrySolcVersion, resolveSolidityProjectRoot } from "../src/foundry";

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

test("resolveSolidityProjectRoot selects the nearest nested Foundry project", async () => {
  const workspace = await mkdtemp(path.join(os.tmpdir(), "echoevm-workspace-"));
  const project = path.join(workspace, "contracts");
  const source = path.join(project, "src", "Example.sol");
  await mkdir(path.dirname(source), { recursive: true });
  await writeFile(path.join(project, "foundry.toml"), "[profile.default]\n");
  await writeFile(source, "contract Example {}\n");

  assert.equal(await resolveSolidityProjectRoot(source, workspace), project);
});

test("resolveSolidityProjectRoot falls back to the workspace without a project marker", async () => {
  const workspace = await mkdtemp(path.join(os.tmpdir(), "echoevm-workspace-"));
  const source = path.join(workspace, "src", "Example.sol");
  await mkdir(path.dirname(source), { recursive: true });
  await writeFile(source, "contract Example {}\n");

  assert.equal(await resolveSolidityProjectRoot(source, workspace), workspace);
});
