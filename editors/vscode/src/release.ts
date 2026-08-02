import { createHash } from "node:crypto";
import * as https from "node:https";

export const releaseAPI = "https://api.github.com/repos/smallyunet/echoevm/releases/latest";

export interface ReleaseAsset {
  name: string;
  browser_download_url: string;
}

export interface LatestRelease {
  tag_name: string;
  assets: ReleaseAsset[];
}

export function releaseAssetName(platform: NodeJS.Platform, arch: string): string {
  const goos = platform === "win32" ? "windows" : platform;
  const goarch = arch === "x64" ? "amd64" : arch;
  if (!(["linux", "darwin", "windows"] as string[]).includes(goos) || !(["amd64", "arm64"] as string[]).includes(goarch)) {
    throw new Error(`EchoEVM does not publish a CLI for ${platform}/${arch}.`);
  }
  if (goos === "windows" && goarch !== "amd64") {
    throw new Error("EchoEVM does not yet publish a Windows ARM64 CLI.");
  }
  return `echoevm-${goos}-${goarch}${goos === "windows" ? ".exe" : ""}`;
}

export function checksumForAsset(manifest: string, assetName: string): string {
  for (const line of manifest.split(/\r?\n/u)) {
    const match = /^([a-fA-F0-9]{64})\s+\*?(.+)$/u.exec(line.trim());
    if (match?.[2] === assetName) {
      return match[1].toLowerCase();
    }
  }
  throw new Error(`SHA256SUMS does not contain ${assetName}.`);
}

export function sha256(contents: Uint8Array): string {
  return createHash("sha256").update(contents).digest("hex");
}

export async function fetchLatestRelease(): Promise<LatestRelease> {
  return JSON.parse((await download(releaseAPI)).toString("utf8")) as LatestRelease;
}

export async function download(url: string, redirects = 5): Promise<Buffer> {
  if (redirects < 0) {
    throw new Error("Too many redirects while downloading EchoEVM.");
  }
  return new Promise((resolve, reject) => {
    const request = https.get(url, {
      headers: { Accept: "application/vnd.github+json", "User-Agent": "smallyu.echoevm-vscode" },
    }, (response) => {
      const location = response.headers.location;
      if (response.statusCode && response.statusCode >= 300 && response.statusCode < 400 && location) {
        response.resume();
        void download(new URL(location, url).toString(), redirects - 1).then(resolve, reject);
        return;
      }
      if (response.statusCode !== 200) {
        response.resume();
        reject(new Error(`Download failed with HTTP ${response.statusCode ?? "unknown"}.`));
        return;
      }
      const chunks: Buffer[] = [];
      response.on("data", (chunk: Buffer) => chunks.push(chunk));
      response.on("end", () => resolve(Buffer.concat(chunks)));
      response.on("error", reject);
    });
    request.setTimeout(30_000, () => request.destroy(new Error("EchoEVM download timed out.")));
    request.on("error", reject);
  });
}
