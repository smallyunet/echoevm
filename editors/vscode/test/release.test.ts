import assert from "node:assert/strict";
import test from "node:test";
import { checksumForAsset, latestReleaseAssetURL, releaseAssetName, sha256 } from "../src/release";

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
