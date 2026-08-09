import * as path from "node:path";
import * as vscode from "vscode";
import { buildEvidenceModel, type EvidenceNodeModel } from "./insightModel";
import type { RunResult, SourceLocation } from "./protocol";

interface EvidenceNode {
  model: EvidenceNodeModel;
  children: EvidenceNode[];
}

export class EvidenceTreeProvider implements vscode.TreeDataProvider<EvidenceNode> {
  private readonly changed = new vscode.EventEmitter<EvidenceNode | undefined>();
  private roots: EvidenceNode[] = [];
  private result?: RunResult;
  private basePath?: string;

  public readonly onDidChangeTreeData = this.changed.event;

  public update(result: RunResult, basePath: string): void {
    this.result = result;
    this.basePath = basePath;
    this.roots = buildEvidenceModel(result).map(toNode);
    this.changed.fire(undefined);
  }

  public getTreeItem(node: EvidenceNode): vscode.TreeItem {
    const collapsibleState = node.children.length > 0
      ? vscode.TreeItemCollapsibleState.Expanded
      : vscode.TreeItemCollapsibleState.None;
    const item = new vscode.TreeItem(node.model.label, collapsibleState);
    item.description = node.model.description;
    item.tooltip = [node.model.label, node.model.description].filter(Boolean).join(" — ");
    item.iconPath = new vscode.ThemeIcon(iconName(node.model.icon));
    if (node.model.location && this.result && this.basePath) {
      item.command = {
        command: "echoevm.revealEvidenceLocation",
        title: "Reveal Solidity source",
        arguments: [this.result, this.basePath, node.model.location],
      };
    } else if (node.model.action === "trace") {
      item.command = { command: "echoevm.showLastTrace", title: "Open full opcode trace" };
    }
    return item;
  }

  public getChildren(node?: EvidenceNode): EvidenceNode[] {
    return node?.children ?? this.roots;
  }
}

export async function revealEvidenceLocation(result: RunResult, basePath: string, location: SourceLocation): Promise<void> {
  const filename = path.isAbsolute(location.file) ? location.file : path.resolve(basePath, location.file);
  const document = await vscode.workspace.openTextDocument(vscode.Uri.file(filename));
  const range = byteRange(document, location.start, location.length);
  const editor = await vscode.window.showTextDocument(document, { preview: true, viewColumn: vscode.ViewColumn.One });
  editor.selection = new vscode.Selection(range.start, range.start);
  editor.revealRange(range, vscode.TextEditorRevealType.InCenterIfOutsideViewport);
  const suffix = result.execution.status === "success" ? "executed here" : `${result.execution.status} evidence`;
  await vscode.window.setStatusBarMessage(`EchoEVM: ${suffix} in ${path.basename(filename)}`, 3_000);
}

export function byteRange(document: vscode.TextDocument, start: number, length: number): vscode.Range {
  const contents = Buffer.from(document.getText(), "utf8");
  const safeStart = Math.max(0, Math.min(start, contents.length));
  const safeEnd = Math.max(safeStart, Math.min(start + Math.max(length, 1), contents.length));
  const utf16Start = contents.subarray(0, safeStart).toString("utf8").length;
  const utf16End = contents.subarray(0, safeEnd).toString("utf8").length;
  return new vscode.Range(document.positionAt(utf16Start), document.positionAt(utf16End));
}

function toNode(model: EvidenceNodeModel): EvidenceNode {
  return { model, children: (model.children ?? []).map(toNode) };
}

function iconName(icon: EvidenceNodeModel["icon"]): string {
  switch (icon) {
    case "pass": return "pass-filled";
    case "error": return "error";
    case "warning": return "warning";
    case "gas": return "flame";
    case "state": return "database";
    case "compare": return "compare-changes";
    case "trace": return "list-tree";
    default: return "info";
  }
}
