import * as path from "node:path";
import * as vscode from "vscode";
import { EchoEVMClient } from "./client";
import { EvidenceTreeProvider, revealEvidenceLocation } from "./evidenceView";
import { ExecutionDecorationManager, SolidityCodeLensProvider, type FunctionTarget } from "./editorInsights";
import { resolveSolidityProjectRoot } from "./foundry";
import type { CommonCommandOptions, RunResult, SolidityContract, SolidityFunction } from "./protocol";
import { scanSolidityFunctions } from "./solidityScanner";
import { ToolchainManager } from "./toolchain";
import { showTracePanel } from "./tracePanel";

let lastResult: RunResult | undefined;

export function activate(context: vscode.ExtensionContext): void {
  const output = vscode.window.createOutputChannel("EchoEVM", { log: true });
  const toolchain = new ToolchainManager(context, output);
  const evidence = new EvidenceTreeProvider();
  const codeLens = new SolidityCodeLensProvider();
  const decorations = new ExecutionDecorationManager();
  const status = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Left, 30);
  status.command = "echoevm.setup";
  status.show();
  const refreshStatus = async (): Promise<void> => {
    const resource = vscode.window.activeTextEditor?.document.uri;
    const projectRoot = resource ? await projectRootForResource(resource) : undefined;
    const result = await toolchain.diagnose(resource, projectRoot);
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
    decorations,
    vscode.window.registerTreeDataProvider("echoevm.executionEvidence", evidence),
    vscode.languages.registerCodeLensProvider({ language: "solidity", scheme: "file" }, codeLens),
    vscode.commands.registerCommand("echoevm.setup", async () => {
      try {
        const resource = vscode.window.activeTextEditor?.document.uri;
        await runSetup(toolchain, resource, resource ? await projectRootForResource(resource) : undefined);
        await refreshStatus();
      } catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        output.error(message);
        await vscode.window.showErrorMessage(`EchoEVM setup failed: ${message}`);
      }
    }),
    vscode.commands.registerCommand("echoevm.openExample", () => openExample(context)),
    vscode.commands.registerCommand("echoevm.runFunction", async () => {
      await executeActiveFunction(output, toolchain, evidence, codeLens, decorations);
      await refreshStatus();
    }),
    vscode.commands.registerCommand("echoevm.runAtFunction", async (target: FunctionTarget) => {
      await executeActiveFunction(output, toolchain, evidence, codeLens, decorations, target);
      await refreshStatus();
    }),
    vscode.commands.registerCommand("echoevm.revealEvidenceLocation", revealEvidenceLocation),
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
      if (event.affectsConfiguration("echoevm.codeLens")) {
        codeLens.refresh();
      }
      if (event.affectsConfiguration("echoevm.inlineResults") && !vscode.workspace.getConfiguration("echoevm").get<boolean>("inlineResults", true)) {
        decorations.clear();
      }
    }),
  );
  void refreshStatus();
}

export function deactivate(): void {}

async function executeActiveFunction(
  output: vscode.LogOutputChannel,
  toolchain: ToolchainManager,
  evidence: EvidenceTreeProvider,
  codeLens: SolidityCodeLensProvider,
  decorations: ExecutionDecorationManager,
  requestedTarget?: FunctionTarget,
): Promise<void> {
  const document = requestedTarget
    ? await vscode.workspace.openTextDocument(vscode.Uri.parse(requestedTarget.uri))
    : vscode.window.activeTextEditor?.document;
  if (!document || path.extname(document.uri.fsPath).toLowerCase() !== ".sol") {
    await vscode.window.showErrorMessage("Open a Solidity (.sol) file before running EchoEVM.");
    return;
  }
  if (document.isDirty && !(await document.save())) {
    await vscode.window.showErrorMessage("Save the Solidity file before running EchoEVM.");
    return;
  }

  const source = document.uri.fsPath;
  const workspaceFolder = vscode.workspace.getWorkspaceFolder(document.uri);
  const projectRoot = await resolveSolidityProjectRoot(source, workspaceFolder?.uri.fsPath);
  const tools = await toolchain.ensureReady(document.uri, projectRoot);
  if (!tools) {
    return;
  }
  const cwd = projectRoot;
  const configuration = vscode.workspace.getConfiguration("echoevm", document.uri);
  const executable = tools.echoevm;
  const solcPath = tools.solc;
  const includePaths = configuration.get<string[]>("includePaths", []).map((item) => path.resolve(cwd, item));
  const optimize = configuration.get<boolean>("optimize", false);
  const optimizerRuns = configuration.get<number>("optimizerRuns", 0);
  const viaIR = configuration.get<boolean>("viaIR", false);
  const remappings = configuration.get<string[]>("remappings", []);
  const gasLimit = configuration.get<number>("gasLimit", 1_000_000);
  const common: CommonCommandOptions = {
    source, solcPath, solcArgs: tools.solcArgs, basePath: cwd, includePaths,
    optimize, optimizerRuns, viaIR, remappings,
  };
  const client = new EchoEVMClient(executable, tools.environment);
  output.info(`Project root: ${projectRoot}`);

  try {
    const inspection = await vscode.window.withProgress(
      { location: vscode.ProgressLocation.Notification, title: "EchoEVM: compiling Solidity", cancellable: true },
      (_progress, token) => client.inspect(common, cwd, token),
    );
    const contract = await pickContract(inspection.contracts, requestedTarget);
    if (!contract) {
      return;
    }
    const solidityFunction = await pickFunction(contract, requestedTarget);
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
        trace: true,
      }, cwd, token),
    );
    lastResult = result;
    writeResult(output, result);
    const runTarget = requestedTarget ?? targetForSelection(document, solidityFunction);
    evidence.update(result, cwd);
    codeLens.update(runTarget, result);
    await decorations.update(result, cwd, runTarget);
    await vscode.commands.executeCommand("echoevm.executionEvidence.focus");
    const verdict = result.execution.status.toUpperCase();
    const action = await vscode.window.showInformationMessage(
      `EchoEVM ${verdict}: ${result.contract}.${result.function} used ${result.execution.gasUsed} gas.`,
      "Show Evidence",
      "Show Trace",
    );
    if (action === "Show Evidence") {
      await vscode.commands.executeCommand("echoevm.executionEvidence.focus");
    } else if (action === "Show Trace") {
      showTracePanel(result);
    }
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    output.error(message);
    output.show(true);
    await vscode.window.showErrorMessage(message);
  }
}

async function pickContract(contracts: SolidityContract[], target?: FunctionTarget): Promise<SolidityContract | undefined> {
  if (contracts.length === 0) {
    throw new Error("solc produced no deployable contracts.");
  }
  const candidates = target
    ? contracts.filter((contract) => contract.functions.some((fn) => functionMatchesTarget(fn, target)))
    : contracts;
  const available = candidates.length > 0 ? candidates : contracts;
  if (available.length === 1) {
    return available[0];
  }
  const selected = await vscode.window.showQuickPick(
    available.map((contract) => ({ label: contract.name, detail: contract.key, contract })),
    { title: "Select a contract to deploy", placeHolder: "Solidity contract" },
  );
  return selected?.contract;
}

async function pickFunction(contract: SolidityContract, target?: FunctionTarget): Promise<SolidityFunction | undefined> {
  if (contract.functions.length === 0) {
    throw new Error(`${contract.name} exposes no ABI functions.`);
  }
  const candidates = target ? contract.functions.filter((fn) => functionMatchesTarget(fn, target)) : contract.functions;
  const available = candidates.length > 0 ? candidates : contract.functions;
  if (available.length === 1) {
    return available[0];
  }
  const selected = await vscode.window.showQuickPick(
    available.map((solidityFunction) => ({
      label: solidityFunction.signature,
      description: solidityFunction.stateMutability,
      detail: outputDescription(solidityFunction),
      solidityFunction,
    })),
    { title: `Select a function from ${contract.name}`, placeHolder: "ABI function" },
  );
  return selected?.solidityFunction;
}

function functionMatchesTarget(fn: SolidityFunction, target: FunctionTarget): boolean {
  if (fn.name !== target.name) return false;
  const location = fn.sourceLocation;
  return !location || (target.offset >= location.start && target.offset <= location.start + location.length);
}

function targetForSelection(document: vscode.TextDocument, fn: SolidityFunction): FunctionTarget {
  const declaration = scanSolidityFunctions(document.getText()).find((candidate) => candidate.name === fn.name);
  return { uri: document.uri.toString(), name: fn.name, offset: declaration?.offset ?? 0 };
}

async function runSetup(toolchain: ToolchainManager, resource?: vscode.Uri, projectRoot?: string): Promise<void> {
  const current = await toolchain.diagnose(resource, projectRoot);
  const choices: string[] = [
    "Show detected versions",
    "Install or update EchoEVM CLI",
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
  } else if (action === "Install or update EchoEVM CLI") {
    await vscode.window.withProgress(
      { location: vscode.ProgressLocation.Notification, title: "Installing EchoEVM CLI" },
      () => toolchain.installEchoEVM(),
    );
    await vscode.window.showInformationMessage("EchoEVM CLI installed successfully.");
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

async function projectRootForResource(resource: vscode.Uri): Promise<string | undefined> {
  if (resource.scheme !== "file" || path.extname(resource.fsPath).toLowerCase() !== ".sol") {
    return undefined;
  }
  const workspaceFolder = vscode.workspace.getWorkspaceFolder(resource);
  return resolveSolidityProjectRoot(resource.fsPath, workspaceFolder?.uri.fsPath);
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
  output.info(`Trace steps: ${result.execution.trace?.length ?? 0}; completed in ${result.durationMs} ms`);
}
