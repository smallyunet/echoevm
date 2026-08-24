import subprocess
import sys
import tempfile
import unittest
import zipfile
from pathlib import Path


class PackageAgentSkillsTest(unittest.TestCase):
    def test_packages_installable_archive_with_stable_paths_and_modes(self) -> None:
        root = Path(__file__).resolve().parents[1]
        script = root / "tools" / "package_agent_skills.py"
        with tempfile.TemporaryDirectory() as temp_dir:
            subprocess.run(
                [sys.executable, str(script), "--output", temp_dir],
                check=True,
                capture_output=True,
                text=True,
            )
            archive_path = Path(temp_dir) / "echoevm-debug.skill"
            with zipfile.ZipFile(archive_path) as archive:
                names = archive.namelist()
                self.assertIn("SKILL.md", names)
                self.assertFalse(any(name.startswith("scripts/") for name in names))
                self.assertFalse(any(name.startswith("echoevm-debug/") for name in names))
                self.assertEqual(archive.getinfo("SKILL.md").date_time, (1980, 1, 1, 0, 0, 0))
            self.assertFalse((Path(temp_dir) / "echoevm-conformance.skill").exists())


if __name__ == "__main__":
    unittest.main()
