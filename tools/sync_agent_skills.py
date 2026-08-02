#!/usr/bin/env python3
"""Synchronize canonical .agents skills into Claude Code's project directory."""

from __future__ import annotations

import argparse
import filecmp
import shutil
import sys
from pathlib import Path

MANAGED_SKILLS = ("echoevm-debug", "echoevm-conformance")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true", help="fail instead of updating stale mirrors")
    return parser.parse_args()


def same_tree(source: Path, target: Path) -> bool:
    if not source.is_dir() or not target.is_dir():
        return False
    comparison = filecmp.dircmp(source, target)
    if comparison.left_only or comparison.right_only or comparison.funny_files:
        return False
    if any(not filecmp.cmp(source / name, target / name, shallow=False) for name in comparison.common_files):
        return False
    return all(same_tree(source / name, target / name) for name in comparison.common_dirs)


def main() -> int:
    args = parse_args()
    root = Path(__file__).resolve().parents[1]
    canonical = root / ".agents" / "skills"
    claude = root / ".claude" / "skills"
    stale = []
    for name in MANAGED_SKILLS:
        source = canonical / name
        target = claude / name
        if not source.joinpath("SKILL.md").is_file():
            print(f"error: missing canonical skill {source}", file=sys.stderr)
            return 1
        if same_tree(source, target):
            continue
        stale.append(name)
        if not args.check:
            if target.exists():
                shutil.rmtree(target)
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copytree(source, target)
    if stale and args.check:
        print("error: stale Claude skill mirror: " + ", ".join(stale), file=sys.stderr)
        print("run: python3 tools/sync_agent_skills.py", file=sys.stderr)
        return 1
    if stale:
        print("synchronized Claude skills: " + ", ".join(stale))
    else:
        print("Claude skill mirrors are current")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
