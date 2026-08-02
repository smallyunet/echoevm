import { spawn } from "node:child_process";
import type { CancellationToken } from "vscode";
import {
  buildInspectArguments,
  buildRunArguments,
  isErrorResult,
  parseProtocolOutput,
  type CommonCommandOptions,
  type ErrorResult,
  type InspectResult,
  type RunCommandOptions,
  type RunResult,
} from "./protocol";

const maxOutputBytes = 32 * 1024 * 1024;

export class EchoEVMClient {
  public constructor(
    private readonly executable: string,
    private readonly environment?: NodeJS.ProcessEnv,
  ) {}

  public async inspect(options: CommonCommandOptions, cwd: string, token: CancellationToken): Promise<InspectResult> {
    const result = await runProcess(this.executable, buildInspectArguments(options), cwd, token, this.environment);
    return decodeResult<InspectResult>(result);
  }

  public async run(options: RunCommandOptions, cwd: string, token: CancellationToken): Promise<RunResult> {
    const result = await runProcess(this.executable, buildRunArguments(options), cwd, token, this.environment);
    return decodeResult<RunResult>(result, true);
  }
}

interface ProcessResult {
  stdout: string;
  stderr: string;
  code: number;
}

function decodeResult<T extends { schemaVersion: number }>(result: ProcessResult, allowReportedExecution = false): T {
  if (!result.stdout.trim()) {
    throw new Error(commandFailureMessage(result));
  }
  const decoded = parseProtocolOutput<T | ErrorResult>(result.stdout);
  if (isErrorResult(decoded)) {
    throw new Error(`${decoded.error.code}: ${decoded.error.message}`);
  }
  if (result.code !== 0 && !allowReportedExecution) {
    throw new Error(commandFailureMessage(result));
  }
  return decoded as T;
}

function commandFailureMessage(result: ProcessResult): string {
  const detail = result.stderr.trim() || result.stdout.trim() || `exit code ${result.code}`;
  return `EchoEVM command failed: ${detail}`;
}

function runProcess(executable: string, args: string[], cwd: string, token: CancellationToken, environment?: NodeJS.ProcessEnv): Promise<ProcessResult> {
  return new Promise((resolve, reject) => {
    if (token.isCancellationRequested) {
      reject(new Error("EchoEVM execution canceled"));
      return;
    }
    const child = spawn(executable, args, { cwd, env: environment, shell: false, windowsHide: true });
    const stdout: Buffer[] = [];
    const stderr: Buffer[] = [];
    let outputBytes = 0;
    let settled = false;

    const cancellation = token.onCancellationRequested(() => {
      child.kill();
    });
    const collect = (target: Buffer[], chunk: Buffer): void => {
      outputBytes += chunk.length;
      if (outputBytes > maxOutputBytes) {
        child.kill();
        if (!settled) {
          settled = true;
          reject(new Error("EchoEVM output exceeded 32 MiB"));
        }
        return;
      }
      target.push(chunk);
    };
    child.stdout.on("data", (chunk: Buffer) => collect(stdout, chunk));
    child.stderr.on("data", (chunk: Buffer) => collect(stderr, chunk));
    child.on("error", (error) => {
      cancellation.dispose();
      if (!settled) {
        settled = true;
        reject(new Error(`Unable to start ${executable}: ${error.message}`));
      }
    });
    child.on("close", (code) => {
      cancellation.dispose();
      if (settled) {
        return;
      }
      settled = true;
      if (token.isCancellationRequested) {
        reject(new Error("EchoEVM execution canceled"));
        return;
      }
      resolve({
        stdout: Buffer.concat(stdout).toString("utf8"),
        stderr: Buffer.concat(stderr).toString("utf8"),
        code: code ?? 1,
      });
    });
  });
}
