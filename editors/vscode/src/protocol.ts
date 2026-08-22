export const protocolVersion = 1;

export interface SolidityParameter {
  name?: string;
  type: string;
}

export interface SolidityFunction {
  name: string;
  signature: string;
  inputs: SolidityParameter[];
  outputs: SolidityParameter[];
  stateMutability: string;
  sourceLocation?: SourceLocation;
}

export interface SourceLocation {
  file: string;
  start: number;
  length: number;
}

export interface PCSourceLocation extends SourceLocation {
  pc: number;
}

export interface RuntimeSourceMap {
  locations: PCSourceLocation[];
}

export interface SolidityContract {
  key: string;
  name: string;
  constructorInputs: SolidityParameter[];
  functions: SolidityFunction[];
}

export interface CompilerInfo {
  executable: string;
  version: string;
}

export interface InspectResult {
  schemaVersion: number;
  source: string;
  compiler: CompilerInfo;
  durationMs: number;
  contracts: SolidityContract[];
}

export interface TraceStep {
  index: number;
  depth: number;
  address?: string;
  pc: number;
  opcode: string;
  opcodeName: string;
  gasBefore: number;
  gasAfter: number;
  stackBefore: string[];
  stackAfter?: string[];
  haltClass?: string;
}

export interface ExecutionResult {
  engine: string;
  engineVersion: string;
  status: "success" | "revert" | "fault";
  returnData: string;
  gasUsed: number;
  storage: Record<string, string>;
  trace: TraceStep[] | null;
  error?: string;
}

export interface RunResult {
  schemaVersion: number;
  source: string;
  contract: string;
  function: string;
  compiler: CompilerInfo;
  durationMs: number;
  execution: ExecutionResult;
  sourceMap?: RuntimeSourceMap;
}

export interface ErrorResult {
  schemaVersion: number;
  error: {
    code: string;
    message: string;
  };
}

export interface CommonCommandOptions {
  source: string;
  solcPath: string;
  solcArgs?: string[];
  basePath: string;
  includePaths: string[];
  optimize: boolean;
  optimizerRuns: number;
  viaIR: boolean;
  remappings: string[];
}

export interface RunCommandOptions extends CommonCommandOptions {
  contract: string;
  functionSignature: string;
  constructorArgs?: string;
  functionArgs?: string;
  gasLimit: number;
  trace: boolean;
}

export function buildInspectArguments(options: CommonCommandOptions): string[] {
  const args = [
    "solidity",
    "inspect",
    options.source,
    "--format",
    "json",
    "--solc",
    options.solcPath,
    "--base-path",
    options.basePath,
  ];
  appendCompilerPrefix(args, options);
  appendCompilerOptions(args, options);
  return args;
}

export function buildRunArguments(options: RunCommandOptions): string[] {
  const args = [
    "solidity",
    "run",
    options.source,
    "--format",
    "json",
    "--solc",
    options.solcPath,
    "--base-path",
    options.basePath,
    "--contract",
    options.contract,
    "--function",
    options.functionSignature,
    "--gas",
    String(options.gasLimit),
  ];
  appendCompilerPrefix(args, options);
  if (options.constructorArgs) {
    args.push("--constructor-args", options.constructorArgs);
  }
  if (options.functionArgs) {
    args.push("--args", options.functionArgs);
  }
  if (options.trace) {
    args.push("--trace");
  }
  appendCompilerOptions(args, options);
  return args;
}

function appendCompilerPrefix(args: string[], options: CommonCommandOptions): void {
  for (const argument of options.solcArgs ?? []) {
    args.push("--solc-arg", argument);
  }
}

function appendCompilerOptions(args: string[], options: CommonCommandOptions): void {
  for (const includePath of options.includePaths) {
    args.push("--include-path", includePath);
  }
  for (const remapping of options.remappings) {
    args.push("--remapping", remapping);
  }
  if (options.optimize) {
    args.push("--optimize");
  }
  if (options.optimizerRuns > 0) {
    args.push("--optimizer-runs", String(options.optimizerRuns));
  }
  if (options.viaIR) {
    args.push("--via-ir");
  }
}

export function parseProtocolOutput<T extends { schemaVersion: number }>(text: string): T {
  let value: unknown;
  try {
    value = JSON.parse(text.trim());
  } catch (error) {
    throw new Error(`EchoEVM returned invalid JSON: ${error instanceof Error ? error.message : String(error)}`);
  }
  if (!isRecord(value) || value.schemaVersion !== protocolVersion) {
    const actual = isRecord(value) ? String(value.schemaVersion) : "missing";
    throw new Error(`Unsupported EchoEVM protocol version: ${actual}`);
  }
  return value as T;
}

export function isErrorResult(value: unknown): value is ErrorResult {
  return isRecord(value) && isRecord(value.error) && typeof value.error.message === "string";
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}
