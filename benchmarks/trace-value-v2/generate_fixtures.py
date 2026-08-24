#!/usr/bin/env python3
"""Generate frozen Solidity execution, evidence, and explanation fixtures."""

from __future__ import annotations

import argparse
import json
from pathlib import Path
import shutil
import subprocess


ROOT = Path(__file__).resolve().parent
GAS = 1_000_000


def run_json(args: list[str]) -> dict[str, object]:
    proc = subprocess.run(args, cwd=ROOT.parents[1], text=True, capture_output=True, timeout=120)
    try:
        payload = json.loads(proc.stdout)
    except json.JSONDecodeError as error:
        raise RuntimeError(f"command failed: {' '.join(args)}\n{proc.stderr}") from error
    if proc.returncode != 0:
        raise RuntimeError(f"command failed: {' '.join(args)}\n{proc.stderr}")
    return payload


def command(binary: Path, case: dict[str, object], solc: str, solc_args: list[str], profile: str, limit: int) -> list[str]:
    source = (ROOT / str(case["source"])).relative_to(ROOT.parents[1])
    args = [
        str(binary), "solidity", "run", str(source), "--contract", str(case["contract"]),
        "--function", str(case["function"]), "--args", str(case["args"]),
        "--solc", solc, "--gas", str(GAS),
        "--format", "evidence-json", "--profile", profile, "--limit", str(limit),
    ]
    for value in solc_args:
        args.extend(["--solc-arg", value])
    return args


def explain_command(binary: Path, case: dict[str, object], solc: str, solc_args: list[str]) -> list[str]:
    source = (ROOT / str(case["source"])).relative_to(ROOT.parents[1])
    args = [
        str(binary), "explain", "solidity", str(source),
        "--contract", str(case["contract"]), "--function", str(case["function"]),
        "--args", str(case["args"]), "--solc", solc, "--gas", str(GAS),
        "--format", "json", "--profile", str(case["profile"]), "--limit", "40",
    ]
    if case.get("expectedReturn"):
        args.extend(["--expect-return", str(case["expectedReturn"])])
    for value in solc_args:
        args.extend(["--solc-arg", value])
    return args


def require_oracle(case: dict[str, object], payload: dict[str, object]) -> None:
    events = payload["events"]
    for name in ("primary", "secondary"):
        expected = case["oracle"][name]
        if not any(
            int(event.get("depth", 0)) == expected["depth"] and event["pc"] == expected["pc"] and event["op"] == expected["opcode"]
            for event in events
        ):
            raise RuntimeError(f"{case['id']} missing {name} oracle location {expected}")
    kinds = {link["kind"] for link in payload.get("links", [])}
    missing = set(case["requiredLinks"]) - kinds
    if missing:
        raise RuntimeError(f"{case['id']} missing causal links {sorted(missing)}")
    primary = case["oracle"]["primary"]
    secondary = case["oracle"]["secondary"]
    root_cause = case["oracle"]["rootCause"]
    if root_cause in {"IGNORED_LOW_LEVEL_CALL_FAILURE", "SWALLOWED_CREATE_FAILURE"}:
        if not any(
            link["kind"] == "returns-to"
            and link["from"]["depth"] == secondary["depth"]
            and link["from"]["pc"] == secondary["pc"]
            and link["to"]["depth"] == primary["depth"]
            and link["to"]["pc"] == primary["pc"]
            for link in payload.get("links", [])
        ):
            raise RuntimeError(f"{case['id']} missing exact child-failure return link")
        if not any(
            link["kind"] == "rolls-back"
            and link["to"]["depth"] == secondary["depth"]
            and link["to"]["pc"] == secondary["pc"]
            for link in payload.get("links", [])
        ):
            raise RuntimeError(f"{case['id']} missing exact rollback link")
    elif root_cause == "DELEGATECALL_STORAGE_CONTEXT":
        primary_event = next(event for event in events if event["depth"] == primary["depth"] and event["pc"] == primary["pc"])
        secondary_event = next(event for event in events if event["depth"] == secondary["depth"] and event["pc"] == secondary["pc"])
        writes = [access for access in secondary_event.get("storage", []) if access["kind"] == "write"]
        if not writes or any(access["address"] != primary_event["address"] for access in writes):
            raise RuntimeError(f"{case['id']} does not prove delegate storage context")
    elif root_cause == "WRONG_DIVISOR":
        if not any(
            link["kind"] == "value-flow"
            and link["from"]["depth"] == secondary["depth"]
            and link["from"]["pc"] == secondary["pc"]
            and link["to"]["depth"] == primary["depth"]
            and link["to"]["pc"] == primary["pc"]
            and int(link["value"], 16) == 3
            for link in payload.get("links", [])
        ):
            raise RuntimeError(f"{case['id']} missing exact wrong-divisor value flow")


def require_explanation(case: dict[str, object], payload: dict[str, object]) -> None:
    expected = case["expectedExplanation"]
    if payload.get("schema") != "echoevm.explanation.v1":
        raise RuntimeError(f"{case['id']} emitted unexpected explanation schema")
    if payload.get("verdict", {}).get("code") != expected["verdict"]:
        raise RuntimeError(f"{case['id']} emitted unexpected explanation verdict")
    root = payload.get("rootCause") or {}
    if root.get("code") != expected["rootCause"]:
        raise RuntimeError(f"{case['id']} emitted unexpected root cause")
    locations = {(item["depth"], item["pc"], item["op"]) for item in root.get("evidence", [])}
    for name in ("primary", "secondary"):
        oracle = case["oracle"][name]
        location = (oracle["depth"], oracle["pc"], oracle["opcode"])
        if location not in locations:
            raise RuntimeError(f"{case['id']} explanation missing {name} location {location}")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--echoevm", required=True, type=Path)
    parser.add_argument("--solc", default="node")
    parser.add_argument("--solc-arg", action="append", default=["editors/vscode/dist/solcjs.cjs"])
    parser.add_argument("--output", type=Path, default=ROOT / "fixtures")
    args = parser.parse_args()
    binary = args.echoevm.resolve()
    cases = json.loads((ROOT / "cases.json").read_text())
    version = run_json([str(binary), "version", "--json"])
    output = args.output.resolve()
    if output.exists():
        shutil.rmtree(output)
    output.mkdir(parents=True)
    for case in cases:
        broad = run_json(command(binary, case, args.solc, args.solc_arg, "full", 0))
        evidence = run_json(command(binary, case, args.solc, args.solc_arg, str(case["profile"]), 40))
        explanation = run_json(explain_command(binary, case, args.solc, args.solc_arg))
        require_oracle(case, broad)
        require_oracle(case, evidence)
        require_explanation(case, explanation)
        if broad["execution"] != evidence["execution"]:
            raise RuntimeError(f"{case['id']} execution changed across evidence profiles")
        common = {
            "benchmark": "echoevm.trace-value.v2", "case": case["id"],
            "request": {"contract": case["contract"], "function": case["function"], "args": case["args"], "fork": "Osaka"},
        }
        control = {**common, "format": "execution-summary.v1", "execution": broad["execution"]}
        for condition, payload in (
            ("control", control),
            ("broad", {**common, **broad}),
            ("evidence", {**common, **evidence}),
            ("explanation", {**common, **explanation}),
        ):
            case_dir = output / case["id"]
            case_dir.mkdir(exist_ok=True)
            (case_dir / f"{condition}.json").write_text(json.dumps(payload, separators=(",", ":")) + "\n")
    (output / "MANIFEST.json").write_text(json.dumps({
        "schema": "echoevm.trace-value-v2-fixtures.v1", "echoevm": version,
        "compiler": {"executable": args.solc, "arguments": args.solc_arg},
        "gasLimit": GAS, "cases": [case["id"] for case in cases],
    }, indent=2) + "\n")


if __name__ == "__main__":
    main()
