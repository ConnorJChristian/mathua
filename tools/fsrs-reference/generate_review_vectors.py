#!/usr/bin/env python3
"""Generate reference review sequences for internal/review (the fork's FSRS wrapper).

This file belongs to the ma-econ fork, not to upstream's tools/fsrs-reference
(generate_vectors.py and vectors.json are kept byte-identical to upstream so
later fixes cherry-pick cleanly). It drives the reference *scheduler* end to
end, Scheduler.review_card, rather than its formula methods, so the pinned
values cover what upstream's vectors do not:

- the first review of a new card (S = w[rating-1], D = D0(rating); upstream's
  integration seeds from an SM-2 interval instead, one step behind);
- a review less than a day after the last (the FSRS-6 short-term formula,
  w17..w19), which upstream's integration never uses;
- lapses and relearning, gaps of more than a year, retrievability at t = 0;
- the parameter vector at the reference's lower and upper bounds, so the
  wrapper is shown to take the weights as a parameter rather than bake the
  defaults in;
- the fork's maximum interval of 365 days and desired retention of 0.9.

Every expected value comes from fsrs==6.3.2. Nothing reads Mathua's Go code.
Output: review_vectors.json next to this file.
"""

from __future__ import annotations

import importlib.metadata
import json
import sys
from datetime import datetime, timedelta, timezone
from pathlib import Path

import fsrs
from fsrs.scheduler import (
    DEFAULT_PARAMETERS,
    LOWER_BOUNDS_PARAMETERS,
    UPPER_BOUNDS_PARAMETERS,
)

HERE = Path(__file__).resolve().parent
OUT = HERE / "review_vectors.json"

BASE = datetime(2026, 1, 1, tzinfo=timezone.utc)
RATINGS = [1, 2, 3, 4]


def scheduler(parameters, retention: float = 0.9, max_interval: int = 365) -> fsrs.Scheduler:
    return fsrs.Scheduler(
        parameters=tuple(parameters),
        desired_retention=retention,
        learning_steps=(),
        relearning_steps=(),
        maximum_interval=max_interval,
        enable_fuzzing=False,
    )


def run(s: fsrs.Scheduler, events):
    """events: [(rating, days_after_previous_review)], the first gap ignored.

    Returns one record per event: the whole-day elapsed time the reference
    used, the retrievability it read before updating, and the card after.
    """
    card = fsrs.Card(card_id=1)
    at = BASE
    out = []
    for i, (rating, gap) in enumerate(events):
        if i > 0:
            at = at + timedelta(days=gap)
        elapsed = (at - card.last_review).days if card.last_review else None
        r_before = s.get_card_retrievability(card, current_datetime=at) if card.last_review else None
        card, _ = s.review_card(card, fsrs.Rating(rating), review_datetime=at)
        out.append(
            {
                "rating": rating,
                "gap_days": gap if i > 0 else 0,
                "elapsed_whole_days": elapsed,
                "retrievability_before": r_before,
                "stability": card.stability,
                "difficulty": card.difficulty,
                "interval_days": (card.due - at).days,
                "state": card.state.name.lower(),
            }
        )
    return out


def main() -> None:
    default = scheduler(DEFAULT_PARAMETERS)
    sequences = {
        # The first review, each rating: S = w[G-1], D = D0(G). Interval from S.
        **{f"first_review_rating_{r}": [(r, 0)] for r in RATINGS},
        # Same-day re-review (short-term formula) after a first Good, each rating.
        **{f"same_day_after_good_rating_{r}": [(3, 0), (r, 0.5)] for r in RATINGS},
        # Same-day after a first Again (S = 0.212): Again again, then Good.
        "same_day_after_again_again": [(1, 0), (1, 0.25)],
        "same_day_after_again_good": [(1, 0), (3, 0.25)],
        # A review 23 hours later is still the same day to the reference
        # (whole days); one 24 hours later is not.
        "twenty_three_hours_is_same_day": [(3, 0), (3, 23 / 24)],
        "twenty_four_hours_is_next_day": [(3, 0), (3, 1.0)],
        # Three same-day Goods in a row: the gain floors at x1 each time.
        "same_day_three_goods": [(3, 0), (3, 0.1), (3, 0.1), (3, 0.1)],
        # Lapses: a lapse at the due date, a relearn next day, a lapse long overdue.
        "lapse_at_due": [(3, 0), (3, 2), (1, 11), (3, 1), (3, 2)],
        "lapse_overdue": [(3, 0), (3, 2), (3, 11), (1, 60), (3, 1)],
        "two_lapses": [(3, 0), (1, 2), (1, 1), (3, 1), (3, 3)],
        "lapse_same_day_relearn": [(3, 0), (3, 2), (1, 11), (3, 0.01)],
        # Gaps over a year.
        "gap_400_days_good": [(3, 0), (3, 400)],
        "gap_400_days_again": [(3, 0), (1, 400)],
        "gap_800_days_after_s100": [(4, 0), (4, 8), (4, 50), (3, 800)],
        # Easy from the start, repeated at due: reaches the 365-day cap.
        "easy_ladder_hits_cap": [(4, 0), (4, 8), (4, 50), (4, 200), (4, 365), (4, 365)],
        # Good ladder at due, ten reviews.
        "good_ladder_ten": [(3, 0)] + [(3, None)] * 9,
        # Hard ladder.
        "hard_ladder": [(2, 0), (2, 1), (2, 2), (2, 3), (2, 5)],
    }
    # A ladder "at due" takes the previous interval as its gap.
    def materialise(events):
        out = []
        s = scheduler(DEFAULT_PARAMETERS)
        card = fsrs.Card(card_id=1)
        at = BASE
        for i, (rating, gap) in enumerate(events):
            if gap is None:
                gap = (card.due - at).days
            if i > 0:
                at = at + timedelta(days=gap)
            card, _ = s.review_card(card, fsrs.Rating(rating), review_datetime=at)
            out.append((rating, gap))
        return out

    cases = []
    for name, events in sequences.items():
        ev = materialise(events)
        cases.append(
            {
                "name": name,
                "parameters": "default",
                "desired_retention": 0.9,
                "maximum_interval": 365,
                "events": [{"rating": r, "gap_days": g} for r, g in ev],
                "steps": run(default, ev),
            }
        )

    # Boundary weights: the reference's own bounds, both ends, on a mixed sequence.
    mixed = [(3, 0), (3, 2), (1, 5), (3, 0.5), (4, 1), (2, 20), (3, 400)]
    for label, params in (("lower_bounds", LOWER_BOUNDS_PARAMETERS), ("upper_bounds", UPPER_BOUNDS_PARAMETERS)):
        s = scheduler(params)
        cases.append(
            {
                "name": f"mixed_{label}",
                "parameters": list(params),
                "desired_retention": 0.9,
                "maximum_interval": 365,
                "events": [{"rating": r, "gap_days": g} for r, g in mixed],
                "steps": run(s, mixed),
            }
        )
    # Other retentions, default weights.
    for ret in (0.85, 0.95):
        s = scheduler(DEFAULT_PARAMETERS, retention=ret)
        cases.append(
            {
                "name": f"good_ladder_retention_{ret}",
                "parameters": "default",
                "desired_retention": ret,
                "maximum_interval": 365,
                "events": [{"rating": r, "gap_days": g} for r, g in [(3, 0), (3, 2), (3, 7), (3, 20)]],
                "steps": run(s, [(3, 0), (3, 2), (3, 7), (3, 20)]),
            }
        )

    # Retrievability of a stored state at t = 0 and at whole and fractional days
    # (the reference floors to whole days).
    retr = []
    for st in (0.212, 2.3065, 30.0, 400.0):
        for elapsed in (0, 0.5, 0.999, 1, 1.5, 7, 30, 365, 400):
            card = fsrs.Card(
                card_id=1,
                state=fsrs.State.Review,
                step=None,
                stability=st,
                difficulty=5.0,
                due=BASE,
                last_review=BASE - timedelta(days=elapsed),
            )
            retr.append(
                {
                    "stability": st,
                    "elapsed_days": elapsed,
                    "value": default.get_card_retrievability(card, current_datetime=BASE),
                }
            )

    data = {
        "schema_version": 1,
        "reference": {
            "distribution": "fsrs",
            "version": importlib.metadata.version("fsrs"),
            "python": ".".join(str(part) for part in sys.version_info[:3]),
            "default_parameters": list(DEFAULT_PARAMETERS),
            "lower_bounds": list(LOWER_BOUNDS_PARAMETERS),
            "upper_bounds": list(UPPER_BOUNDS_PARAMETERS),
            "learning_steps_seconds": [],
            "relearning_steps_seconds": [],
            "enable_fuzzing": False,
        },
        "sequences": cases,
        "retrievability": retr,
    }
    OUT.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")
    n = sum(len(c["steps"]) for c in cases)
    print(f"wrote {OUT}: {len(cases)} sequences, {n} reviews, {len(retr)} retrievability values")


if __name__ == "__main__":
    main()
