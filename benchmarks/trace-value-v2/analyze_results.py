#!/usr/bin/env python3
"""Analyze v2 with task-clustered relative bootstrap intervals."""

from __future__ import annotations

import argparse
import json
from pathlib import Path
import random
import statistics


def mean(values: list[float]) -> float:
    return sum(values) / len(values)


def per_case(rows: list[dict[str, object]], condition: str, field: str) -> dict[str, float]:
    cases = sorted({str(row["case"]) for row in rows})
    return {case: mean([float(row[field]) for row in rows if row["case"] == case and row["condition"] == condition]) for case in cases}


def interval(rows: list[dict[str, object]], baseline: str, treatment: str, field: str, relative: bool, seed: int = 20260806) -> dict[str, float]:
    base, treated = per_case(rows, baseline, field), per_case(rows, treatment, field)
    cases = sorted(base)
    deltas = {case: ((treated[case] - base[case]) / base[case] if relative else treated[case] - base[case]) for case in cases}
    rng = random.Random(seed)
    samples = sorted(mean([deltas[rng.choice(cases)] for _ in cases]) for _ in range(10_000))
    key = "relativeDelta" if relative else "delta"
    return {key: mean(list(deltas.values())), "ci95Low": samples[250], "ci95High": samples[9750]}


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("results", type=Path)
    parser.add_argument("--publish-dir", type=Path)
    args = parser.parse_args()
    summary = json.loads((args.results / "summary.json").read_text())
    rows = summary["results"]
    for row in rows:
        use = row["usage"]
        row["freshTokens"] = use["input_tokens"] - use["cached_input_tokens"] + use["output_tokens"]
    comparisons = {}
    for baseline in ("control", "broad"):
        if baseline not in summary["groups"] or "evidence" not in summary["groups"]:
            continue
        comparisons[f"evidence-vs-{baseline}"] = {
            "accuracy": interval(rows, baseline, "evidence", "diagnosis_correct", False),
            "score": interval(rows, baseline, "evidence", "score", False),
            "freshTokens": interval(rows, baseline, "evidence", "freshTokens", True),
            "duration": interval(rows, baseline, "evidence", "duration_seconds", True),
            "evidenceBytes": interval(rows, baseline, "evidence", "evidence_bytes", True),
        }
    analysis = {"schema": "echoevm.trace-value-v2-analysis.v1", "groups": summary["groups"], "comparisons": comparisons}
    (args.results / "analysis.json").write_text(json.dumps(analysis, indent=2) + "\n")
    if args.publish_dir:
        publish = args.publish_dir.resolve()
        publish.mkdir(parents=True, exist_ok=False)
        (publish / "analysis.json").write_text(json.dumps(analysis, indent=2) + "\n")
        (publish / "runs.jsonl").write_text("".join(json.dumps(row, sort_keys=True) + "\n" for row in sorted(rows, key=lambda row: row["run_id"])))
        (publish / "run-metadata.json").write_text((args.results / "RUN_METADATA.json").read_text())
    print(json.dumps(analysis, indent=2))


if __name__ == "__main__":
    main()
