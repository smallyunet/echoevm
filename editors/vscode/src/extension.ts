import * as path from "node:path";
import * as vscode from "vscode";
import { EchoEVMClient } from "./client";
import type { CommonCommandOptions, RunResult, SolidityContract, SolidityFunction } from "./protocol";
import { showTracePanel } from "./tracePanel";

let lastResult: RunResult | undefined;

export function activate(context: vscode.ExtensionContext): void {
  const output = vscode.window.createOutputChannel("EchoEVM", { log: true });
  context.subscriptions.push(
    output,
    vscode.commands.registerCommand("echoevm.runFunction", () => executeActiveFunction(output, false)),
    vscode.commands.registerCommand("echoevm.runAndCompare", () => executeActiveFunction(output, true)),
    vscode.commands.registerCommand("echoevm.showLastTrace", () => {
      if (!lastResult) {
        void vscode.window.showInformationMessage("Run a Solidity function with EchoEVM first.");
        return;
      }
      showTracePanel(lastResult);
    }),
  );
}

export function deactivate(): void {}

async function executeActiveFunction(output: vscode.LogOutputChannel, diff: boolean): Promise<void> {
  const editor = vscode.window.activeTextEditor;
  if (!editor || path.extname(editor.document.uri.fsPath).toLowerCase() !== ".sol") {
    await vscode.window.showErrorMessage("Open a Solidity (.sol) file before running EchoEVM.");
    return;
  }
  if (editor.document.isDirty && !(await editor.document.save())) {
    await vscode.window.showErrorMessage("Save the Solidity file before running EchoEVM.");
    return;
  }

  const source = editor.document.uri.fsPath;
  const workspaceFolder = vscode.workspace.getWorkspaceFolder(editor.document.uri);
  const cwd = workspaceFolder?.uri.fsPath ?? path.dirname(source);
  const configuration = vscode.workspace.getConfiguration("echoevm", editor.document.uri);
  const executable = configuration.get<string>("executablePath", "echoevm");
  const solcPath = configuration.get<string>("solcPath", "solc");
  const includePaths = configuration.get<string[]>("includePaths", []).map((item) => path.resolve(cwd, item));
  const optimize = configuration.get<boolean>("optimize", false);
  const gasLimit = configuration.get<number>("gasLimit", 1_000_000);
  const common: CommonCommandOptions = { source, solcPath, basePath: cwd, includePaths, optimize };
  const client = new EchoEVMClient(executable);

  try {
    const inspection = await vscode.window.withProgress(
      { location: vscode.ProgressLocation.Notification, title: "EchoEVM: compiling Solidity", cancellable: true },
      (_progress, token) => client.inspect(common, cwd, token),
    );
    const contract = await pickContract(inspection.contracts);
    if (!contract) {
      return;
    }
    const solidityFunction = await pickFunction(contract);
    if (!solidityFunction) {
      return;
    }
    const constructorArgs = await collectArguments("Constructor arguments", contract.constructorInputs);
    if (constructorArgs === undefined) {
      return;
    }
    const functionArgs = await collectArguments(`${solidityFunction.name} arguments`, solidityFunction.inputs);
    if (functionArgs === undefined) {
      return;
    }

    const result = await vscode.window.withProgress(
      { location: vscode.ProgressLocation.Notification, title: `EchoEVM: running ${solidityFunction.signature}`, cancellable: true },
      (_progress, token) => client.run({
        ...common,
        contract: contract.key,
        functionSignature: solidityFunction.signature,
        constructorArgs,
        functionArgs,
        gasLimit,
        diff,
        trace: true,
      }, cwd, token),
    );
    lastResult = result;
    writeResult(output, result);
    output.show(true);
    const verdict = result.comparison ? (result.comparison.match ? "MATCH" : "DIVERGENCE") : result.execution.status.toUpperCase();
    const action = await vscode.window.showInformationMessage(
      `EchoEVM ${verdict}: ${result.contract}.${result.function} used ${result.execution.gasUsed} gas.`,
      "Show Trace",
    );
    if (action === "Show Trace") {
      showTracePanel(result);
    }
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    output.error(message);
    output.show(true);
    await vscode.window.showErrorMessage(message);
  }
}

async function pickContract(contracts: SolidityContract[]): Promise<SolidityContract | undefined> {
  if (contracts.length === 0) {
    throw new Error("solc produced no deployable contracts.");
  }
  if (contracts.length === 1) {
    return contracts[0];
  }
  const selected = await vscode.window.showQuickPick(
    contracts.map((contract) => ({ label: contract.name, detail: contract.key, contract })),
    { title: "Select a contract to deploy", placeHolder: "Solidity contract" },
  );
  return selected?.contract;
}

async function pickFunction(contract: SolidityContract): Promise<SolidityFunction | undefined> {
  if (contract.functions.length === 0) {
    throw new Error(`${contract.name} exposes no ABI functions.`);
  }
  const selected = await vscode.window.showQuickPick(
    contract.functions.map((solidityFunction) => ({
      label: solidityFunction.signature,
      description: solidityFunction.stateMutability,
      detail: outputDescription(solidityFunction),
      solidityFunction,
    })),
    { title: `Select a function from ${contract.name}`, placeHolder: "ABI function" },
  );
  return selected?.solidityFunction;
}

async function collectArguments(title: string, parameters: Array<{ name?: string; type: string }>): Promise<string | undefined> {
  if (parameters.length === 0) {
    return "";
  }
  const signature = parameters.map((parameter) => parameter.name ? `${parameter.name}: ${parameter.type}` : parameter.type).join(", ");
  return vscode.window.showInputBox({
    title,
    prompt: `Enter comma-separated values for ${signature}`,
    placeHolder: parameters.map((parameter) => exampleForType(parameter.type)).join(","),
    ignoreFocusOut: true,
    validateInput: (value) => value.trim() === "" ? "Arguments are required." : undefined,
  });
}

function outputDescription(solidityFunction: SolidityFunction): string {
  if (solidityFunction.outputs.length === 0) {
    return "No return values";
  }
  return `Returns ${solidityFunction.outputs.map((output) => output.type).join(", ")}`;
}

function exampleForType(type: string): string {
  if (type === "bool") return "true";
  if (type === "address") return "0x...";
  if (type.endsWith("[]")) return "[1;2;3]";
  if (type.startsWith("bytes")) return "0x...";
  if (type === "string") return "hello";
  return "0";
}

function writeResult(output: vscode.LogOutputChannel, result: RunResult): void {
  output.info(`${result.contract}.${result.function}`);
  output.info(`Compiler: ${result.compiler.version} (${result.compiler.executable})`);
  output.info(`EchoEVM: status=${result.execution.status} gas=${result.execution.gasUsed} return=${result.execution.returnData}`);
  if (result.execution.error) {
    output.error(`Execution error: ${result.execution.error}`);
  }
  const storageEntries = Object.entries(result.execution.storage);
  if (storageEntries.length > 0) {
    output.info(`Storage: ${storageEntries.map(([key, value]) => `${key}=${value}`).join(" ")}`);
  }
  if (result.comparison) {
    output.info(`Geth: status=${result.comparison.geth.status} gas=${result.comparison.geth.gasUsed} return=${result.comparison.geth.returnData}`);
    output.info(`Comparison: ${result.comparison.match ? "MATCH" : "DIVERGENCE"} status=${result.comparison.statusMatch} return=${result.comparison.returnDataMatch} gas=${result.comparison.gasMatch} storage=${result.comparison.storageMatch} trace=${result.comparison.traceMatch}`);
    if (result.comparison.firstDivergence) {
      output.warn(`First divergence: ${JSON.stringify(result.comparison.firstDivergence)}`);
    }
  }
  output.info(`Trace steps: ${result.execution.trace?.length ?? 0}; completed in ${result.durationMs} ms`);
}
