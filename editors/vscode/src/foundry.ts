import { constants as fsConstants } from "node:fs";
import { access, readFile } from "node:fs/promises";
import * as os from "node:os";
import * as path from "node:path";

export function parseFoundrySolcVersion(contents: string, requestedProfile = process.env.FOUNDRY_PROFILE ?? "default"): string | undefined {
  const profiles = new Map<string, string>();
  let currentProfile: string | undefined;
  for (const rawLine of contents.split(/\r?\n/u)) {
    const line = rawLine.replace(/\s+#.*$/u, "").trim();
    const section = /^\[profile\.([^\]]+)\]$/u.exec(line);
    if (section) {
      currentProfile = section[1].trim();
      continue;
    }
    if (!currentProfile) {
      continue;
    }
    const assignment = /^(?:solc_version|solc)\s*=\s*["']v?([^"']+)["']$/u.exec(line);
    if (assignment) {
      profiles.set(currentProfile, assignment[1].trim());
    }
  }
  return profiles.get(requestedProfile) ?? profiles.get("default");
}

export async function resolveFoundrySolc(workspaceFolder: string): Promise<string | undefined> {
  let contents: string;
  try {
    contents = await readFile(path.join(workspaceFolder, "foundry.toml"), "utf8");
  } catch {
    return undefined;
  }
  const version = parseFoundrySolcVersion(contents);
  if (!version) {
    return undefined;
  }
  const suffix = process.platform === "win32" ? ".exe" : "";
  const candidate = path.join(os.homedir(), ".svm", version, `solc-${version}${suffix}`);
  try {
    await access(candidate, process.platform === "win32" ? fsConstants.F_OK : fsConstants.X_OK);
    return candidate;
  } catch {
    return undefined;
  }
}
