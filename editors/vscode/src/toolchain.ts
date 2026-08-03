import { constants as fsConstants } from "node:fs";
import { access, chmod, mkdir, writeFile } from "node:fs/promises";
import * as path from "node:path";
import { spawn } from "node:child_process";
import * as vscode from "vscode";
import { resolveFoundrySolc } from "./foundry";
import {
  checksumForAsset,
  download,
  homebrewEchoEVMPath,
  homebrewExecutableCandidates,
  homebrewFormula,
  isVersionAtLeast,
  latestReleaseAssetURL,
  minimumEchoEVMVersion,
  releaseAssetName,
  sha256,
} from "./release";

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
            { location: vscode.ProgressLocation.Notification, title: "Installing EchoEVM CLI" },
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
    if (status.echoevm && !isVersionAtLeast(status.echoevm.version, minimumEchoEVMVersion)) {
      const action = await vscode.window.showWarningMessage(
        `EchoEVM CLI v${minimumEchoEVMVersion} or newer is required by this extension (found ${status.echoevm.version}).`,
        "Update EchoEVM",
        "Choose Executable",
      );
      if (action === "Update EchoEVM") {
        try {
          await vscode.window.withProgress(
            { location: vscode.ProgressLocation.Notification, title: "Updating EchoEVM CLI" },
            () => this.installEchoEVM(),
          );
        } catch (error) {
          const message = error instanceof Error ? error.message : String(error);
          this.output.error(message);
          await vscode.window.showErrorMessage(`EchoEVM update failed: ${message}`);
          return undefined;
        }
      } else if (action === "Choose Executable") {
        await this.chooseExecutable("executablePath", "Select EchoEVM v0.0.37 or newer");
      } else {
        return undefined;
      }
      status = await this.diagnose(resource);
    }
    if (status.echoevm && !isVersionAtLeast(status.echoevm.version, minimumEchoEVMVersion)) {
      await vscode.window.showErrorMessage(
        `EchoEVM CLI remains at ${status.echoevm.version}; install v${minimumEchoEVMVersion} or newer and retry.`,
      );
      return undefined;
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
    if (process.platform === "darwin") {
      return this.installEchoEVMWithHomebrew();
    }

    const assetName = releaseAssetName(process.platform, process.arch);
    const [binary, manifest] = await Promise.all([
      download(latestReleaseAssetURL(assetName)),
      download(latestReleaseAssetURL("SHA256SUMS")),
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
    this.output.info(`Installed the latest EchoEVM CLI to ${destination} after SHA-256 verification.`);
    return destination;
  }

  private async installEchoEVMWithHomebrew(): Promise<string> {
    const brew = await this.findHomebrew();
    let installed = false;
    try {
      installed = Boolean((await runCommand(brew, ["list", "--versions", homebrewFormula], 30_000)).trim());
    } catch {
      // A missing formula is expected on first install.
    }
    const installOutput = await runCommand(brew, [installed ? "upgrade" : "install", homebrewFormula], 5 * 60_000);
    if (installOutput.trim()) {
      this.output.info(installOutput.trim());
    }
    const prefix = (await runCommand(brew, ["--prefix", homebrewFormula], 30_000)).trim();
    const destination = homebrewEchoEVMPath(prefix);
    if (!await isExecutable(destination)) {
      throw new Error(`Homebrew completed without an executable at ${destination}.`);
    }
    await vscode.workspace.getConfiguration("echoevm").update("executablePath", destination, vscode.ConfigurationTarget.Global);
    this.output.info(`Installed EchoEVM with Homebrew at ${destination}.`);
    return destination;
  }

  private async findHomebrew(): Promise<string> {
    for (const candidate of homebrewExecutableCandidates(process.platform)) {
      if (await probe(candidate, ["--version"])) {
        return candidate;
      }
    }
    throw new Error("Homebrew is required to install EchoEVM on macOS. Install it from https://brew.sh and retry.");
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
    const foundryCompiler = folder ? await resolveFoundrySolc(folder.uri.fsPath) : undefined;
    if (foundryCompiler) {
      return { path: foundryCompiler, args: [] };
    }
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

async function runCommand(executable: string, args: string[], timeout: number): Promise<string> {
  return new Promise((resolve, reject) => {
    const child = spawn(executable, args, { shell: false, windowsHide: true });
    const stdout: Buffer[] = [];
    const stderr: Buffer[] = [];
    const timer = setTimeout(() => child.kill(), timeout);
    child.stdout.on("data", (chunk: Buffer) => stdout.push(chunk));
    child.stderr.on("data", (chunk: Buffer) => stderr.push(chunk));
    child.on("error", (error) => {
      clearTimeout(timer);
      reject(error);
    });
    child.on("close", (code, signal) => {
      clearTimeout(timer);
      const output = Buffer.concat([...stdout, ...stderr]).toString("utf8");
      if (code === 0) {
        resolve(output);
        return;
      }
      const reason = signal ? `signal ${signal}` : `exit code ${code ?? "unknown"}`;
      reject(new Error(`${executable} ${args.join(" ")} failed with ${reason}: ${output.trim()}`));
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
