"""Tests for anki_sources.py: python3 -m unittest scripts/anki_sources_test.py

The decks and tags are invented fixtures shaped like an export's."""
import json
import os
import sys
import tempfile
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import anki_sources as src  # noqa: E402


class SourcesTest(unittest.TestCase):
    def setUp(self):
        self.dir = tempfile.mkdtemp()
        with open(os.path.join(self.dir, "subjects.json"), "w") as f:
            json.dump({"subjects": [], "books": {"tags": {"mmbef": "schaums-math-methods"},
                                                 "decks": {"Course::Week 3": "leithold"}}}, f)

    def test_derivation(self):
        recs = {
            "a": {"deck": "Course::Part 2::05 Derivatives", "tags": "source::Hoy_Mathematics_for_Economics::ch05 topic::x"},
            "b": {"deck": "Course::Module 2", "tags": "source::Schaums_Calc_Econ::ch06"},
            "c": {"deck": "Course::Module 4", "tags": "calculus mmbef schaum"},
            "d": {"deck": "Course::Week 3::Practice", "tags": ""},
            "e": {"deck": "Course::Data::Python Data Science Handbook::Ch5", "tags": ""},
            "f": {"deck": "Wooldridge - Introductory Econometrics 7e::Ch02 The Simple Regression Model", "tags": ""},
            "g": {"deck": "Course::Module 1: Introduction", "tags": "daily"},
            "h": {"deck": "Course::ISLP::Ch3 Linear Regression"},
        }
        got = src.apply(recs, self.dir)
        self.assertEqual(got, {"a": "hoy", "b": "schaums-calc-econ", "c": "schaums-math-methods",
                               "d": "leithold", "e": "pdsh", "f": "wooldridge", "h": "islp"})
        with open(os.path.join(self.dir, src.SOURCES_FILE)) as f:
            self.assertEqual(json.load(f), got)

    def test_key_matches_the_server(self):
        # internal/books.Key
        self.assertEqual(src.key("ma-program/hoy-3e.pdf"), "hoy-3e")
        self.assertEqual(src.key("Schaum's Outline of Calculus"), "schaums-outline-of-calculus")
        self.assertEqual(src.key("Axler_Linear_Algebra_Done_Right"), "axler-linear-algebra-done-right")
        self.assertEqual(src.Registry().known("Mathematical Economics I (Hoy et al., Schaum's, Axler)"), "")


if __name__ == "__main__":
    unittest.main()
