import * as path from "node:path";
import * as vscode from "vscode";
import { EchoEVMClient } from "./client";
import type { CommonCommandOptions, RunResult, SolidityContract, SolidityFunction } from "./protocol";
import { ToolchainManager } from "./toolchain";
import { showTracePanel } from "./tracePanel";

let lastResult: RunResult | undefined;

export function activate(context: vscode.ExtensionContext): void {
  const output = vscode.window.createOutputChannel("EchoEVM", { log: true });
  const toolchain = new ToolchainManager(context, output);
  const status = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Left, 30);
  status.command = "echoevm.setup";
  status.show();
  const refreshStatus = async (): Promise<void> => {
    const resource = vscode.window.activeTextEditor?.document.uri;
    const result = await toolchain.diagnose(resource);
    const ready = Boolean(result.echoevm && result.solc);
    status.text = ready ? "$(check) EchoEVM" : "$(warning) EchoEVM Setup";
    status.tooltip = ready
      ? `EchoEVM ${result.echoevm?.version}; solc ${result.solc?.version}`
      : "Install or configure the EchoEVM CLI and Solidity compiler";
    status.backgroundColor = ready ? undefined : new vscode.ThemeColor("statusBarItem.warningBackground");
  };
  context.subscriptions.push(
    output,
    status,
    vscode.commands.registerCommand("echoevm.setup", async () => {
      try {
        await runSetup(toolchain, vscode.window.activeTextEditor?.document.uri);
        await refreshStatus();
      } catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        output.error(message);
        await vscode.window.showErrorMessage(`EchoEVM setup failed: ${message}`);
      }
    }),
    vscode.commands.registerCommand("echoevm.openExample", () => openExample(context)),
    vscode.commands.registerCommand("echoevm.runFunction", async () => {
      await executeActiveFunction(output, toolchain, false);
      await refreshStatus();
    }),
    vscode.commands.registerCommand("echoevm.runAndCompare", async () => {
      await executeActiveFunction(output, toolchain, true);
      await refreshStatus();
    }),
    vscode.commands.registerCommand("echoevm.showLastTrace", () => {
      if (!lastResult) {
        void vscode.window.showInformationMessage("Run a Solidity function with EchoEVM first.");
        return;
      }
      showTracePanel(lastResult);
    }),
    vscode.workspace.onDidChangeConfiguration((event) => {
      if (event.affectsConfiguration("echoevm")) {
        void refreshStatus();
      }
    }),
  );
  void refreshStatus();
}

export function deactivate(): void {}

async function executeActiveFunction(output: vscode.LogOutputChannel, toolchain: ToolchainManager, diff: boolean): Promise<void> {
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
  const tools = await toolchain.ensureReady(editor.document.uri);
  if (!tools) {
    return;
  }
  const workspaceFolder = vscode.workspace.getWorkspaceFolder(editor.document.uri);
  const cwd = workspaceFolder?.uri.fsPath ?? path.dirname(source);
  const configuration = vscode.workspace.getConfiguration("echoevm", editor.document.uri);
  const executable = tools.echoevm;
  const solcPath = tools.solc;
  const includePaths = configuration.get<string[]>("includePaths", []).map((item) => path.resolve(cwd, item));
  const optimize = configuration.get<boolean>("optimize", false);
  const gasLimit = configuration.get<number>("gasLimit", 1_000_000);
  const common: CommonCommandOptions = { source, solcPath, solcArgs: tools.solcArgs, basePath: cwd, includePaths, optimize };
  const client = new EchoEVMClient(executable, tools.environment);

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
  if (contract.functions.length === 1) {
    return contract.functions[0];
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

async function runSetup(toolchain: ToolchainManager, resource?: vscode.Uri): Promise<void> {
  const current = await toolchain.diagnose(resource);
  const choices: string[] = [
    "Show detected versions",
    "Install or update verified EchoEVM CLI",
    "Choose EchoEVM executable",
    "Use bundled Solidity compiler",
    "Choose Solidity compiler",
  ];
  choices.push("Open Solidity installation guide");
  const action = await vscode.window.showQuickPick(choices, {
    title: "EchoEVM Setup",
    placeHolder: "Resolve a missing tool",
  });
  if (action === "Show detected versions") {
    await vscode.window.showInformationMessage(
      `EchoEVM: ${current.echoevm?.version ?? "missing"}; compiler: ${current.solc?.version ?? "missing"}.`,
    );
  } else if (action === "Install or update verified EchoEVM CLI") {
    await vscode.window.withProgress(
      { location: vscode.ProgressLocation.Notification, title: "Installing verified EchoEVM CLI" },
      () => toolchain.installEchoEVM(),
    );
    await vscode.window.showInformationMessage("EchoEVM CLI installed and verified.");
  } else if (action === "Choose EchoEVM executable") {
    await toolchain.chooseExecutable("executablePath", "Select the EchoEVM executable");
  } else if (action === "Use bundled Solidity compiler") {
    await toolchain.useBundledCompiler();
    await vscode.window.showInformationMessage("EchoEVM will use the bundled solc-js compiler.");
  } else if (action === "Choose Solidity compiler") {
    await toolchain.chooseExecutable("solcPath", "Select solc or solcjs");
  } else if (action === "Open Solidity installation guide") {
    await vscode.env.openExternal(vscode.Uri.parse("https://docs.soliditylang.org/en/latest/installing-solidity.html"));
  }
}

async function openExample(context: vscode.ExtensionContext): Promise<void> {
  let folder = vscode.workspace.workspaceFolders?.[0]?.uri;
  if (!folder) {
    const selected = await vscode.window.showOpenDialog({
      title: "Choose a folder for the EchoEVM example",
      canSelectFolders: true,
      canSelectFiles: false,
      canSelectMany: false,
    });
    folder = selected?.[0];
  }
  if (!folder) {
    return;
  }
  const exampleDirectory = vscode.Uri.joinPath(folder, ".echoevm");
  const destination = vscode.Uri.joinPath(exampleDirectory, "Counter.sol");
  try {
    await vscode.workspace.fs.stat(destination);
  } catch {
    const source = vscode.Uri.joinPath(context.extensionUri, "examples", "Counter.sol");
    await vscode.workspace.fs.createDirectory(exampleDirectory);
    await vscode.workspace.fs.writeFile(destination, await vscode.workspace.fs.readFile(source));
  }
  const document = await vscode.workspace.openTextDocument(destination);
  await vscode.window.showTextDocument(document);
  const action = await vscode.window.showInformationMessage(
    "EchoEVM example is ready. Run increment() and inspect its opcode trace.",
    "Run Example",
  );
  if (action === "Run Example") {
    await vscode.commands.executeCommand("echoevm.runFunction");
  }
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
