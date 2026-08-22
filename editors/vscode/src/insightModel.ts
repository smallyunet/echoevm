import type { PCSourceLocation, RunResult, SourceLocation, TraceStep } from "./protocol";

export interface EvidenceNodeModel {
  label: string;
  description?: string;
  icon: "pass" | "error" | "warning" | "gas" | "state" | "compare" | "trace" | "info";
  location?: SourceLocation;
  action?: "trace";
  children?: EvidenceNodeModel[];
}

export function buildEvidenceModel(result: RunResult): EvidenceNodeModel[] {
  const statusIcon = result.execution.status === "success" ? "pass" : "error";
  const nodes: EvidenceNodeModel[] = [{
    label: result.execution.status === "success" ? "Execution succeeded" : `Execution ${result.execution.status}`,
    description: `${result.contract}.${result.function}`,
    icon: statusIcon,
    location: terminalSourceLocation(result),
  }, {
    label: "Gas used",
    description: result.execution.gasUsed.toLocaleString("en-US"),
    icon: "gas",
  }];

  if (result.execution.error) {
    nodes.push({ label: "Execution error", description: result.execution.error, icon: "error", location: terminalSourceLocation(result) });
  }

  const storage = Object.entries(result.execution.storage);
  nodes.push({
    label: "Storage after execution",
    description: `${storage.length} ${storage.length === 1 ? "slot" : "slots"}`,
    icon: "state",
    children: storage.length > 0
      ? storage.map(([slot, value]) => ({ label: compactHex(slot), description: compactHex(value), icon: "state" }))
      : [{ label: "No observed storage values", icon: "info" }],
  });

  const keySteps = result.execution.trace?.filter(isKeyStep) ?? [];
  nodes.push({
    label: "Key execution steps",
    description: `${keySteps.length} selected / ${result.execution.trace?.length ?? 0} total`,
    icon: "trace",
    action: "trace",
    children: keySteps.slice(0, 40).map((step) => ({
      label: `${step.index}. ${step.opcodeName}`,
      description: `depth ${step.depth} · pc ${step.pc} · gas ${step.gasBefore.toLocaleString("en-US")}`,
      icon: step.haltClass && step.haltClass !== "success" ? "error" : "trace",
      location: locationForPC(result, step.pc),
    })),
  });
  nodes.push({ label: "Open full opcode trace", description: "Detailed stack and gas table", icon: "trace", action: "trace" });
  return nodes;
}

export function terminalSourceLocation(result: RunResult): SourceLocation | undefined {
  const terminal = result.execution.trace?.at(-1);
  return locationForPC(result, terminal?.pc);
}

export function locationForPC(result: RunResult, pc?: number): PCSourceLocation | undefined {
  if (pc === undefined) return undefined;
  return result.sourceMap?.locations.find((location) => location.pc === pc);
}

function isKeyStep(step: TraceStep): boolean {
  return /^(CALL|CALLCODE|DELEGATECALL|STATICCALL|CREATE|CREATE2|SLOAD|SSTORE|TLOAD|TSTORE|LOG[0-4]|RETURN|REVERT|INVALID|SELFDESTRUCT)$/u.test(step.opcodeName)
    || Boolean(step.haltClass && step.haltClass !== "success");
}

function compactHex(value: string): string {
  return value.length <= 22 ? value : `${value.slice(0, 12)}…${value.slice(-8)}`;
}
