#!/usr/bin/env python3
"""Generate frozen control, raw-opcode, and EchoEVM evidence from one binary."""

from __future__ import annotations

import argparse
import json
from pathlib import Path
import shutil
import subprocess
import tempfile


ROOT = Path(__file__).resolve().parent
GAS_LIMIT = 15_000_000


def run_json(args: list[str]) -> dict[str, object]:
    proc = subprocess.run(args, text=True, capture_output=True, timeout=60)
    try:
        result = json.loads(proc.stdout)
    except json.JSONDecodeError as error:
        raise RuntimeError(f"command failed: {' '.join(args)}\n{proc.stderr}") from error
    # `trace` deliberately emits a complete fault document before returning a
    # non-zero process status. That is valid diagnostic evidence, unlike a
    # command failure with no machine-readable result.
    if proc.returncode != 0 and result.get("execution", {}).get("status") != "fault":
        raise RuntimeError(f"command failed: {' '.join(args)}\n{proc.stderr}")
    return result


def raw_opcode_evidence(diff: dict[str, object]) -> dict[str, object]:
    geth = diff["geth"]
    logs = []
    for step in geth["trace"]:
        logs.append({
            "pc": step["pc"],
            "op": step["opcodeName"],
            "gas": step["gasBefore"],
            "gasCost": step["gasBefore"] - step["gasAfter"],
            "depth": step["depth"],
            "stack": step["stackBefore"],
            **({"error": step["haltClass"]} if step.get("haltClass") == "fault" else {}),
        })
    return {
        "format": "geth-style-raw-opcode.v1",
        "engine": geth["engine"],
        "engineVersion": geth["engineVersion"],
        "execution": {
            "status": geth["status"], "gasUsed": geth["gasUsed"],
            "returnData": geth["returnData"], "storage": geth["storage"],
        },
        "structLogs": logs,
    }


def require_oracle_steps(case: dict[str, object], diff: dict[str, object], trace: dict[str, object]) -> None:
    oracle = case["oracle"]
    expected = [(oracle["primaryPC"], oracle["primaryOpcode"])]
    if oracle["secondaryPC"] is not None:
        expected.append((oracle["secondaryPC"], oracle["secondaryOpcode"]))
    normalize = lambda name: "SHA3" if name == "KECCAK256" else name
    raw_steps = {(step["pc"], normalize(step["opcodeName"])) for step in diff["geth"]["trace"]}
    echo_steps = {(step["pc"], normalize(step["opcodeName"])) for step in trace["events"]}
    for step in expected:
        if step not in raw_steps or step not in echo_steps:
            raise RuntimeError(f"{case['id']} oracle step {step} missing from generated evidence")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--echoevm", type=Path, required=True)
    parser.add_argument("--output", type=Path, default=ROOT / "fixtures")
    args = parser.parse_args()
    binary = args.echoevm.resolve()
    version = run_json([str(binary), "version", "--json"])
    cases = json.loads((ROOT / "cases.json").read_text())
    output = args.output.resolve()
    if output.exists():
        shutil.rmtree(output)
    output.mkdir(parents=True)
    with tempfile.TemporaryDirectory(prefix="echoevm-trace-value-") as temp:
        temp_root = Path(temp)
        for case in cases:
            case_dir = output / case["id"]
            case_dir.mkdir()
            runtime = temp_root / f"{case['id']}.bin"
            runtime.write_text(case["bytecode"] + "\n")
            diff = run_json([
                str(binary), "diff", "--code", case["bytecode"], "--input", case["calldata"],
                "--gas", str(GAS_LIMIT), "--fork", "Cancun", "--format", "json",
            ])
            if not diff["match"]:
                raise RuntimeError(f"{case['id']} diverges: {diff.get('firstDivergence')}")
            trace = run_json([
                str(binary), "trace", "--bin-runtime", str(runtime), "--calldata", case["calldata"],
                "--changes-only", "--fields", "gas,stack,memory,storage,control,explanation",
                "--limit", "200", "--format", "json",
            ])
            if trace["execution"]["truncated"]:
                raise RuntimeError(f"{case['id']} explainable trace was truncated")
            require_oracle_steps(case, diff, trace)
            common = {
                "benchmark": "echoevm.trace-value.v1", "case": case["id"],
                "request": {"fork": "Cancun", "bytecode": case["bytecode"], "calldata": case["calldata"], "gasLimit": GAS_LIMIT},
            }
            control = {**common, "format": "execution-summary.v1", "engine": diff["geth"]["engine"],
                       "engineVersion": diff["geth"]["engineVersion"],
                       "execution": {key: diff["geth"][key] for key in ("status", "gasUsed", "returnData", "storage")}}
            raw = {**common, **raw_opcode_evidence(diff)}
            explainable = {**common, "engineVersion": version, **trace}
            for condition, evidence in (("control", control), ("raw", raw), ("echo", explainable)):
                (case_dir / f"{condition}.json").write_text(json.dumps(evidence, indent=2) + "\n")
    (output / "MANIFEST.json").write_text(json.dumps({
        "schema": "echoevm.trace-value-fixtures.v1", "echoevm": version,
        "gasLimit": GAS_LIMIT, "cases": [case["id"] for case in cases],
    }, indent=2) + "\n")


if __name__ == "__main__":
    main()
