import stat
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
                self.assertIn("scripts/compact_result.py", names)
                self.assertFalse(any(name.startswith("echoevm-debug/") for name in names))
                mode = archive.getinfo("scripts/compact_result.py").external_attr >> 16
                self.assertTrue(mode & stat.S_IXUSR)
                self.assertEqual(archive.getinfo("SKILL.md").date_time, (1980, 1, 1, 0, 0, 0))


if __name__ == "__main__":
    unittest.main()
