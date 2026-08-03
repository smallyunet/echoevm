import * as fs from "node:fs";
import * as path from "node:path";
import solc from "solc";
import { readUtf8 } from "./stdin";

async function main(): Promise<void> {
  const args = process.argv.slice(2);
  if (args.includes("--version")) {
    process.stdout.write(`${solc.version()}\n`);
    return;
  }
  if (!args.includes("--standard-json")) {
    throw new Error("Bundled EchoEVM solc-js only supports --version and --standard-json.");
  }

  const basePath = optionValues(args, "--base-path").at(-1) ?? process.cwd();
  const includePaths = optionValues(args, "--include-path");
  const input = await readUtf8(process.stdin);
  const output = solc.compile(input, {
    import: (sourcePath: string): { contents?: string; error?: string } => {
      for (const prefix of [basePath, ...includePaths]) {
        const candidate = path.resolve(prefix, sourcePath);
        try {
          return { contents: fs.readFileSync(candidate, "utf8") };
        } catch {
          // Try the next explicitly allowed compiler search path.
        }
      }
      return { error: `File not found inside the base path or include paths: ${sourcePath}` };
    },
  });
  process.stdout.write(`${output}\n`);
}

void main().catch((error: unknown) => {
  process.stderr.write(`${error instanceof Error ? error.message : String(error)}\n`);
  process.exitCode = 1;
});

function optionValues(argv: string[], name: string): string[] {
  const values: string[] = [];
  for (let index = 0; index < argv.length - 1; index += 1) {
    if (argv[index] === name) {
      values.push(argv[index + 1]);
      index += 1;
    }
  }
  return values;
}
