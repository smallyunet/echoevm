import importlib.util
from pathlib import Path
import unittest


SCRIPT = Path(__file__).parents[1] / "benchmarks" / "trace-value-v2" / "run_benchmark.py"
SPEC = importlib.util.spec_from_file_location("trace_value_v2_runner", SCRIPT)
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class TraceValueV2BenchmarkTest(unittest.TestCase):
    def test_exact_nested_answer_scores_ten(self):
        oracle = {
            "rootCause": "IGNORED_LOW_LEVEL_CALL_FAILURE",
            "primary": {"depth": 0, "pc": 682, "opcode": "CALL"},
            "secondary": {"depth": 1, "pc": 237, "opcode": "REVERT"},
            "fix": "REQUIRE_CALL_SUCCESS",
        }
        answer = {
            "rootCause": "IGNORED_LOW_LEVEL_CALL_FAILURE",
            "primary": {"depth": 0, "pc": 682, "opcode": "call"},
            "secondary": {"depth": 1, "pc": 237, "opcode": "revert"},
            "fix": "REQUIRE_CALL_SUCCESS",
        }
        result = MODULE.score(answer, oracle)
        self.assertEqual(result["score"], 10)
        self.assertTrue(result["diagnosis_correct"])

    def test_wrong_depth_fails_strict_diagnosis(self):
        oracle = {
            "rootCause": "DELEGATECALL_STORAGE_CONTEXT",
            "primary": {"depth": 0, "pc": 454, "opcode": "DELEGATECALL"},
            "secondary": {"depth": 1, "pc": 244, "opcode": "SSTORE"},
            "fix": "USE_CALL_NOT_DELEGATECALL",
        }
        answer = {
            "rootCause": "DELEGATECALL_STORAGE_CONTEXT",
            "primary": {"depth": 0, "pc": 454, "opcode": "DELEGATECALL"},
            "secondary": {"depth": 0, "pc": 244, "opcode": "SSTORE"},
            "fix": "USE_CALL_NOT_DELEGATECALL",
        }
        result = MODULE.score(answer, oracle)
        self.assertEqual(result["score"], 8)
        self.assertFalse(result["diagnosis_correct"])


if __name__ == "__main__":
    unittest.main()
