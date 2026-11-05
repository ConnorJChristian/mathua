# FSRS reference vectors

This directory pins the Python reference used by `internal/fsrs`. It is not a
runtime dependency of Mathua.

## Reference package

The PyPI distribution is `fsrs`, not `py-fsrs`. The installed version is
`fsrs==6.3.2`. The package reports its own version through
`importlib.metadata.version("fsrs")`.

## Setup

Use an isolated environment. The reference was generated with Python 3.15.0.
The package requires Python 3.10 or later.

```sh
python3 -m venv tools/fsrs-reference/.venv
tools/fsrs-reference/.venv/bin/python -m pip install \
  --require-hashes \
  --requirement tools/fsrs-reference/requirements.txt
```

## Regenerate the vectors

```sh
tools/fsrs-reference/.venv/bin/python \
  tools/fsrs-reference/generate_vectors.py
```

The generator writes `tools/fsrs-reference/vectors.json`. It calls only the
Python package. It never calls Mathua's Go code.

## Check the Go implementation

```sh
go test ./internal/fsrs -run TestReferenceVectors -v
```

## Scope

The reference scheduler is configured with no learning steps, no relearning
steps and fuzzing disabled. That configuration isolates the long-term model
that Mathua implements.

Mathua does not implement learning steps, relearning steps, the same-day
stability update or interval fuzzing. The conformance matrix records those
features as not applicable by design. It does not treat them as passing.
