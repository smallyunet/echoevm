import assert from "node:assert/strict";
import test from "node:test";
import {
  checksumForAsset,
  homebrewEchoEVMPath,
  homebrewExecutableCandidates,
  homebrewFormula,
  latestReleaseAssetURL,
  releaseAssetName,
  sha256,
} from "../src/release";

test("Homebrew installation metadata targets the public EchoEVM tap on macOS", () => {
  assert.equal(homebrewFormula, "smallyunet/tap/echoevm");
  assert.deepEqual(homebrewExecutableCandidates("darwin"), ["/opt/homebrew/bin/brew", "/usr/local/bin/brew", "brew"]);
  assert.deepEqual(homebrewExecutableCandidates("linux"), []);
  assert.equal(homebrewEchoEVMPath("/opt/homebrew/opt/echoevm/"), "/opt/homebrew/opt/echoevm/bin/echoevm");
});

test("releaseAssetName maps VS Code runtime platforms to release binaries", () => {
  assert.equal(releaseAssetName("darwin", "arm64"), "echoevm-darwin-arm64");
  assert.equal(releaseAssetName("linux", "x64"), "echoevm-linux-amd64");
  assert.equal(releaseAssetName("win32", "x64"), "echoevm-windows-amd64.exe");
  assert.throws(() => releaseAssetName("win32", "arm64"), /Windows ARM64/);
});

test("latestReleaseAssetURL uses GitHub's API-free latest download route", () => {
  assert.equal(
    latestReleaseAssetURL("echoevm-darwin-arm64"),
    "https://github.com/smallyunet/echoevm/releases/latest/download/echoevm-darwin-arm64",
  );
  assert.equal(
    latestReleaseAssetURL("SHA256SUMS"),
    "https://github.com/smallyunet/echoevm/releases/latest/download/SHA256SUMS",
  );
});

test("checksumForAsset selects an exact asset and sha256 verifies bytes", () => {
  const digest = sha256(Buffer.from("echoevm"));
  const manifest = `${"0".repeat(64)}  other-file\n${digest}  echoevm-linux-amd64\n`;
  assert.equal(checksumForAsset(manifest, "echoevm-linux-amd64"), digest);
  assert.throws(() => checksumForAsset(manifest, "echoevm-darwin-arm64"), /does not contain/);
});
