# Fork addition: review sequences (ma-econ)

`README.md`, `conformance.md`, `generate_vectors.py`, `requirements.txt` and
`vectors.json` in this directory are byte-identical to upstream, so upstream
fixes cherry-pick cleanly. This file and the two below belong to the fork.

`generate_review_vectors.py` and `review_vectors.json` drive
`Scheduler.review_card` end to end (the first review, the same-day path,
lapses, gaps over a year, the reference's parameter bounds, a 365-day
maximum interval). `internal/review` (the fork's wrapper) is pinned against
them in `TestReview_MatchesReferenceSequences`.

```sh
/opt/py311venv/bin/python -I tools/fsrs-reference/generate_review_vectors.py
go test ./internal/review -run 'Reference|Retrievability_Matches' -v
```
