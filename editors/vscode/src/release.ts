import { createHash } from "node:crypto";
import * as https from "node:https";

export const latestReleaseDownloadBase = "https://github.com/smallyunet/echoevm/releases/latest/download";
export const homebrewFormula = "smallyunet/tap/echoevm";
export const minimumEchoEVMVersion = "0.0.37";

export function isVersionAtLeast(actual: string, minimum: string): boolean {
  const parse = (value: string): number[] | undefined => {
    const match = /v?(\d+)\.(\d+)\.(\d+)/u.exec(value);
    return match ? match.slice(1).map(Number) : undefined;
  };
  const actualParts = parse(actual);
  const minimumParts = parse(minimum);
  if (!actualParts || !minimumParts) {
    return false;
  }
  for (let index = 0; index < 3; index += 1) {
    if (actualParts[index] !== minimumParts[index]) {
      return actualParts[index] > minimumParts[index];
    }
  }
  return true;
}

export function homebrewExecutableCandidates(platform: NodeJS.Platform): string[] {
  return platform === "darwin" ? ["/opt/homebrew/bin/brew", "/usr/local/bin/brew", "brew"] : [];
}

export function homebrewEchoEVMPath(prefix: string): string {
  return `${prefix.replace(/\/+$/u, "")}/bin/echoevm`;
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

export function latestReleaseAssetURL(assetName: string): string {
  return `${latestReleaseDownloadBase}/${encodeURIComponent(assetName)}`;
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

export async function download(url: string, redirects = 5): Promise<Buffer> {
  if (redirects < 0) {
    throw new Error("Too many redirects while downloading EchoEVM.");
  }
  return new Promise((resolve, reject) => {
    const request = https.get(url, {
      headers: { Accept: "application/octet-stream", "User-Agent": "smallyu.echoevm-vscode" },
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
