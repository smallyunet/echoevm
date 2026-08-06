"""Shared case loading and deterministic benchmark variants."""

from __future__ import annotations

from copy import deepcopy
import json
from pathlib import Path


def load_cases(path: Path) -> list[dict[str, object]]:
    """Load concrete cases and expand neutral stack-noise variants."""
    definitions = json.loads(path.read_text())
    concrete = [deepcopy(row) for row in definitions if "baseCase" not in row]
    by_id = {str(row["id"]): row for row in concrete}
    result = list(concrete)
    for definition in definitions:
        if "baseCase" not in definition:
            continue
        base = deepcopy(by_id[str(definition["baseCase"])])
        noise_pairs = int(definition["noisePairs"])
        prefix = "600050" * noise_pairs  # PUSH1 0; POP, repeated.
        pc_shift = 3 * noise_pairs
        base["id"] = definition["id"]
        base["bytecode"] = prefix + str(base["bytecode"])
        base["question"] = str(base["question"]) + " The runtime also contains unrelated stack setup before the failing logic."
        oracle = base["oracle"]
        oracle["primaryPC"] += pc_shift
        if oracle["secondaryPC"] is not None:
            oracle["secondaryPC"] += pc_shift
        for location in oracle["acceptedPrimary"]:
            location["pc"] += pc_shift
        base["variant"] = {"kind": "neutral-stack-prefix", "pairs": noise_pairs, "pcShift": pc_shift}
        result.append(base)
    return result
