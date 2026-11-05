#!/usr/bin/env python3
"""Generate FSRS reference vectors from the Python package.

The output is written to vectors.json next to this file. Every expected value
comes from fsrs==6.3.2. Nothing reads Mathua's Go implementation.
"""

from __future__ import annotations

import importlib.metadata
import json
from datetime import datetime, timedelta, timezone
from pathlib import Path

import fsrs

HERE = Path(__file__).resolve().parent
OUT = HERE / "vectors.json"

PARAMETERS = [
    0.212,
    1.2931,
    2.3065,
    8.2956,
    6.4133,
    0.8334,
    3.0194,
    0.001,
    1.8722,
    0.1666,
    0.796,
    1.4835,
    0.0614,
    0.2629,
    1.6483,
    0.6014,
    1.8729,
    0.5425,
    0.0912,
    0.0658,
    0.1542,
]

RATINGS = [1, 2, 3, 4]
BASE = datetime(2026, 1, 1, tzinfo=timezone.utc)


def scheduler(retention: float = 0.9) -> fsrs.Scheduler:
    return fsrs.Scheduler(
        parameters=tuple(PARAMETERS),
        desired_retention=retention,
        learning_steps=(),
        relearning_steps=(),
        maximum_interval=36500,
        enable_fuzzing=False,
    )


def value(call, **kwargs):
    return float(call(**kwargs))


def main() -> None:
    s = scheduler()
    data = {
        "schema_version": 1,
        "reference": {
            "distribution": "fsrs",
            "version": importlib.metadata.version("fsrs"),
            "python": ".".join(str(part) for part in __import__("sys").version_info[:3]),
            "scheduler": {
                "parameters": PARAMETERS,
                "desired_retention": 0.9,
                "learning_steps_seconds": [],
                "relearning_steps_seconds": [],
                "maximum_interval": 36500,
                "enable_fuzzing": False,
            },
        },
        "cases": {
            "constants": {
                "decay": s._DECAY,
                "factor": s._FACTOR,
            },
            "initial_stability": [
                {"rating": r, "value": value(s._initial_stability, rating=r)}
                for r in RATINGS
            ],
            "initial_difficulty": {
                "clamped": [
                    {
                        "rating": r,
                        "value": value(s._initial_difficulty, rating=r, clamp=True),
                    }
                    for r in RATINGS
                ],
                "unclamped": [
                    {
                        "rating": r,
                        "value": value(s._initial_difficulty, rating=r, clamp=False),
                    }
                    for r in RATINGS
                ],
            },
            "next_difficulty": [
                {
                    "difficulty": d,
                    "rating": r,
                    "value": value(s._next_difficulty, difficulty=d, rating=r),
                }
                for d in (1, 5, 9, 10)
                for r in RATINGS
            ],
            "next_stability": [
                {
                    "stability": st,
                    "difficulty": d,
                    "retrievability": ret,
                    "rating": r,
                    "value": value(
                        s._next_stability,
                        stability=st,
                        difficulty=d,
                        retrievability=ret,
                        rating=r,
                    ),
                }
                for st, d, ret in (
                    (1, 1, 0.9),
                    (1, 1, 0.7),
                    (10, 5, 0.9),
                    (10, 9, 0.7),
                    (100, 1, 0.9),
                    (100, 5, 0.7),
                    (100, 9, 0.9),
                    (36500, 10, 0.1),
                    (50000, 5, 0.9),
                    (100000, 5, 0.9),
                    (0.001, 1, 0.9),
                )
                for r in RATINGS
            ],
            "retrievability": [
                {
                    "stability": st,
                    "elapsed_days": elapsed,
                    "value": s.get_card_retrievability(
                        fsrs.Card(
                            card_id=1,
                            state=fsrs.State.Review,
                            step=0,
                            stability=st,
                            difficulty=5,
                            due=BASE,
                            last_review=BASE - timedelta(days=elapsed),
                        ),
                        current_datetime=BASE,
                    ),
                }
                for st in (0.5, 1, 3, 10, 100)
                for elapsed in (0, 1, 3, 10)
            ],
            "interval": [
                {
                    "stability": st,
                    "value": s._next_interval(stability=st),
                }
                for st in (0.001, 0.01, 0.1, 0.5, 1, 2.5, 3, 10, 100, 1000, 36500, 1000000)
            ],
            "interval_for_retention": [
                {
                    "stability": st,
                    "retention": ret,
                    "value": scheduler(ret)._next_interval(stability=st),
                }
                for ret in (0.8, 0.9, 0.95)
                for st in (1, 10, 100)
            ],
        },
    }
    OUT.write_text(json.dumps(data, indent=2, sort_keys=False) + "\n", encoding="utf-8")
    print(f"wrote {OUT}")


if __name__ == "__main__":
    main()
