"""Tests for anki_grading.py: python3 -m unittest scripts/anki_grading_test.py

The cards are invented fixtures, not anyone's deck."""
import json
import os
import sys
import tempfile
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import anki_grading as ag  # noqa: E402


def verdict(i, front, v, back):
    return {"id": i, "kind": "mc", "level": 2, "prompt": front, "correct": v,
            "wrong": ["No" if v == "Yes" else "Yes"], "explain": back}


def typed(i, front, answer, back=""):
    return {"id": i, "kind": "math", "level": 2, "prompt": front, "back": back, "explain": "",
            "parts": [{"label": "y", "answer": answer, "grading": "expression", "tex": answer}]}


class GradingTest(unittest.TestCase):
    def setUp(self):
        self.dir = tempfile.mkdtemp()

    def run_apply(self, items, overrides=None):
        concepts = [{"id": "anki.topic", "items": items}]
        changes, missing = ag.apply(concepts, self.dir, overrides or {})
        return concepts[0]["items"], changes, missing

    def test_rule_turns_a_written_verdict_into_a_flip_card(self):
        items, changes, _ = self.run_apply([
            verdict("a", "Is the relation below a function? Explain why.", "Yes", "Yes: one output per input."),
            verdict("b", "True or false: a square has four sides.", "True", "True."),
        ])
        self.assertEqual(items[0]["kind"], "recall")
        self.assertEqual(items[0]["back"], "Yes: one output per input.")
        self.assertEqual(items[1]["kind"], "mc")
        self.assertEqual([c[0] for c in changes], ["a"])

    def test_rule_turns_a_proof_with_a_typed_key_into_a_flip_card(self):
        items, _, _ = self.run_apply([typed("c", r"Use the definition to prove that \(\lim_{x\to1} 2x = 2\).", "epsilon/2",
                                            "Take delta = epsilon/2.")])
        self.assertEqual(items[0]["kind"], "recall")
        self.assertIn("epsilon/2", items[0]["back"])
        self.assertIn("Take delta", items[0]["back"])

    def test_replace_keeps_the_typed_answer(self):
        ov = {"d": {"why": "the equation is the task", "replace": [["Derive the line", "Find the line"]]}}
        items, _, _ = self.run_apply([typed("d", r"Derive the line with slope \(2\) through the origin.", "2x")], ov)
        self.assertEqual(items[0]["kind"], "math")
        self.assertTrue(items[0]["prompt"].startswith("Find the line"))

    def test_kind_overrides(self):
        ov = {
            "e": {"kind": "mc", "prompt": "Which number is largest?", "correct": "3", "wrong": ["1", "2"]},
            "f": {"kind": "math", "parts": [{"label": "", "answer": "4", "grading": "numeric", "tex": "4"}]},
            "g": {"kind": "recall"},
        }
        items, _, _ = self.run_apply([
            {"id": "e", "kind": "recall", "level": 1, "prompt": "Which is largest? a. 1 b. 2 c. 3", "back": "c. 3", "explain": ""},
            {"id": "f", "kind": "recall", "level": 1, "prompt": "How many corners has a square?", "back": "4", "explain": ""},
            verdict("g", "Solve the system, or say it has no solution.", "No", "No solution."),
        ], ov)
        self.assertEqual((items[0]["kind"], items[0]["correct"], items[0]["explain"]), ("mc", "3", "c. 3"))
        self.assertEqual((items[1]["kind"], items[1]["back"]), ("math", "4"))
        self.assertEqual((items[2]["kind"], items[2]["back"]), ("recall", "No solution."))
        # Applying again changes nothing.
        concepts = [{"id": "anki.topic", "items": items}]
        again = json.dumps(concepts, sort_keys=True)
        ag.apply(concepts, self.dir, ov)
        self.assertEqual(json.dumps(concepts, sort_keys=True), again)

    def test_bad_replace_and_unknown_cards(self):
        with self.assertRaises(SystemExit):
            self.run_apply([typed("h", "Find y.", "1")], {"h": {"replace": [["nowhere", "x"]]}})
        _, _, missing = self.run_apply([typed("i", "Find y.", "1")], {"zz": {"kind": "recall"}})
        self.assertEqual(missing, ["zz"])

    def test_asks_written(self):
        self.assertTrue(ag.asks_written("Prove your answer: is 7 prime?", "mc", "Yes"))
        self.assertFalse(ag.asks_written("Is 7 prime?", "mc", "Yes"))
        self.assertTrue(ag.asks_written("Derive the formula for the area of a circle.", "math"))
        self.assertFalse(ag.asks_written("Find the area of a circle of radius 2.", "math"))


if __name__ == "__main__":
    unittest.main()
