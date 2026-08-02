import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


class CompactResultTest(unittest.TestCase):
    def test_compacts_trace_around_first_divergence(self) -> None:
        root = Path(__file__).resolve().parents[1]
        script = root / ".agents" / "skills" / "echoevm-debug" / "scripts" / "compact_result.py"
        value = {
            "match": False,
            "firstDivergence": {"step": 10, "field": "gasAfter"},
            "echoevm": {
                "trace": [
                    {"index": index, "stackBefore": [f"0x{word:x}" for word in range(20)]}
                    for index in range(25)
                ]
            },
            "geth": {"trace": [{"index": index} for index in range(25)]},
        }
        with tempfile.TemporaryDirectory() as temp_dir:
            source = Path(temp_dir) / "result.json"
            source.write_text(json.dumps(value), encoding="utf-8")
            completed = subprocess.run(
                [sys.executable, str(script), str(source), "--window", "2"],
                check=True,
                capture_output=True,
                text=True,
            )
        result = json.loads(completed.stdout)
        trace = result["echoevm"]["trace"]
        self.assertEqual(trace["stepCount"], 25)
        self.assertEqual(trace["windowStart"], 8)
        self.assertEqual(trace["windowEnd"], 13)
        self.assertEqual(len(trace["steps"]), 5)
        self.assertEqual(trace["steps"][0]["stackBefore"]["wordCount"], 20)

    def test_rejects_invalid_window(self) -> None:
        root = Path(__file__).resolve().parents[1]
        script = root / ".agents" / "skills" / "echoevm-debug" / "scripts" / "compact_result.py"
        with tempfile.TemporaryDirectory() as temp_dir:
            source = Path(temp_dir) / "result.json"
            source.write_text("{}", encoding="utf-8")
            completed = subprocess.run(
                [sys.executable, str(script), str(source), "--window", "101"],
                capture_output=True,
                text=True,
            )
        self.assertEqual(completed.returncode, 2)
        self.assertIn("between 0 and 100", completed.stderr)


if __name__ == "__main__":
    unittest.main()
