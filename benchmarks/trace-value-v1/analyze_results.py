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


def clustered_percent_interval(rows: list[dict[str, object]], treatment: str, field: str, seed: int = 20260806) -> dict[str, float]:
    control = per_case(rows, "control", field)
    treated = per_case(rows, treatment, field)
    cases = sorted(control)
    deltas = {case: (treated[case] - control[case]) / control[case] for case in cases}
    rng = random.Random(seed)
    samples = []
    for _ in range(10_000):
        picked = [rng.choice(cases) for _ in cases]
        samples.append(mean([deltas[case] for case in picked]))
    samples.sort()
    return {"relativeDelta": mean(list(deltas.values())),
            "ci95Low": samples[int(len(samples) * 0.025)],
            "ci95High": samples[int(len(samples) * 0.975)]}


def comparison(rows: list[dict[str, object]], treatment: str) -> dict[str, object]:
    return {
        "accuracy": clustered_interval(rows, treatment, "diagnosis_correct"),
        "causalAccuracy": clustered_interval(rows, treatment, "causal_correct"),
        "score": clustered_interval(rows, treatment, "score"),
        "freshTokens": clustered_percent_interval(rows, treatment, "fresh_tokens"),
        "duration": clustered_percent_interval(rows, treatment, "duration_seconds"),
        "evidenceBytes": clustered_percent_interval(rows, treatment, "evidence_bytes"),
    }


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("results", type=Path)
    parser.add_argument("--publish-dir", type=Path)
    args = parser.parse_args()
    summary = json.loads((args.results / "summary.json").read_text())
    rows = summary["results"]
    for row in rows:
        usage = row["usage"]
        row["fresh_tokens"] = usage["input_tokens"] - usage["cached_input_tokens"] + usage["output_tokens"]
    groups = summary["groups"]
    conditions = sorted(groups)
    for condition, group in groups.items():
        selected = [row for row in rows if row["condition"] == condition]
        group["semanticCorrect"] = sum(
            bool(row["components"]["rootCause"] and row["components"]["fix"]) for row in selected
        )
        group["semanticAccuracy"] = group["semanticCorrect"] / len(selected)
        group["causalCorrect"] = sum(bool(row.get("causal_correct")) for row in selected)
        group["causalAccuracy"] = group["causalCorrect"] / len(selected)
    analysis = {"schema": "echoevm.trace-value-analysis.v1", "groups": groups, "comparisons": {}}
    if "control" in conditions:
        for condition in [item for item in conditions if item != "control"]:
            analysis["comparisons"][f"{condition}-vs-control"] = comparison(rows, condition)
    if "evidence" in conditions:
        for baseline in ("raw", "echo"):
            if baseline not in conditions:
                continue
            renamed = []
            for row in [item for item in rows if item["condition"] in (baseline, "evidence")]:
                copy = dict(row)
                if copy["condition"] == baseline:
                    copy["condition"] = "control"
                renamed.append(copy)
            analysis["comparisons"][f"evidence-vs-{baseline}"] = comparison(renamed, "evidence")
    by_case = {}
    for case in sorted({row["case"] for row in rows}):
        by_case[case] = {}
        for condition in conditions:
            selected = [row for row in rows if row["case"] == case and row["condition"] == condition]
            by_case[case][condition] = {"correct": sum(row["diagnosis_correct"] for row in selected),
                                        "causalCorrect": sum(row.get("causal_correct", False) for row in selected),
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
