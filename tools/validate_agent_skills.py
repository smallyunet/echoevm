#!/usr/bin/env python3
"""Validate EchoEVM's portable Agent Skills without third-party packages."""

from __future__ import annotations

import re
import sys
from pathlib import Path

NAME_RE = re.compile(r"^[a-z0-9]+(?:-[a-z0-9]+)*$")
LINK_RE = re.compile(r"\[[^]]+\]\(([^)]+)\)")
EXPECTED = ("echoevm-debug", "echoevm-conformance")


def validate_skill(skill_dir: Path) -> list[str]:
    errors: list[str] = []
    path = skill_dir / "SKILL.md"
    if not path.is_file():
        return [f"{path}: missing"]
    text = path.read_text(encoding="utf-8")
    lines = text.splitlines()
    if len(lines) > 500:
        errors.append(f"{path}: exceeds 500 lines")
    if "TODO" in text:
        errors.append(f"{path}: contains TODO placeholder")
    if not lines or lines[0] != "---":
        errors.append(f"{path}: missing YAML frontmatter")
        return errors
    try:
        end = lines.index("---", 1)
    except ValueError:
        errors.append(f"{path}: unterminated YAML frontmatter")
        return errors
    fields: dict[str, str] = {}
    for line in lines[1:end]:
        if ":" not in line:
            errors.append(f"{path}: invalid frontmatter line {line!r}")
            continue
        key, value = line.split(":", 1)
        fields[key.strip()] = value.strip()
    if set(fields) != {"name", "description"}:
        errors.append(f"{path}: frontmatter must contain only name and description")
    name = fields.get("name", "")
    if name != skill_dir.name or not NAME_RE.fullmatch(name) or len(name) > 64:
        errors.append(f"{path}: invalid or mismatched name {name!r}")
    description = fields.get("description", "")
    if not description or len(description) > 1024 or "Use when" not in description:
        errors.append(f"{path}: description must state what the skill does and when to use it")
    for target in LINK_RE.findall(text):
        if "://" in target or target.startswith("#"):
            continue
        resolved = skill_dir / target.split("#", 1)[0]
        if not resolved.exists():
            errors.append(f"{path}: broken local reference {target}")
    metadata = skill_dir / "agents" / "openai.yaml"
    if not metadata.is_file():
        errors.append(f"{metadata}: missing Codex interface metadata")
    return errors


def main() -> int:
    root = Path(__file__).resolve().parents[1]
    skills_root = root / ".agents" / "skills"
    errors: list[str] = []
    for name in EXPECTED:
        errors.extend(validate_skill(skills_root / name))
    unexpected = sorted(path.name for path in skills_root.iterdir() if path.is_dir() and path.name not in EXPECTED)
    if unexpected:
        errors.append("unexpected canonical skills: " + ", ".join(unexpected))
    if errors:
        for error in errors:
            print("error: " + error, file=sys.stderr)
        return 1
    print(f"validated {len(EXPECTED)} portable Agent Skills")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
