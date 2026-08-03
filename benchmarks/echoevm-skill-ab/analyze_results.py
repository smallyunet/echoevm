#!/usr/bin/env python3
"""Produce per-task metrics and EchoEVM evidence audits from benchmark runs."""

from __future__ import annotations

import argparse
import json
from pathlib import Path
import statistics


def median(values: list[float | int]) -> float | None:
    return statistics.median(values) if values else None


def command_audit(jsonl: Path) -> dict[str, int]:
    commands = 0
    failed = 0
    echo_attempts = 0
    echo_successes = 0
    for line in jsonl.read_text(errors="replace").splitlines():
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            continue
        if event.get("type") != "item.completed":
            continue
        item = event.get("item", {})
        if item.get("type") != "command_execution":
            continue
        commands += 1
        if item.get("exit_code") not in (0, None):
            failed += 1
        command = item.get("command", "")
        output = item.get("aggregated_output", "")
        if "echoevm" in command:
            echo_attempts += 1
            if "echoevm unavailable in control condition" not in output and (
                '"schemaVersion"' in output
                or '"version": "v0.0.37"' in output
                or "status=success" in output
                or "MATCH" in output
            ):
                echo_successes += 1
    return {
        "command_calls": commands,
        "failed_command_calls": failed,
        "echoevm_command_calls": echo_attempts,
        "echoevm_successful_evidence_calls": echo_successes,
    }


def load_rows(root: Path) -> list[dict[str, object]]:
    rows: list[dict[str, object]] = []
    for result_path in sorted((root / "runs").glob("*/result.json")):
        row = json.loads(result_path.read_text())
        usage = row["usage"]
        row["total_tokens"] = usage["input_tokens"] + usage["output_tokens"]
        row["noncached_tokens"] = usage["input_tokens"] - usage["cached_input_tokens"] + usage["output_tokens"]
        row.update(command_audit(result_path.parent / "codex.jsonl"))
        final = (result_path.parent / "final.txt").read_text(errors="replace")
        row["final_reports_echo_evidence"] = "EchoEVM" in final and ("Geth" in final or "gas" in final)
        rows.append(row)
    return rows


def group(rows: list[dict[str, object]]) -> dict[str, object]:
    passed = [row for row in rows if row["passed"]]
    return {
        "runs": len(rows),
        "passed": len(passed),
        "median_duration_seconds": median([row["duration_seconds"] for row in passed]),
        "median_total_tokens": median([row["total_tokens"] for row in passed]),
        "median_noncached_tokens": median([row["noncached_tokens"] for row in passed]),
        "median_output_tokens": median([row["usage"]["output_tokens"] for row in passed]),
        "median_reasoning_tokens": median([row["usage"]["reasoning_output_tokens"] for row in passed]),
        "median_command_calls": median([row["command_calls"] for row in passed]),
        "echoevm_successful_evidence_calls": sum(row["echoevm_successful_evidence_calls"] for row in rows),
        "finals_reporting_echo_evidence": sum(bool(row["final_reports_echo_evidence"]) for row in rows),
    }


def percent_delta(skill: float | None, control: float | None) -> float | None:
    if skill is None or control in (None, 0):
        return None
    return round((skill - control) * 100 / control, 2)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("results", type=Path)
    parser.add_argument("--output", type=Path)
    args = parser.parse_args()
    rows = load_rows(args.results)
    tasks = sorted({str(row["task"]) for row in rows})
    report: dict[str, object] = {"aggregate": {}, "tasks": {}, "runs": rows}
    for condition in ("control", "skill"):
        report["aggregate"][condition] = group([row for row in rows if row["condition"] == condition])
    for task in tasks:
        control = group([row for row in rows if row["task"] == task and row["condition"] == "control"])
        skill = group([row for row in rows if row["task"] == task and row["condition"] == "skill"])
        report["tasks"][task] = {
            "control": control,
            "skill": skill,
            "skill_delta_percent": {
                "duration": percent_delta(skill["median_duration_seconds"], control["median_duration_seconds"]),
                "total_tokens": percent_delta(skill["median_total_tokens"], control["median_total_tokens"]),
                "noncached_tokens": percent_delta(skill["median_noncached_tokens"], control["median_noncached_tokens"]),
                "output_tokens": percent_delta(skill["median_output_tokens"], control["median_output_tokens"]),
            },
        }
    control = report["aggregate"]["control"]
    skill = report["aggregate"]["skill"]
    report["aggregate"]["skill_delta_percent"] = {
        "duration": percent_delta(skill["median_duration_seconds"], control["median_duration_seconds"]),
        "total_tokens": percent_delta(skill["median_total_tokens"], control["median_total_tokens"]),
        "noncached_tokens": percent_delta(skill["median_noncached_tokens"], control["median_noncached_tokens"]),
        "output_tokens": percent_delta(skill["median_output_tokens"], control["median_output_tokens"]),
    }
    rendered = json.dumps(report, indent=2) + "\n"
    output = args.output or args.results / "analysis.json"
    output.write_text(rendered)
    print(json.dumps({"aggregate": report["aggregate"], "tasks": report["tasks"]}, indent=2))


if __name__ == "__main__":
    main()
