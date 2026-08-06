#!/usr/bin/env python3
"""Run a correctness-first frozen-evidence trace comprehension benchmark."""

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

from benchmark_lib import load_cases


ROOT = Path(__file__).resolve().parent
PROMPT = "Read TASK.md and EVIDENCE.json, diagnose the execution, and write ANSWER.json in the required schema. Do not modify any other file."
ROOT_CAUSES = ["CALLDATA_OFFSET", "OPERAND_ORDER", "MEMORY_WORD_ALIGNMENT", "RETURN_OFFSET", "HASH_LENGTH", "STORAGE_SLOT", "FRAME_REVERT_ROLLBACK", "INVALID_JUMP_DESTINATION"]
FIXES = ["LOAD_AT_OFFSET_4", "SWAP_OPERANDS", "USE_MSTORE", "RETURN_FROM_OFFSET_0", "HASH_32_BYTES", "USE_STORAGE_SLOT_0", "REMOVE_REVERT_OR_MOVE_WRITE", "TARGET_A_JUMPDEST"]
AGENT_RULES = """# Frozen-evidence benchmark

This is a diagnosis-only benchmark. Use only TASK.md and EVIDENCE.json. Do not
run EVM tools, compile code, access the network, or inspect paths outside this
isolated repository. Write exactly one ANSWER.json file and do not modify the
provided evidence.
"""


def command(args: list[str], cwd: Path) -> subprocess.CompletedProcess[str]:
    return subprocess.run(args, cwd=cwd, text=True, capture_output=True, timeout=60)


def task_text(case: dict[str, object]) -> str:
    return f"""# Diagnose one EVM execution

{case['question']}

Choose one rootCause from: {', '.join(ROOT_CAUSES)}.
Choose one fix from: {', '.join(FIXES)}.

Write ANSWER.json with this exact shape:

```json
{{
  "rootCause": "...",
  "primary": {{"pc": 0, "opcode": "..."}},
  "secondary": null,
  "fix": "...",
  "evidence": "one concise sentence"
}}
```

Use `secondary` with the same pc/opcode shape only when a second instruction is
causally required. PC values are decimal. Do not include markdown in the file.
"""


def prepare_run(run_dir: Path, case: dict[str, object], condition: str) -> None:
    run_dir.mkdir(parents=True)
    (run_dir / "TASK.md").write_text(task_text(case))
    shutil.copy2(ROOT / "fixtures" / str(case["id"]) / f"{condition}.json", run_dir / "EVIDENCE.json")
    (run_dir / "AGENTS.md").write_text(AGENT_RULES)
    (run_dir / ".gitignore").write_text("codex.jsonl\ncodex.stderr\n")
    command(["git", "init", "-q"], run_dir)
    command(["git", "add", "."], run_dir)
    command(["git", "-c", "user.name=EchoEVM Benchmark", "-c", "user.email=benchmark@example.invalid", "commit", "-qm", "fixture"], run_dir)


def usage_from_jsonl(path: Path) -> dict[str, int]:
    usage: dict[str, int] = {}
    for line in path.read_text(errors="replace").splitlines():
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            continue
        if event.get("type") == "turn.completed":
            usage = event.get("usage", usage)
    return {key: int(usage.get(key, 0)) for key in ("input_tokens", "cached_input_tokens", "output_tokens", "reasoning_output_tokens")}


def command_count(path: Path) -> int:
    count = 0
    for line in path.read_text(errors="replace").splitlines():
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            continue
        if event.get("type") == "item.completed" and event.get("item", {}).get("type") == "command_execution":
            count += 1
    return count


def score_answer(answer: object, oracle: dict[str, object]) -> dict[str, object]:
    if not isinstance(answer, dict):
        return {"score": 0, "diagnosis_correct": False, "causal_correct": False, "components": {}}
    primary = answer.get("primary") if isinstance(answer.get("primary"), dict) else {}
    secondary = answer.get("secondary") if isinstance(answer.get("secondary"), dict) else None
    normalize_opcode = lambda value: "SHA3" if str(value).upper() == "KECCAK256" else str(value).upper()
    causal_location = any(
        primary.get("pc") == candidate["pc"] and normalize_opcode(primary.get("opcode", "")) == candidate["opcode"]
        for candidate in oracle.get("acceptedPrimary", [{"pc": oracle["primaryPC"], "opcode": oracle["primaryOpcode"]}])
    )
    checks = {
        "rootCause": answer.get("rootCause") == oracle["rootCause"],
        "primaryPC": primary.get("pc") == oracle["primaryPC"],
        "primaryOpcode": normalize_opcode(primary.get("opcode", "")) == oracle["primaryOpcode"],
        "fix": answer.get("fix") == oracle["fix"],
        "secondary": ((oracle["secondaryPC"] is None and secondary is None) or
                      (secondary is not None and secondary.get("pc") == oracle["secondaryPC"] and
                       normalize_opcode(secondary.get("opcode", "")) == oracle["secondaryOpcode"])),
    }
    weights = {"rootCause": 4, "primaryPC": 2, "primaryOpcode": 1, "fix": 2, "secondary": 1}
    score = sum(weights[key] for key, passed in checks.items() if passed)
    correct = all(checks[key] for key in ("rootCause", "primaryPC", "primaryOpcode", "fix"))
    causal_correct = bool(checks["rootCause"] and checks["fix"] and causal_location)
    return {"score": score, "diagnosis_correct": correct, "causal_correct": causal_correct, "components": {**checks, "causalLocation": causal_location}}


def run_one(spec: dict[str, object], args: argparse.Namespace, output: Path, cases: dict[str, dict[str, object]]) -> dict[str, object]:
    case_id, condition, repetition = str(spec["case"]), str(spec["condition"]), int(spec["repetition"])
    run_id = f"{case_id}-{condition}-{repetition}"
    run_dir = output / "runs" / run_id
    case = cases[case_id]
    prepare_run(run_dir, case, condition)
    env = os.environ.copy()
    jsonl = run_dir / "codex.jsonl"
    stderr_path = run_dir / "codex.stderr"
    started = time.monotonic()
    with jsonl.open("w") as stdout, stderr_path.open("w") as stderr:
        proc = subprocess.run([
            "codex", "exec", "--json", "--ephemeral", "--ignore-user-config",
            "-m", args.model, "-c", f'model_reasoning_effort="{args.reasoning_effort}"',
            "-c", 'skills.config=[{name="echoevm-debug",enabled=false},{name="echoevm-conformance",enabled=false}]',
            "-s", "workspace-write", "-C", str(run_dir), PROMPT,
        ], cwd=run_dir, env=env, text=True, stdout=stdout, stderr=stderr, timeout=args.timeout)
    duration = time.monotonic() - started
    answer_path = run_dir / "ANSWER.json"
    answer: object = None
    answer_error = None
    if answer_path.is_file():
        try:
            answer = json.loads(answer_path.read_text())
        except json.JSONDecodeError as error:
            answer_error = str(error)
    else:
        answer_error = "ANSWER.json missing"
    grading = score_answer(answer, case["oracle"])
    result = {
        "run_id": run_id, "case": case_id, "condition": condition, "repetition": repetition,
        "codex_exit": proc.returncode, "duration_seconds": round(duration, 3),
        "answer_error": answer_error, "answer": answer, **grading,
        "evidence_bytes": (run_dir / "EVIDENCE.json").stat().st_size,
        "command_calls": command_count(jsonl), "usage": usage_from_jsonl(jsonl),
    }
    (run_dir / "result.json").write_text(json.dumps(result, indent=2) + "\n")
    return result


def summarize(results: list[dict[str, object]]) -> dict[str, object]:
    groups = {}
    for condition in sorted({str(row["condition"]) for row in results}):
        rows = [row for row in results if row["condition"] == condition]
        groups[condition] = {
            "runs": len(rows),
            "correct": sum(bool(row["diagnosis_correct"]) for row in rows),
            "accuracy": sum(bool(row["diagnosis_correct"]) for row in rows) / len(rows),
            "causal_correct": sum(bool(row["causal_correct"]) for row in rows),
            "causal_accuracy": sum(bool(row["causal_correct"]) for row in rows) / len(rows),
            "median_score": statistics.median(float(row["score"]) for row in rows),
            "median_duration_seconds": statistics.median(float(row["duration_seconds"]) for row in rows),
            "median_noncached_tokens": statistics.median(int(row["usage"]["input_tokens"]) - int(row["usage"]["cached_input_tokens"]) + int(row["usage"]["output_tokens"]) for row in rows),
            "median_output_tokens": statistics.median(int(row["usage"]["output_tokens"]) for row in rows),
            "median_command_calls": statistics.median(int(row["command_calls"]) for row in rows),
            "median_evidence_bytes": statistics.median(int(row["evidence_bytes"]) for row in rows),
        }
    return {"schema": "echoevm.trace-value-results.v1", "groups": groups, "results": results}


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", type=Path, default=Path("/private/tmp/echoevm-trace-value-results"))
    parser.add_argument("--cases", default="")
    parser.add_argument("--conditions", default="control,raw,echo,evidence")
    parser.add_argument("--repetitions", type=int, default=3)
    parser.add_argument("--jobs", type=int, default=2)
    parser.add_argument("--seed", type=int, default=20260806)
    parser.add_argument("--model", default="gpt-5.6-sol")
    parser.add_argument("--reasoning-effort", default="medium")
    parser.add_argument("--timeout", type=int, default=600)
    args = parser.parse_args()
    output = args.output.resolve()
    if output.exists():
        raise SystemExit(f"output already exists: {output}")
    (output / "runs").mkdir(parents=True)
    case_rows = load_cases(ROOT / "cases.json")
    cases = {row["id"]: row for row in case_rows}
    selected = [item for item in args.cases.split(",") if item] or [
        str(row["id"]) for row in case_rows if "variant" not in row
    ]
    conditions = [item for item in args.conditions.split(",") if item]
    plan = [{"case": case, "condition": condition, "repetition": repetition}
            for case in selected for condition in conditions for repetition in range(1, args.repetitions + 1)]
    random.Random(args.seed).shuffle(plan)
    (output / "plan.json").write_text(json.dumps(plan, indent=2) + "\n")
    codex_version = subprocess.run(["codex", "--version"], text=True, capture_output=True, timeout=30)
    (output / "RUN_METADATA.json").write_text(json.dumps({
        "schema": "echoevm.trace-value-run.v1", "model": args.model,
        "reasoningEffort": args.reasoning_effort, "seed": args.seed,
        "repetitions": args.repetitions, "jobs": args.jobs,
        "conditions": conditions, "cases": selected,
        "codexVersion": codex_version.stdout.strip(),
        "fixtureManifest": json.loads((ROOT / "fixtures" / "MANIFEST.json").read_text()),
    }, indent=2) + "\n")
    results = []
    with concurrent.futures.ThreadPoolExecutor(max_workers=args.jobs) as executor:
        futures = {executor.submit(run_one, spec, args, output, cases): spec for spec in plan}
        for future in concurrent.futures.as_completed(futures):
            result = future.result()
            results.append(result)
            print(json.dumps({key: result[key] for key in ("run_id", "score", "diagnosis_correct", "duration_seconds")}), flush=True)
            (output / "partial-results.json").write_text(json.dumps(results, indent=2) + "\n")
    summary = summarize(results)
    (output / "summary.json").write_text(json.dumps(summary, indent=2) + "\n")
    print(json.dumps(summary["groups"], indent=2))


if __name__ == "__main__":
    main()
