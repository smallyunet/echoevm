#!/usr/bin/env python3
"""Compact EchoEVM JSON results into a model-friendly evidence window."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any

DEFAULT_WINDOW = 12
MAX_WINDOW = 100
MAX_STRING = 512
MAX_MAP_ENTRIES = 32
MAX_STACK_WORDS = 16


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("result", type=Path, help="EchoEVM JSON result file")
    parser.add_argument("--window", type=int, default=DEFAULT_WINDOW, help="trace steps before and after the first divergence")
    return parser.parse_args()


def first_divergence_step(value: Any) -> int | None:
    if not isinstance(value, dict):
        return None
    divergence = value.get("firstDivergence")
    if isinstance(divergence, dict) and isinstance(divergence.get("step"), int):
        return divergence["step"]
    comparison = value.get("comparison")
    if isinstance(comparison, dict):
        return first_divergence_step(comparison)
    return None


def compact_string(value: str) -> Any:
    if len(value) <= MAX_STRING:
        return value
    return {"length": len(value), "prefix": value[:MAX_STRING]}


def compact_map(value: dict[str, Any], divergence_step: int | None, window: int) -> dict[str, Any]:
    items = list(value.items())
    if len(items) <= MAX_MAP_ENTRIES:
        return {key: compact(item, divergence_step, window, key) for key, item in items}
    return {
        "entryCount": len(items),
        "entries": {
            key: compact(item, divergence_step, window, key)
            for key, item in items[:MAX_MAP_ENTRIES]
        },
        "truncated": True,
    }


def compact_trace(trace: list[Any], divergence_step: int | None, window: int) -> dict[str, Any]:
    center = divergence_step if divergence_step is not None else 0
    start = max(0, center - window)
    end = min(len(trace), center + window + 1)
    return {
        "stepCount": len(trace),
        "windowStart": start,
        "windowEnd": end,
        "steps": [compact(step, divergence_step, window, "traceStep") for step in trace[start:end]],
        "truncated": start > 0 or end < len(trace),
    }


def compact(value: Any, divergence_step: int | None, window: int, key: str = "") -> Any:
    if isinstance(value, str):
        return compact_string(value)
    if isinstance(value, list):
        if key == "trace":
            return compact_trace(value, divergence_step, window)
        if key in {"stackBefore", "stackAfter"} and len(value) > MAX_STACK_WORDS:
            return {
                "wordCount": len(value),
                "topWords": value[-MAX_STACK_WORDS:],
                "truncated": True,
            }
        return [compact(item, divergence_step, window) for item in value]
    if isinstance(value, dict):
        if key in {"storage", "echoState", "gethState", "initialStorage"}:
            return compact_map(value, divergence_step, window)
        return {name: compact(item, divergence_step, window, name) for name, item in value.items()}
    return value


def main() -> int:
    args = parse_args()
    if args.window < 0 or args.window > MAX_WINDOW:
        print(f"error: --window must be between 0 and {MAX_WINDOW}", file=sys.stderr)
        return 2
    try:
        with args.result.open(encoding="utf-8") as handle:
            value = json.load(handle)
    except (OSError, json.JSONDecodeError) as error:
        print(f"error: cannot read EchoEVM JSON result: {error}", file=sys.stderr)
        return 1
    divergence_step = first_divergence_step(value)
    json.dump(compact(value, divergence_step, args.window), sys.stdout, indent=2, sort_keys=True)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
