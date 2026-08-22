import * as path from "node:path";
import * as vscode from "vscode";
import { byteRange } from "./evidenceView";
import { terminalSourceLocation } from "./insightModel";
import type { RunResult, SourceLocation } from "./protocol";
import { scanSolidityFunctions } from "./solidityScanner";

export interface FunctionTarget {
  uri: string;
  name: string;
  offset: number;
}

interface LastRun {
  target: FunctionTarget;
  result: RunResult;
}

export class SolidityCodeLensProvider implements vscode.CodeLensProvider {
  private readonly changed = new vscode.EventEmitter<void>();
  private lastRun?: LastRun;

  public readonly onDidChangeCodeLenses = this.changed.event;

  public update(target: FunctionTarget, result: RunResult): void {
    this.lastRun = { target, result };
    this.changed.fire();
  }

  public refresh(): void {
    this.changed.fire();
  }

  public provideCodeLenses(document: vscode.TextDocument): vscode.CodeLens[] {
    if (!vscode.workspace.getConfiguration("echoevm", document.uri).get<boolean>("codeLens", true)) {
      return [];
    }
    const lenses: vscode.CodeLens[] = [];
    for (const declaration of scanSolidityFunctions(document.getText())) {
      const position = document.positionAt(declaration.offset);
      const range = new vscode.Range(position, position);
      const target: FunctionTarget = { uri: document.uri.toString(), name: declaration.name, offset: declaration.offset };
      lenses.push(new vscode.CodeLens(range, {
        title: "$(play) EchoEVM Run",
        command: "echoevm.runAtFunction",
        tooltip: `Execute ${declaration.name} with EchoEVM`,
        arguments: [target],
      }));
      if (this.lastRun?.target.uri === document.uri.toString() && this.lastRun.target.offset === declaration.offset) {
        const result = this.lastRun.result;
        const status = result.execution.status.toUpperCase();
        lenses.push(new vscode.CodeLens(range, {
          title: `$(pulse) Last: ${status} · ${result.execution.gasUsed.toLocaleString("en-US")} gas`,
          command: "echoevm.executionEvidence.focus",
          tooltip: "Focus the latest EchoEVM execution evidence",
        }));
      }
    }
    return lenses;
  }
}

export class ExecutionDecorationManager implements vscode.Disposable {
  private readonly success = vscode.window.createTextEditorDecorationType({
    isWholeLine: true,
    overviewRulerColor: new vscode.ThemeColor("testing.iconPassed"),
    overviewRulerLane: vscode.OverviewRulerLane.Right,
  });
  private readonly failure = vscode.window.createTextEditorDecorationType({
    isWholeLine: true,
    overviewRulerColor: new vscode.ThemeColor("testing.iconFailed"),
    overviewRulerLane: vscode.OverviewRulerLane.Right,
  });
  private readonly diagnostics = vscode.languages.createDiagnosticCollection("echoevm");
  private decoratedEditors = new Set<vscode.TextEditor>();

  public async update(result: RunResult, basePath: string, target: FunctionTarget): Promise<void> {
    this.clear();
    this.diagnostics.clear();
    const fallbackDocument = await vscode.workspace.openTextDocument(vscode.Uri.parse(target.uri));
    if (!vscode.workspace.getConfiguration("echoevm", fallbackDocument.uri).get<boolean>("inlineResults", true)) {
      return;
    }
    const fallbackPosition = fallbackDocument.positionAt(target.offset);
    const fallback = new vscode.Range(fallbackPosition, fallbackPosition);
    const terminal = result.execution.status === "success" ? undefined : terminalSourceLocation(result);
    const resolved = terminal ? await resolveSourceRange(basePath, terminal) : undefined;
    const document = resolved?.document ?? fallbackDocument;
    const range = resolved?.range ?? fallback;
    const editor = vscode.window.visibleTextEditors.find((candidate) => candidate.document.uri.toString() === document.uri.toString());
    const status = result.execution.status.toUpperCase();
    const message = `EchoEVM: ${status} · ${result.execution.gasUsed.toLocaleString("en-US")} gas`;
    const hover = new vscode.MarkdownString(undefined, true);
    hover.appendMarkdown(`**${message}**\n\n`);
    hover.appendMarkdown(`Function: \`${result.contract}.${result.function}\`  \n`);
    hover.appendMarkdown(`Storage values observed: ${Object.keys(result.execution.storage).length}  \n`);
    if (result.execution.error) hover.appendMarkdown(`Error: \`${result.execution.error}\``);
    if (editor) {
      const decoration: vscode.DecorationOptions = {
        range,
        hoverMessage: hover,
        renderOptions: { after: {
          contentText: `  ${message}`,
          color: new vscode.ThemeColor(result.execution.status === "success" ? "testing.iconPassed" : "testing.iconFailed"),
          fontStyle: "italic",
        } },
      };
      editor.setDecorations(result.execution.status === "success" ? this.success : this.failure, [decoration]);
      this.decoratedEditors.add(editor);
    }
    if (result.execution.status !== "success") {
      const diagnostic = new vscode.Diagnostic(
        range,
        `${result.contract}.${result.function} ${result.execution.status}${result.execution.error ? `: ${result.execution.error}` : ""}`,
        vscode.DiagnosticSeverity.Error,
      );
      diagnostic.source = "EchoEVM execution";
      diagnostic.code = result.execution.status;
      this.diagnostics.set(document.uri, [diagnostic]);
    }
  }

  public dispose(): void {
    this.clear();
    this.success.dispose();
    this.failure.dispose();
    this.diagnostics.dispose();
  }

  public clear(): void {
    for (const editor of this.decoratedEditors) {
      editor.setDecorations(this.success, []);
      editor.setDecorations(this.failure, []);
    }
    this.decoratedEditors.clear();
    this.diagnostics.clear();
  }
}

async function resolveSourceRange(basePath: string, location: SourceLocation): Promise<{ document: vscode.TextDocument; range: vscode.Range } | undefined> {
  try {
    const filename = path.isAbsolute(location.file) ? location.file : path.resolve(basePath, location.file);
    const document = await vscode.workspace.openTextDocument(vscode.Uri.file(filename));
    return { document, range: byteRange(document, location.start, location.length) };
  } catch {
    return undefined;
  }
}
