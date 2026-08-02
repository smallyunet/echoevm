import { randomBytes } from "node:crypto";
import * as vscode from "vscode";
import type { RunResult } from "./protocol";

export function showTracePanel(result: RunResult): void {
  const panel = vscode.window.createWebviewPanel(
    "echoevm.trace",
    `EchoEVM Trace: ${result.contract}.${result.function}`,
    vscode.ViewColumn.Beside,
    { enableScripts: false },
  );
  panel.webview.html = renderTraceHTML(result);
}

function renderTraceHTML(result: RunResult): string {
  const nonce = randomNonce();
  const trace = result.execution.trace ?? [];
  const rows = trace.map((step) => `
    <tr>
      <td>${step.index}</td>
      <td>${step.depth}</td>
      <td>${step.pc}</td>
      <td><code>${escapeHTML(step.opcodeName)}</code></td>
      <td>${step.gasBefore}</td>
      <td><code>${escapeHTML(step.stackBefore.join(" "))}</code></td>
    </tr>`).join("");
  const comparison = result.comparison;
  const verdict = comparison
    ? `<span class="badge ${comparison.match ? "match" : "divergence"}">${comparison.match ? "MATCH" : "DIVERGENCE"}</span>`
    : "";
  const divergence = comparison?.firstDivergence
    ? `<section><h2>First divergence</h2><pre>${escapeHTML(JSON.stringify(comparison.firstDivergence, null, 2))}</pre></section>`
    : "";
  const empty = trace.length === 0 ? "<p>No trace was returned. Run the function again with trace collection enabled.</p>" : "";
  return `<!doctype html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'nonce-${nonce}';">
  <style nonce="${nonce}">
    body { color: var(--vscode-foreground); background: var(--vscode-editor-background); font-family: var(--vscode-font-family); padding: 16px; }
    h1 { font-size: 18px; margin: 0 0 8px; }
    h2 { font-size: 15px; }
    .meta { color: var(--vscode-descriptionForeground); margin-bottom: 16px; }
    .badge { border-radius: 999px; display: inline-block; font-size: 11px; font-weight: 700; margin-left: 8px; padding: 2px 8px; }
    .match { background: var(--vscode-testing-iconPassed); color: var(--vscode-editor-background); }
    .divergence { background: var(--vscode-testing-iconFailed); color: var(--vscode-editor-background); }
    .table-wrap { overflow: auto; max-height: 72vh; border: 1px solid var(--vscode-panel-border); }
    table { border-collapse: collapse; width: 100%; font-size: 12px; }
    th { position: sticky; top: 0; background: var(--vscode-editorWidget-background); text-align: left; }
    th, td { border-bottom: 1px solid var(--vscode-panel-border); padding: 6px 8px; vertical-align: top; }
    td:last-child { min-width: 420px; word-break: break-all; }
    code, pre { font-family: var(--vscode-editor-font-family); }
    pre { background: var(--vscode-textCodeBlock-background); overflow: auto; padding: 12px; }
  </style>
</head>
<body>
  <h1>${escapeHTML(result.contract)}.${escapeHTML(result.function)} ${verdict}</h1>
  <div class="meta">${escapeHTML(result.execution.engine)} ${escapeHTML(result.execution.engineVersion)} · ${result.execution.status} · ${result.execution.gasUsed} gas · ${result.durationMs} ms</div>
  ${divergence}
  ${empty}
  ${trace.length > 0 ? `<div class="table-wrap"><table><thead><tr><th>#</th><th>Depth</th><th>PC</th><th>Opcode</th><th>Gas before</th><th>Stack before</th></tr></thead><tbody>${rows}</tbody></table></div>` : ""}
</body>
</html>`;
}

function escapeHTML(value: string): string {
  return value.replace(/[&<>'"]/g, (character) => ({
    "&": "&amp;",
    "<": "&lt;",
    ">": "&gt;",
    "'": "&#39;",
    '"': "&quot;",
  })[character] ?? character);
}

function randomNonce(): string {
  return randomBytes(16).toString("base64");
}
