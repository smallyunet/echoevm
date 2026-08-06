#!/usr/bin/env python3
"""Analyze trace-value results with task-clustered bootstrap intervals."""

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


def clustered_interval(rows: list[dict[str, object]], treatment: str, field: str, seed: int = 20260806) -> dict[str, float]:
    control = per_case(rows, "control", field)
    treated = per_case(rows, treatment, field)
    cases = sorted(control)
    rng = random.Random(seed)
    samples = []
    for _ in range(10_000):
        picked = [rng.choice(cases) for _ in cases]
        samples.append(mean([treated[case] - control[case] for case in picked]))
    samples.sort()
    return {"delta": mean([treated[case] - control[case] for case in cases]),
            "ci95Low": samples[int(len(samples) * 0.025)],
            "ci95High": samples[int(len(samples) * 0.975)]}


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("results", type=Path)
    parser.add_argument("--publish-dir", type=Path)
    args = parser.parse_args()
    summary = json.loads((args.results / "summary.json").read_text())
    rows = summary["results"]
    groups = summary["groups"]
    for condition, group in groups.items():
        selected = [row for row in rows if row["condition"] == condition]
        group["semanticCorrect"] = sum(
            bool(row["components"]["rootCause"] and row["components"]["fix"]) for row in selected
        )
        group["semanticAccuracy"] = group["semanticCorrect"] / len(selected)
    analysis = {"schema": "echoevm.trace-value-analysis.v1", "groups": groups, "comparisons": {}}
    for condition in ("raw", "echo"):
        analysis["comparisons"][f"{condition}-vs-control"] = {
            "accuracy": clustered_interval(rows, condition, "diagnosis_correct"),
            "score": clustered_interval(rows, condition, "score"),
        }
    by_case = {}
    for case in sorted({row["case"] for row in rows}):
        by_case[case] = {}
        for condition in ("control", "raw", "echo"):
            selected = [row for row in rows if row["case"] == case and row["condition"] == condition]
            by_case[case][condition] = {"correct": sum(row["diagnosis_correct"] for row in selected),
                                        "runs": len(selected), "meanScore": mean([row["score"] for row in selected])}
    analysis["cases"] = by_case
    (args.results / "analysis.json").write_text(json.dumps(analysis, indent=2) + "\n")
    if args.publish_dir:
        publish = args.publish_dir.resolve()
        publish.mkdir(parents=True, exist_ok=False)
        (publish / "analysis.json").write_text(json.dumps(analysis, indent=2) + "\n")
        ordered = sorted(rows, key=lambda row: row["run_id"])
        (publish / "runs.jsonl").write_text("".join(json.dumps(row, sort_keys=True) + "\n" for row in ordered))
        metadata_path = args.results / "RUN_METADATA.json"
        if metadata_path.is_file():
            (publish / "run-metadata.json").write_text(metadata_path.read_text())
    print(json.dumps(analysis, indent=2))


if __name__ == "__main__":
    main()
