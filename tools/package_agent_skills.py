#!/usr/bin/env python3
"""Package EchoEVM Agent Skills as portable .skill ZIP archives."""

from __future__ import annotations

import argparse
import stat
import zipfile
from pathlib import Path

# The debug skill is the portable end-user workflow. Conformance remains a
# repository-local contributor skill under .agents/skills.
SKILLS = ("echoevm-debug",)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, default=Path("dist"), help="archive output directory")
    return parser.parse_args()


def package_skill(source: Path, destination: Path) -> None:
    with zipfile.ZipFile(destination, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=9) as archive:
        for path in sorted(item for item in source.rglob("*") if item.is_file()):
            relative = path.relative_to(source).as_posix()
            info = zipfile.ZipInfo(relative, date_time=(1980, 1, 1, 0, 0, 0))
            mode = 0o755 if path.stat().st_mode & stat.S_IXUSR else 0o644
            info.external_attr = (stat.S_IFREG | mode) << 16
            info.compress_type = zipfile.ZIP_DEFLATED
            archive.writestr(info, path.read_bytes())


def main() -> int:
    args = parse_args()
    root = Path(__file__).resolve().parents[1]
    source_root = root / ".agents" / "skills"
    output = args.output if args.output.is_absolute() else root / args.output
    output.mkdir(parents=True, exist_ok=True)
    for name in SKILLS:
        source = source_root / name
        if not source.joinpath("SKILL.md").is_file():
            raise SystemExit(f"missing skill: {source}")
        destination = output / f"{name}.skill"
        package_skill(source, destination)
        print(destination)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
