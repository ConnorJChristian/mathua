# FSRS-6 conformance matrix

This matrix compares the pure functions in `internal/fsrs` with the pinned
Python reference `fsrs==6.3.2`. It is evidence for Phase 2 and Phase 3. It does
not change production formulas. It is separate from the replay harness.

## Reference environment

| Item | Value |
|---|---|
| Distribution | `fsrs` |
| Version | `6.3.2` |
| Python | 3.15.0 |
| Install | `tools/fsrs-reference/requirements.txt` with `--require-hashes` |
| Scheduler parameters | FSRS-6 21 default parameters |
| Desired retention | 0.9 |
| Learning steps | none |
| Relearning steps | none |
| Maximum interval | 36500 days |
| Fuzzing | disabled |
| Reference package dependency | `typing_extensions==4.16.0` |

The PyPI distribution name is `fsrs`. There is no `py-fsrs==6.3.2`
distribution on PyPI.

Regenerate the evidence:

```sh
tools/fsrs-reference/.venv/bin/python \
  tools/fsrs-reference/generate_vectors.py
go test ./internal/fsrs -run TestReferenceVectors -v
```

The generator calls only the Python package. It never calls Mathua's Go code.

## Vector set

`tools/fsrs-reference/vectors.json` holds 115 reference values:

| Group | Values |
|---|---:|
| Constants | 2 |
| Initial stability | 4 |
| Initial difficulty, clamped | 4 |
| Initial difficulty, unclamped | 4 |
| Next difficulty | 16 |
| Next stability | 44 |
| Retrievability | 20 |
| Interval | 12 |
| Interval for retention | 9 |

Floating-point equality uses a relative tolerance of `1e-12`. Interval values
are compared as exact integers.

## Matrix

Status meanings:

- `MATCHES_REFERENCE`: every compared value agrees within tolerance, or every
  integer agrees exactly.
- `DIFFERS_FROM_REFERENCE`: a confirmed difference exists. No row currently carries this status.
- `MISSING`: the reference has a surface with no Mathua equivalent. This is not
  a value mismatch.
- `NOT_APPLICABLE_BY_DESIGN`: Mathua deliberately implements a smaller model.
- `UNVERIFIED`: the behaviour was not exercised by this vector set or depends on
  a later phase.

| Surface | Cases | Status | Evidence |
|---|---:|---|---|
| Reference identity | 1 | `MATCHES_REFERENCE` | Distribution `fsrs`, version `6.3.2`, Python 3.15.0, no learning steps, no relearning steps, maximum interval 36500, fuzzing disabled. |
| 21 default parameters | 21 | `MATCHES_REFERENCE` | Every parameter is byte-identical to the reference scheduler. |
| Decay and factor | 2 | `MATCHES_REFERENCE` | `decay = -0.1542`, `factor = 0.9803464944134797`. |
| Initial stability, ratings Again/Hard/Good/Easy | 4 | `MATCHES_REFERENCE` | `0.212`, `1.2931`, `2.3065`, `8.2956`. |
| Initial difficulty, clamped | 4 | `MATCHES_REFERENCE` | Values agree, including the Easy clamp to 1.0. |
| Initial difficulty, unclamped | 4 | `MATCHES_REFERENCE` | Values agree, including the negative Easy value `-4.771630703161737`. |
| Next difficulty from 1, 5, 9 and 10 for all ratings | 16 | `MATCHES_REFERENCE` | Every value agrees. |
| Next stability for 44 long-term states, including stability 36500, 50000 and 100000 | 44 | `MATCHES_REFERENCE` | Values agree for Again, Hard, Good and Easy. The former cap at 36500 is gone. Above-cap inputs now match the reference. |
| Retrievability at 5 stabilities and 4 elapsed days | 20 | `MATCHES_REFERENCE` | Values agree from elapsed day 0 to day 10. |
| Interval for 12 stabilities from 0.001 to 1000000 | 12 | `MATCHES_REFERENCE` | Exact integers agree. Small stabilities map to 1. Large stabilities map to 36500. |
| Interval for retention 0.8, 0.9 and 0.95 | 9 | `MATCHES_REFERENCE` | Exact integers agree, including retention values other than the default. |
| Same-day stability update | 0 | `NOT_APPLICABLE_BY_DESIGN` | `internal/fsrs` implements the long-term model only. Its package documentation explains why a second same-day attempt is practice, not a scheduled review. |
| Learning steps | 0 | `NOT_APPLICABLE_BY_DESIGN` | Mathua schedules in days. It has no sub-day learning queue. |
| Relearning steps | 0 | `NOT_APPLICABLE_BY_DESIGN` | Same reason as learning steps. |
| Interval fuzzing | 0 | `NOT_APPLICABLE_BY_DESIGN` | Deterministic intervals are a product requirement. The reference ran with fuzzing disabled. |
| Full `Scheduler.review_card` state machine across Learning, Review and Relearning | 0 | `MISSING` | Mathua has no stateful scheduler port. Production composes the pure functions inside `computeGradedAttempt` and the scheduler. Phase 2 and Phase 3 do not compare that integration. |
| SM-2 seeding (`SeedFromInterval`) | 0 | `NOT_APPLICABLE_BY_DESIGN` | Mathua-specific migration path. The reference has no equivalent function. |
| Production end-to-end scheduling equivalence | 0 | `UNVERIFIED` | The Phase 3 replay harness compares synthetic histories, not production histories. This matrix covers pure functions only. |
| Invalid ratings and non-finite inputs | 0 | `UNVERIFIED` | Mathua now has an explicit fail-closed policy and `TestNonFiniteInputsFailClosed`. The Python reference does not expose a comparable total-function contract, so this matrix does not claim reference agreement for invalid input. |

## Correction record

Phase 2 confirmed one mismatch at stability 36500, difficulty 10 and
retrievability 0.1 for Hard, Good and Easy. Mathua returned 36500. The reference
returned 62467.61564449483, 79678.60931908019 and 117369.21739370529.

Phase 3 removed the upper stability clamp from `NextStability` and its input.
The same values now match the reference. The regression vectors also cover input
stability 50000 and 100000, which Mathua previously reduced to 36500. The
published interval remains capped at 36500 by `Interval`.

## Documentation correction

`internal/fsrs/fsrs.go` previously said the reference interval helper could
return 0 and that Mathua clamped to 1 as an intentional difference. In
`fsrs==6.3.2` the reference already clamps `_next_interval` to at least 1. The
generated vectors confirm that stability 0.001, 0.01 and 0.1 all produce
interval 1 in both implementations. Phase 3 corrected the comment. The code and
the reference now agree, and no intentional interval difference is claimed.

## Unverified areas

- PostgreSQL parity remains unverified because `TEST_POSTGRES_DSN` is unset.
- The replay harness compares synthetic fixtures only. It does not
  replay production histories, so no claim is made about schedule equivalence
  on real learner data.
- The conformance test covers sampled values, not an exhaustive proof over the
  real-number domain.
