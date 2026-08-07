#!/usr/bin/env python3
"""Run the compiled-Solidity nested-evidence benchmark."""

from __future__ import annotations

import argparse
import concurrent.futures
import json
import os
from pathlib import Path
import random
import shutil
import statistics
import subprocess
import time


ROOT = Path(__file__).resolve().parent
PROMPT = "Read TASK.md and EVIDENCE.json, diagnose the execution, and write ANSWER.json. Do not modify any other file."
ROOT_CAUSES = ["IGNORED_LOW_LEVEL_CALL_FAILURE", "SWALLOWED_CREATE_FAILURE", "DELEGATECALL_STORAGE_CONTEXT", "WRONG_DIVISOR"]
FIXES = ["REQUIRE_CALL_SUCCESS", "PROPAGATE_CREATE_REVERT", "USE_CALL_NOT_DELEGATECALL", "DIVIDE_BY_LENGTH"]
AGENT_RULES = """# Frozen Solidity evidence benchmark

Use only TASK.md and EVIDENCE.json. Do not run tools, compile code, access the
network, or inspect paths outside this isolated repository. Write ANSWER.json
and do not modify the evidence.
"""


def task_text(case: dict[str, object]) -> str:
    source = (ROOT / str(case["source"])).read_text()
    return f"""# Diagnose one compiled Solidity execution

{case['question']}

Executed `{case['contract']}.{case['function']}` with arguments `{case['args']}`.

```solidity
{source}
```

Choose rootCause from: {', '.join(ROOT_CAUSES)}.
Choose fix from: {', '.join(FIXES)}.

Write exactly:

```json
{{"rootCause":"...","primary":{{"depth":0,"pc":0,"opcode":"..."}},"secondary":{{"depth":0,"pc":0,"opcode":"..."}},"fix":"...","evidence":"one concise sentence"}}
```

Depth and PC are decimal. Do not include markdown in ANSWER.json.
"""


def command(args: list[str], cwd: Path) -> subprocess.CompletedProcess[str]:
    return subprocess.run(args, cwd=cwd, text=True, capture_output=True, timeout=60)


def prepare_run(run_dir: Path, case: dict[str, object], condition: str) -> None:
    run_dir.mkdir(parents=True)
    (run_dir / "TASK.md").write_text(task_text(case))
    shutil.copy2(ROOT / "fixtures" / str(case["id"]) / f"{condition}.json", run_dir / "EVIDENCE.json")
    (run_dir / "AGENTS.md").write_text(AGENT_RULES)
    (run_dir / ".gitignore").write_text("codex.jsonl\ncodex.stderr\n")
    command(["git", "init", "-q"], run_dir)
    command(["git", "add", "."], run_dir)
    command(["git", "-c", "user.name=EchoEVM Benchmark", "-c", "user.email=benchmark@example.invalid", "commit", "-qm", "fixture"], run_dir)


def usage(path: Path) -> dict[str, int]:
    result: dict[str, int] = {}
    for line in path.read_text(errors="replace").splitlines():
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            continue
        if event.get("type") == "turn.completed":
            result = event.get("usage", result)
    return {key: int(result.get(key, 0)) for key in ("input_tokens", "cached_input_tokens", "output_tokens", "reasoning_output_tokens")}


def score(answer: object, oracle: dict[str, object]) -> dict[str, object]:
    if not isinstance(answer, dict):
        return {"score": 0, "diagnosis_correct": False, "components": {}}
    checks = {"rootCause": answer.get("rootCause") == oracle["rootCause"], "fix": answer.get("fix") == oracle["fix"]}
    for name in ("primary", "secondary"):
        actual = answer.get(name) if isinstance(answer.get(name), dict) else {}
        expected = oracle[name]
        checks[name] = actual.get("depth") == expected["depth"] and actual.get("pc") == expected["pc"] and str(actual.get("opcode", "")).upper() == expected["opcode"]
    weights = {"rootCause": 4, "primary": 2, "secondary": 2, "fix": 2}
    value = sum(weights[key] for key, passed in checks.items() if passed)
    return {"score": value, "diagnosis_correct": all(checks.values()), "components": checks}


def run_one(spec: dict[str, object], args: argparse.Namespace, output: Path, cases: dict[str, dict[str, object]]) -> dict[str, object]:
    case_id, condition, repetition = str(spec["case"]), str(spec["condition"]), int(spec["repetition"])
    run_id = f"{case_id}-{condition}-{repetition}"
    run_dir = output / "runs" / run_id
    case = cases[case_id]
    prepare_run(run_dir, case, condition)
    jsonl, stderr_path = run_dir / "codex.jsonl", run_dir / "codex.stderr"
    started = time.monotonic()
    with jsonl.open("w") as stdout, stderr_path.open("w") as stderr:
        proc = subprocess.run([
            "codex", "exec", "--json", "--ephemeral", "--ignore-user-config", "-m", args.model,
            "-c", f'model_reasoning_effort="{args.reasoning_effort}"',
            "-c", 'skills.config=[{name="echoevm-debug",enabled=false},{name="echoevm-conformance",enabled=false}]',
            "-s", "workspace-write", "-C", str(run_dir), PROMPT,
        ], cwd=run_dir, env=os.environ.copy(), text=True, stdout=stdout, stderr=stderr, timeout=args.timeout)
    answer: object = None
    answer_error = None
    answer_path = run_dir / "ANSWER.json"
    if answer_path.is_file():
        try:
            answer = json.loads(answer_path.read_text())
        except json.JSONDecodeError as error:
            answer_error = str(error)
    else:
        answer_error = "ANSWER.json missing"
    grading = score(answer, case["oracle"])
    result = {
        "run_id": run_id, "case": case_id, "condition": condition, "repetition": repetition,
        "codex_exit": proc.returncode, "duration_seconds": round(time.monotonic() - started, 3),
        "answer": answer, "answer_error": answer_error, **grading,
        "evidence_bytes": (run_dir / "EVIDENCE.json").stat().st_size, "usage": usage(jsonl),
    }
    (run_dir / "result.json").write_text(json.dumps(result, indent=2) + "\n")
    return result


def summarize(results: list[dict[str, object]]) -> dict[str, object]:
    groups = {}
    for condition in sorted({str(row["condition"]) for row in results}):
        rows = [row for row in results if row["condition"] == condition]
        groups[condition] = {
            "runs": len(rows), "correct": sum(bool(row["diagnosis_correct"]) for row in rows),
            "accuracy": sum(bool(row["diagnosis_correct"]) for row in rows) / len(rows),
            "medianScore": statistics.median(float(row["score"]) for row in rows),
            "medianFreshTokens": statistics.median(int(row["usage"]["input_tokens"]) - int(row["usage"]["cached_input_tokens"]) + int(row["usage"]["output_tokens"]) for row in rows),
            "medianDurationSeconds": statistics.median(float(row["duration_seconds"]) for row in rows),
            "medianEvidenceBytes": statistics.median(int(row["evidence_bytes"]) for row in rows),
        }
    return {"schema": "echoevm.trace-value-v2-results.v1", "groups": groups, "results": results}


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", type=Path, default=Path("/private/tmp/echoevm-trace-value-v2"))
    parser.add_argument("--cases", default="")
    parser.add_argument("--conditions", default="control,broad,evidence")
    parser.add_argument("--repetitions", type=int, default=3)
    parser.add_argument("--jobs", type=int, default=4)
    parser.add_argument("--seed", type=int, default=20260806)
    parser.add_argument("--model", default="gpt-5.6-sol")
    parser.add_argument("--reasoning-effort", default="medium")
    parser.add_argument("--timeout", type=int, default=600)
    args = parser.parse_args()
    output = args.output.resolve()
    if output.exists():
        raise SystemExit(f"output already exists: {output}")
    (output / "runs").mkdir(parents=True)
    case_rows = json.loads((ROOT / "cases.json").read_text())
    cases = {row["id"]: row for row in case_rows}
    selected = [item for item in args.cases.split(",") if item] or list(cases)
    conditions = [item for item in args.conditions.split(",") if item]
    plan = [{"case": case, "condition": condition, "repetition": repetition} for case in selected for condition in conditions for repetition in range(1, args.repetitions + 1)]
    random.Random(args.seed).shuffle(plan)
    (output / "plan.json").write_text(json.dumps(plan, indent=2) + "\n")
    (output / "RUN_METADATA.json").write_text(json.dumps({
        "schema": "echoevm.trace-value-v2-run.v1", "model": args.model, "reasoningEffort": args.reasoning_effort,
        "seed": args.seed, "repetitions": args.repetitions, "jobs": args.jobs, "conditions": conditions,
        "cases": selected, "codexVersion": subprocess.run(["codex", "--version"], text=True, capture_output=True, timeout=30).stdout.strip(),
        "fixtureManifest": json.loads((ROOT / "fixtures" / "MANIFEST.json").read_text()),
    }, indent=2) + "\n")
    results = []
    with concurrent.futures.ThreadPoolExecutor(max_workers=args.jobs) as executor:
        futures = [executor.submit(run_one, spec, args, output, cases) for spec in plan]
        for future in concurrent.futures.as_completed(futures):
            result = future.result()
            results.append(result)
            print(json.dumps({key: result[key] for key in ("run_id", "score", "diagnosis_correct", "duration_seconds")}), flush=True)
    summary = summarize(results)
    (output / "summary.json").write_text(json.dumps(summary, indent=2) + "\n")
    print(json.dumps(summary["groups"], indent=2))


if __name__ == "__main__":
    main()
