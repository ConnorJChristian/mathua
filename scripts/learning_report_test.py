"""Tests for learning_report.py: python3 -m unittest scripts/learning_report_test.py

The due rule (is_due, due_at, overdue_days) is the engine's (internal/review,
FORK.md "What is due"). Both read the same fixture,
internal/review/testdata/due_cases.json, so the report and the app cannot
disagree about a row."""
import datetime as dt
import json
import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import learning_report as lr  # noqa: E402

FIXTURE = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
                       "internal", "review", "testdata", "due_cases.json")


class DueRuleTest(unittest.TestCase):
    def setUp(self):
        with open(FIXTURE) as f:
            self.fixture = json.load(f)
        self.now = dt.datetime(2026, 11, 6, 12, 0, tzinfo=dt.timezone.utc)

    def row(self, c):
        at = lambda d: None if d is None else self.now + dt.timedelta(days=d)  # noqa: E731
        return {"status": c["status"], "lr": at(c["last_reviewed"]), "la": at(c["last_attempted"]),
                "due": at(c["next_review_due"])}

    def test_decay_window_matches_go(self):
        self.assertEqual(self.fixture["decay_days"], lr.DECAY_DAYS)

    def test_cases(self):
        self.assertTrue(self.fixture["cases"])
        for c in self.fixture["cases"]:
            with self.subTest(c["name"]):
                p = self.row(c)
                self.assertEqual(lr.is_due(p, self.now), c["due"])
                self.assertAlmostEqual(lr.overdue_days(p, self.now), c["overdue"], places=6)


if __name__ == "__main__":
    unittest.main()
