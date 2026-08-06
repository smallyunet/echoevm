import importlib.util
import sys
from pathlib import Path
import unittest


SCRIPT = Path(__file__).parents[1] / "benchmarks" / "trace-value-v1" / "run_benchmark.py"
sys.path.insert(0, str(SCRIPT.parent))
SPEC = importlib.util.spec_from_file_location("trace_value_runner", SCRIPT)
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class TraceValueBenchmarkTest(unittest.TestCase):
    def test_noise_variant_shifts_oracle_and_preserves_base(self):
        cases = MODULE.load_cases(SCRIPT.parent / "cases.json")
        by_id = {case["id"]: case for case in cases}
        base = by_id["storage_rolled_back"]
        variant = by_id["storage_rolled_back_noise96"]
        self.assertEqual(base["oracle"]["primaryPC"], 8)
        self.assertEqual(variant["oracle"]["primaryPC"], 296)
        self.assertEqual(variant["oracle"]["secondaryPC"], 291)
        self.assertTrue(variant["bytecode"].endswith(base["bytecode"]))

    def test_exact_answer_scores_ten(self):
        oracle = {"rootCause": "STORAGE_SLOT", "primaryPC": 4, "primaryOpcode": "SSTORE",
                  "acceptedPrimary": [{"pc": 4, "opcode": "SSTORE"}, {"pc": 2, "opcode": "PUSH1"}],
                  "secondaryPC": 7, "secondaryOpcode": "SLOAD", "fix": "USE_STORAGE_SLOT_0"}
        answer = {"rootCause": "STORAGE_SLOT", "primary": {"pc": 4, "opcode": "sstore"},
                  "secondary": {"pc": 7, "opcode": "sload"}, "fix": "USE_STORAGE_SLOT_0"}
        result = MODULE.score_answer(answer, oracle)
        self.assertEqual(result["score"], 10)
        self.assertTrue(result["diagnosis_correct"])
        self.assertTrue(result["causal_correct"])

    def test_wrong_fix_fails_correctness(self):
        oracle = {"rootCause": "RETURN_OFFSET", "primaryPC": 8, "primaryOpcode": "RETURN",
                  "acceptedPrimary": [{"pc": 8, "opcode": "RETURN"}, {"pc": 6, "opcode": "PUSH1"}],
                  "secondaryPC": None, "secondaryOpcode": None, "fix": "RETURN_FROM_OFFSET_0"}
        answer = {"rootCause": "RETURN_OFFSET", "primary": {"pc": 8, "opcode": "RETURN"},
                  "secondary": None, "fix": "USE_MSTORE"}
        result = MODULE.score_answer(answer, oracle)
        self.assertEqual(result["score"], 8)
        self.assertFalse(result["diagnosis_correct"])

    def test_producer_location_is_causally_correct(self):
        oracle = {"rootCause": "RETURN_OFFSET", "primaryPC": 8, "primaryOpcode": "RETURN",
                  "acceptedPrimary": [{"pc": 8, "opcode": "RETURN"}, {"pc": 6, "opcode": "PUSH1"}],
                  "secondaryPC": None, "secondaryOpcode": None, "fix": "RETURN_FROM_OFFSET_0"}
        answer = {"rootCause": "RETURN_OFFSET", "primary": {"pc": 6, "opcode": "PUSH1"},
                  "secondary": None, "fix": "RETURN_FROM_OFFSET_0"}
        result = MODULE.score_answer(answer, oracle)
        self.assertFalse(result["diagnosis_correct"])
        self.assertTrue(result["causal_correct"])


if __name__ == "__main__":
    unittest.main()
