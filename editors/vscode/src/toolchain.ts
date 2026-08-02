import { constants as fsConstants } from "node:fs";
import { access, chmod, mkdir, writeFile } from "node:fs/promises";
import * as path from "node:path";
import { spawn } from "node:child_process";
import * as vscode from "vscode";
import { checksumForAsset, download, fetchLatestRelease, releaseAssetName, sha256 } from "./release";

export interface ResolvedToolchain {
  echoevm: string;
  solc: string;
  solcArgs: string[];
  environment?: NodeJS.ProcessEnv;
}

export interface ToolStatus {
  echoevm?: { path: string; version: string };
  solc?: { path: string; version: string };
}

export class ToolchainManager {
  public constructor(
    private readonly context: vscode.ExtensionContext,
    private readonly output: vscode.LogOutputChannel,
  ) {}

  public async diagnose(resource?: vscode.Uri): Promise<ToolStatus> {
    const echoevm = await this.resolveEchoEVM(resource);
    const solc = await this.resolveSolc(resource);
    const [echoVersion, solcVersion] = await Promise.all([
      probe(echoevm, ["version", "--json"]),
      probe(solc.path, [...solc.args, "--version"], solc.environment),
    ]);
    return {
      echoevm: echoVersion ? { path: echoevm, version: compactVersion(echoVersion) } : undefined,
      solc: solcVersion ? { path: solc.path, version: compactVersion(solcVersion) } : undefined,
    };
  }

  public async ensureReady(resource: vscode.Uri): Promise<ResolvedToolchain | undefined> {
    let status = await this.diagnose(resource);
    if (!status.echoevm) {
      const action = await vscode.window.showWarningMessage(
        "EchoEVM CLI is required to run Solidity functions.",
        "Install EchoEVM",
        "Choose Executable",
      );
      if (action === "Install EchoEVM") {
        try {
          await vscode.window.withProgress(
            { location: vscode.ProgressLocation.Notification, title: "Installing verified EchoEVM CLI" },
            () => this.installEchoEVM(),
          );
        } catch (error) {
          const message = error instanceof Error ? error.message : String(error);
          this.output.error(message);
          await vscode.window.showErrorMessage(`EchoEVM installation failed: ${message}`);
          return undefined;
        }
      } else if (action === "Choose Executable") {
        await this.chooseExecutable("executablePath", "Select the EchoEVM executable");
      } else {
        return undefined;
      }
      status = await this.diagnose(resource);
    }
    if (!status.solc) {
      const action = await vscode.window.showWarningMessage(
        "The bundled Solidity compiler is unavailable. Choose a native compiler or review the installation guide.",
        "Choose solc",
        "Installation Guide",
      );
      if (action === "Choose solc") {
        await this.chooseExecutable("solcPath", "Select solc or solcjs");
        status = await this.diagnose(resource);
      } else if (action === "Installation Guide") {
        await vscode.env.openExternal(vscode.Uri.parse("https://docs.soliditylang.org/en/latest/installing-solidity.html"));
      }
    }
    if (!status.echoevm || !status.solc) {
      return undefined;
    }
    const compiler = await this.resolveSolc(resource);
    return {
      echoevm: status.echoevm.path,
      solc: compiler.path,
      solcArgs: compiler.args,
      environment: compiler.environment,
    };
  }

  public async installEchoEVM(): Promise<string> {
    const assetName = releaseAssetName(process.platform, process.arch);
    const release = await fetchLatestRelease();
    const binaryAsset = release.assets.find((asset) => asset.name === assetName);
    const sumsAsset = release.assets.find((asset) => asset.name === "SHA256SUMS");
    if (!binaryAsset || !sumsAsset) {
      throw new Error(`${release.tag_name} does not contain the verified ${assetName} release assets.`);
    }
    const [binary, manifest] = await Promise.all([
      download(binaryAsset.browser_download_url),
      download(sumsAsset.browser_download_url),
    ]);
    const expected = checksumForAsset(manifest.toString("utf8"), assetName);
    const actual = sha256(binary);
    if (actual !== expected) {
      throw new Error(`EchoEVM checksum mismatch: expected ${expected}, got ${actual}.`);
    }
    const destination = this.managedEchoEVMPath();
    await mkdir(path.dirname(destination), { recursive: true });
    const temporary = `${destination}.download`;
    await writeFile(temporary, binary, { mode: 0o755 });
    await vscode.workspace.fs.rename(vscode.Uri.file(temporary), vscode.Uri.file(destination), { overwrite: true });
    if (process.platform !== "win32") {
      await chmod(destination, 0o755);
    }
    await vscode.workspace.getConfiguration("echoevm").update("executablePath", undefined, vscode.ConfigurationTarget.Global);
    this.output.info(`Installed ${release.tag_name} to ${destination} after SHA-256 verification.`);
    return destination;
  }

  public async chooseExecutable(setting: "executablePath" | "solcPath", title: string): Promise<void> {
    const selected = await vscode.window.showOpenDialog({ title, canSelectFiles: true, canSelectFolders: false, canSelectMany: false });
    if (!selected?.[0]) {
      return;
    }
    await vscode.workspace.getConfiguration("echoevm", selected[0]).update(setting, selected[0].fsPath, vscode.ConfigurationTarget.Global);
  }

  public async useBundledCompiler(): Promise<void> {
    await vscode.workspace.getConfiguration("echoevm").update("solcPath", undefined, vscode.ConfigurationTarget.Global);
  }

  private async resolveEchoEVM(resource?: vscode.Uri): Promise<string> {
    const configured = vscode.workspace.getConfiguration("echoevm", resource).get<string>("executablePath", "echoevm");
    if (configured !== "echoevm") {
      return configured;
    }
    const managed = this.managedEchoEVMPath();
    return await isExecutable(managed) ? managed : configured;
  }

  private async resolveSolc(resource?: vscode.Uri): Promise<{ path: string; args: string[]; environment?: NodeJS.ProcessEnv }> {
    const configured = vscode.workspace.getConfiguration("echoevm", resource).get<string>("solcPath", "solc");
    if (configured !== "solc") {
      return { path: configured, args: [] };
    }
    const folder = resource ? vscode.workspace.getWorkspaceFolder(resource) : undefined;
    const candidates = folder && process.platform !== "win32" ? [
      path.join(folder.uri.fsPath, "node_modules", ".bin", "solc"),
      path.join(folder.uri.fsPath, "node_modules", ".bin", "solcjs"),
    ] : [];
    for (const candidate of candidates) {
      if (await isExecutable(candidate)) {
        return { path: candidate, args: [] };
      }
    }
    const environment = { ...process.env, ELECTRON_RUN_AS_NODE: "1" };
    return {
      path: process.execPath,
      args: [path.join(this.context.extensionUri.fsPath, "dist", "solcjs.cjs")],
      environment,
    };
  }

  private managedEchoEVMPath(): string {
    return path.join(this.context.globalStorageUri.fsPath, "bin", process.platform === "win32" ? "echoevm.exe" : "echoevm");
  }
}

async function isExecutable(file: string): Promise<boolean> {
  try {
    await access(file, process.platform === "win32" ? fsConstants.F_OK : fsConstants.X_OK);
    return true;
  } catch {
    return false;
  }
}

async function probe(executable: string, args: string[], environment?: NodeJS.ProcessEnv): Promise<string | undefined> {
  return new Promise((resolve) => {
    const child = spawn(executable, args, { env: environment, shell: false, windowsHide: true });
    const chunks: Buffer[] = [];
    const timer = setTimeout(() => child.kill(), 5_000);
    child.stdout.on("data", (chunk: Buffer) => chunks.push(chunk));
    child.stderr.on("data", (chunk: Buffer) => chunks.push(chunk));
    child.on("error", () => {
      clearTimeout(timer);
      resolve(undefined);
    });
    child.on("close", (code) => {
      clearTimeout(timer);
      resolve(code === 0 ? Buffer.concat(chunks).toString("utf8") : undefined);
    });
  });
}

function compactVersion(output: string): string {
  const trimmed = output.trim();
  try {
    const parsed = JSON.parse(trimmed) as { version?: string };
    if (parsed.version) {
      return parsed.version;
    }
  } catch {
    // Plain-text compiler version output is expected here.
  }
  return trimmed.split(/\r?\n/u).filter(Boolean).at(-1) ?? "unknown";
}
