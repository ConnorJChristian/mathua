"""Tests for anki_levels.py: python3 -m unittest scripts/anki_levels_test.py

The cards are invented fixtures, not anyone's deck."""
import json
import os
import sys
import tempfile
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import anki_levels as al  # noqa: E402


def recall(i, front, back, **kw):
    return dict({"id": i, "kind": "recall", "level": 2, "prompt": front, "back": back, "explain": ""}, **kw)


def math(i, front, n_parts=1):
    parts = [{"label": "", "answer": "1", "grading": "numeric", "tex": "1"}] * n_parts
    return {"id": i, "kind": "math", "level": 2, "prompt": front, "parts": parts, "back": "", "explain": ""}


def mc(i, front, correct, wrong):
    return {"id": i, "kind": "mc", "level": 2, "prompt": front, "correct": correct, "wrong": wrong, "explain": ""}


LONG_SETTING = ("A bakery sells loaves at a price of 4 dollars each and has fixed costs of 300 dollars a week "
                "and a marginal cost of 1.5 dollars per loaf. How many loaves a week must it sell to break even, "
                "and what is its profit at 200 loaves?")


class LevelTest(unittest.TestCase):
    def lv(self, item, **meta):
        return al.level_of(item, meta)[0]

    def test_step_one(self):
        self.assertEqual(self.lv(math("a", r"Differentiate \(x^5\).")), 1)
        self.assertEqual(self.lv(recall("b", "What is a scalar?", "A single number.")), 1)
        self.assertEqual(self.lv(recall("c", "The slope of a flat line is [...].", "The slope of a flat line is zero."),
                                 notetype="Cloze", cloze=1), 1)
        self.assertEqual(self.lv(mc("d", "True or false: every square matrix is invertible.", "False", ["True"])), 1)
        self.assertEqual(self.lv(mc("e", "A matrix with one row is a _____ vector. Which fills the blank?",
                                    "row", ["column", "unit"])), 1)

    def test_step_two(self):
        self.assertEqual(self.lv(recall("f", r"Use the product rule to find the derivative of \(x^2 e^x\).", "..."),
                                 tags="format::worked_example"), 2)
        self.assertEqual(self.lv(math("g", r"Solve the system \(x + y = 3\), \(x - y = 1\) for x and y.", 2)), 2)
        self.assertEqual(self.lv(recall("h", "A [...] has [...] rows and [...] columns, and [...] entries.", "..."),
                                 notetype="Cloze", cloze=4), 2)

    def test_step_three(self):
        self.assertEqual(self.lv(recall("i", "Prove that the sum of two even integers is even.", "...")), 3)
        self.assertEqual(self.lv(recall("j", "Let V be a vector space.", "..."), tags="format::proof"), 3)
        self.assertEqual(self.lv(recall("k", LONG_SETTING, "...")), 3)
        self.assertEqual(self.lv(recall("l", "Simplify the expression.", "..."), deck="Book::Ch 2::Exercises"), 3)
        self.assertEqual(self.lv(recall("m", "For the function f: (a) find its critical points; (b) classify each one; "
                                              "(c) find the global maximum and the global minimum on the closed interval from 0 to 4 and say where each is attained.",
                                    "...")), 3)

    def test_written_choice_stays_low(self):
        # A true/false check whose explanation is the back is not a proof.
        self.assertEqual(self.lv(mc("n", "Is the zero vector in every subspace? Explain why or why not.", "Yes", ["No"])), 1)

    def test_overrides(self):
        ov = {"cards": {"o": 3}, "decks": {"Book::Ch 1": 1, "Ch 1::Harder": 3}}
        item = recall("o", "What is a scalar?", "A number.")
        self.assertEqual(al.decide(item, {"deck": "Book::Ch 1"}, ov), (3, "override (card)"))
        other = recall("p", "Prove it.", "...")
        self.assertEqual(al.decide(other, {"deck": "Book::Ch 1::Part"}, ov)[0], 1)
        # The longest matching deck key wins.
        self.assertEqual(al.decide(other, {"deck": "Book::Ch 1::Harder"}, ov)[0], 3)
        self.assertEqual(al.decide(other, {"deck": "Book::Ch 2"}, ov)[0], 3)  # no override: the rules

    def test_cloze_count(self):
        self.assertEqual(al.cloze_count("{{c1::a}} and {{c1::b}} and {{c2::c}}", 0), 2)
        self.assertEqual(al.cloze_count("{{c1::a}} and {{c2::c}}", 1), 1)

    def test_apply_records_and_reapplies(self):
        d = tempfile.mkdtemp()
        with open(os.path.join(d, al.OVERRIDES_FILE), "w") as f:
            json.dump({"cards": {"q2": 3}}, f)
        concepts = [{"id": "anki.topic_one.2", "items": [
            recall("q1", "What is a scalar?", "A single number."),
            recall("q2", "What is a vector?", "A list of numbers."),
        ]}]
        meta = {"q1": {"deck": "Book::Ch 1", "tags": "topic::x"}}
        changed, hist, records = al.apply(concepts, meta, d)
        self.assertEqual([it["level"] for it in concepts[0]["items"]], [1, 3])
        self.assertEqual(changed, {"q1": (2, 1), "q2": (2, 3)})
        self.assertEqual(dict(hist["topic_one"]), {1: 1, 3: 1})
        saved = json.load(open(os.path.join(d, al.LEVELS_FILE)))
        self.assertEqual(saved["q1"], {"level": 1, "rule": records["q1"]["rule"], "deck": "Book::Ch 1", "tags": "topic::x"})
        self.assertEqual(saved["q2"]["rule"], "override (card)")
        # The recorded metadata is enough to apply again without the export.
        self.assertEqual(al.meta_from_levels(d)["q1"], {"deck": "Book::Ch 1", "tags": "topic::x"})

    def test_bad_override(self):
        d = tempfile.mkdtemp()
        with open(os.path.join(d, al.OVERRIDES_FILE), "w") as f:
            json.dump({"decks": {"X": 4}}, f)
        with self.assertRaises(SystemExit):
            al.load_overrides(d)


if __name__ == "__main__":
    unittest.main()
