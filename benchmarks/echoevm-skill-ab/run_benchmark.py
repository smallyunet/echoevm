#!/usr/bin/env python3
"""Run a correctness-gated EchoEVM skill A/B benchmark."""

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
DEFAULT_TASKS = ["fee_quote", "packed_decoder", "sum_to", "create2_factory"]
PROMPT = "Read TASK.md and implement the requested fix. Work autonomously, keep the patch focused, run relevant validation, and do not commit."
AGENT_RULES = """# Benchmark toolchain

For reproducibility, every benchmark-provided executable must be invoked by its
explicit task-local path:

- `./.benchmark-bin/solc`
- `./.benchmark-bin/forge`
- `./.benchmark-bin/echoevm`

Never invoke `solc`, `forge`, or `echoevm` as an unqualified command because the
login shell may resolve a different installed version. If the task-local
`echoevm` reports that it is unavailable, continue with the other task-local
tools.
"""


def command(args: list[str], cwd: Path, env: dict[str, str] | None = None, timeout: int = 1200) -> subprocess.CompletedProcess[str]:
    return subprocess.run(args, cwd=cwd, env=env, text=True, capture_output=True, timeout=timeout)


def prepare_run(run_dir: Path, task: str) -> None:
    shutil.copytree(ROOT / "tasks" / task, run_dir)
    (run_dir / "AGENTS.md").write_text(AGENT_RULES)
    (run_dir / ".gitignore").write_text(".benchmark-bin/\ncodex.jsonl\ncodex.stderr\ncache/\nout/\n")
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
    return {key: int(usage.get(key, 0)) for key in (
        "input_tokens", "cached_input_tokens", "output_tokens", "reasoning_output_tokens"
    )}


def final_message_from_jsonl(path: Path) -> str:
    messages: list[str] = []
    for line in path.read_text(errors="replace").splitlines():
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            continue
        if event.get("type") == "item.completed":
            item = event.get("item", {})
            if item.get("type") == "agent_message":
                messages.append(item.get("text", ""))
    return messages[-1] if messages else ""


def run_one(spec: dict[str, object], args: argparse.Namespace, output: Path) -> dict[str, object]:
    task = str(spec["task"])
    condition = str(spec["condition"])
    repetition = int(spec["repetition"])
    run_id = f"{task}-{condition}-{repetition}"
    run_dir = output / "runs" / run_id
    prepare_run(run_dir, task)

    tools_dir = run_dir / ".benchmark-bin"
    tools_dir.mkdir()
    shutil.copy2(args.solc, tools_dir / "solc")
    shutil.copy2(args.forge, tools_dir / "forge")
    if condition == "skill":
        shutil.copy2(args.echoevm, tools_dir / "echoevm")
    else:
        unavailable = tools_dir / "echoevm"
        unavailable.write_text("#!/bin/sh\necho 'echoevm unavailable in control condition' >&2\nexit 127\n")
        unavailable.chmod(0o755)

    env = os.environ.copy()
    env["PATH"] = str(tools_dir) + os.pathsep + env.get("PATH", "")
    env["FOUNDRY_SOLC"] = str(tools_dir / "solc")
    skill_config = (
        'skills.config=[{name="echoevm-debug",enabled=true},{name="echoevm-conformance",enabled=false}]'
        if condition == "skill"
        else 'skills.config=[{name="echoevm-debug",enabled=false},{name="echoevm-conformance",enabled=false}]'
    )
    jsonl = run_dir / "codex.jsonl"
    stderr_path = run_dir / "codex.stderr"
    started = time.monotonic()
    with jsonl.open("w") as stdout, stderr_path.open("w") as stderr:
        proc = subprocess.run([
            "codex", "exec", "--json", "--ephemeral", "--ignore-user-config",
            "-m", args.model, "-c", f'model_reasoning_effort="{args.reasoning_effort}"',
            "-c", skill_config, "-s", "workspace-write", "-C", str(run_dir), PROMPT,
        ], cwd=run_dir, env=env, text=True, stdout=stdout, stderr=stderr, timeout=args.timeout)
    duration = time.monotonic() - started

    hidden_target = run_dir / "test" / "Hidden.t.sol"
    hidden_target.parent.mkdir(exist_ok=True)
    shutil.copy2(ROOT / "hidden-tests" / task / "Hidden.t.sol", hidden_target)
    grade = command([str(tools_dir / "forge"), "test", "--root", str(run_dir), "-vv"], run_dir, env=env, timeout=300)
    (run_dir / "grade.stdout").write_text(grade.stdout)
    (run_dir / "grade.stderr").write_text(grade.stderr)
    patch = command(["git", "diff", "--", "src"], run_dir)
    (run_dir / "patch.diff").write_text(patch.stdout)
    final = final_message_from_jsonl(jsonl)
    (run_dir / "final.txt").write_text(final)
    transcript = jsonl.read_text(errors="replace")
    pinned_version_verified = (
        ".benchmark-bin/echoevm version --json" in transcript
        and "version" in transcript
        and "v0.0.37" in transcript
    )
    result: dict[str, object] = {
        "run_id": run_id,
        "task": task,
        "condition": condition,
        "repetition": repetition,
        "codex_exit": proc.returncode,
        "passed": grade.returncode == 0,
        "grade_exit": grade.returncode,
        "duration_seconds": round(duration, 3),
        "echoevm_attempted": "echoevm " in transcript or "echoevm\"" in transcript,
        "skill_read": "echoevm-debug/SKILL.md" in transcript,
        "pinned_echoevm_version_verified": pinned_version_verified,
        "treatment_compliant": condition == "control" or (
            "echoevm-debug/SKILL.md" in transcript and pinned_version_verified
        ),
        "usage": usage_from_jsonl(jsonl),
    }
    (run_dir / "result.json").write_text(json.dumps(result, indent=2) + "\n")
    return result


def summarize(results: list[dict[str, object]]) -> dict[str, object]:
    groups: dict[str, dict[str, object]] = {}
    for condition in sorted({str(result["condition"]) for result in results}):
        rows = [result for result in results if result["condition"] == condition]
        passed = [result for result in rows if result["passed"]]
        totals = [int(row["usage"]["input_tokens"]) + int(row["usage"]["output_tokens"]) for row in passed]
        groups[condition] = {
            "runs": len(rows),
            "passed": len(passed),
            "pass_rate": len(passed) / len(rows) if rows else 0,
            "median_duration_seconds_successes": statistics.median([float(row["duration_seconds"]) for row in passed]) if passed else None,
            "median_total_tokens_successes": statistics.median(totals) if totals else None,
            "median_noncached_input_successes": statistics.median([
                int(row["usage"]["input_tokens"]) - int(row["usage"]["cached_input_tokens"]) for row in passed
            ]) if passed else None,
            "echoevm_attempts": sum(bool(row["echoevm_attempted"]) for row in rows),
            "skill_reads": sum(bool(row["skill_read"]) for row in rows),
        }
    return {"groups": groups, "results": results}


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--echoevm", type=Path, required=True)
    parser.add_argument("--solc", type=Path, required=True)
    parser.add_argument("--forge", type=Path, required=True)
    parser.add_argument("--output", type=Path, default=Path("/private/tmp/echoevm-skill-ab-results"))
    parser.add_argument("--tasks", default=",".join(DEFAULT_TASKS))
    parser.add_argument("--conditions", default="control,skill")
    parser.add_argument("--repetitions", type=int, default=3)
    parser.add_argument("--jobs", type=int, default=2)
    parser.add_argument("--seed", type=int, default=20260803)
    parser.add_argument("--model", default="gpt-5.6-sol")
    parser.add_argument("--reasoning-effort", default="medium")
    parser.add_argument("--timeout", type=int, default=1200)
    args = parser.parse_args()
    args.echoevm = args.echoevm.resolve()
    args.solc = args.solc.expanduser().resolve()
    args.forge = args.forge.expanduser().resolve()
    output = args.output.resolve()
    if output.exists():
        raise SystemExit(f"output already exists: {output}")
    (output / "runs").mkdir(parents=True)
    tasks = [item for item in args.tasks.split(",") if item]
    conditions = [item for item in args.conditions.split(",") if item]
    plan = [
        {"task": task, "condition": condition, "repetition": repetition}
        for task in tasks for condition in conditions for repetition in range(1, args.repetitions + 1)
    ]
    random.Random(args.seed).shuffle(plan)
    (output / "plan.json").write_text(json.dumps(plan, indent=2) + "\n")
    results: list[dict[str, object]] = []
    with concurrent.futures.ThreadPoolExecutor(max_workers=args.jobs) as executor:
        futures = {executor.submit(run_one, spec, args, output): spec for spec in plan}
        for future in concurrent.futures.as_completed(futures):
            result = future.result()
            results.append(result)
            print(json.dumps(result), flush=True)
            (output / "partial-results.json").write_text(json.dumps(results, indent=2) + "\n")
    summary = summarize(results)
    (output / "summary.json").write_text(json.dumps(summary, indent=2) + "\n")
    print(json.dumps(summary["groups"], indent=2))


if __name__ == "__main__":
    main()
