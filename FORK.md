# MA Economics fork

This fork is upstream Mathua plus **modules**. The base install (engine, web
app, mathematics corpus) is unchanged; everything this fork adds lives in a
module folder that can be switched off. See "Modules" below.

The `ma-econ` module adds two courses and one destination. It adds no new
concepts or generators: everything uses the existing graph.

| ID | Course | Targets | Concepts incl. prerequisites |
|---|---|---|---|
| `econ5153` | Math for Economists (ECON 5153) | 42 | 143 |
| `statlearn` | Statistical Learning (ISLR companion) | 25 | 129 |

Destination `ma-econ` chains both courses. Neither course pulls in any
trigonometry. The only geometry prerequisite is plotting a point.

The placement diagnostic credits prerequisites when you answer harder
questions correctly, so arithmetic and algebra basics are skipped unless you
miss something.

## Run it

Requires Go 1.26+ and Node 20+.

```bash
make local        # builds the UI against your own server, then serves http://localhost:8080
```

Sign up in the app, take the diagnostic, then pick the "MA Economics"
destination. Progress is stored in `mathua.db` in the repo root.

Use `make local`, not `npm run build` alone. The default UI build
(`web/next-app/.env.production`) talks to the upstream hosted API, which does not
have these courses.

## What these courses cover

**econ5153:** derivatives and basic integration (substitution, FTC) incl. exp/log and implicit differentiation, convexity
and Jensen, marginal analysis, single-variable optimization, partial
derivatives, gradient, directional derivatives, multivariable chain rule,
Jacobian, tangent plane, second-order Taylor, the two-variable second-derivative
test, extrema on closed regions, Lagrange multipliers (one and two constraints),
implicit function theorem, Euler's homogeneous function theorem; matrix
inverses, determinants, Cramer's rule, rank and rank-nullity, eigenvalues,
diagonalization, definiteness of quadratic forms; linear recurrences,
first-order linear and second-order homogeneous ODEs, linear systems, phase
portraits, equilibria and stability.

**statlearn:** expectation, variance, covariance, correlation, conditional
expectation, CLT, Bayes' theorem; sampling distributions, standard errors,
confidence intervals, t statistics, p-values, Type I/II errors; least-squares
line, projection onto a subspace, R², prediction, gradient descent;
overfitting, cross-validation, ridge penalty, logistic regression, feature
scaling, one-hot encoding; k-means, Gini impurity, bagging and random forests,
PCA variance explained; confusion matrix, precision/recall/F1, ROC, class
imbalance.

One quirk: upstream's graph routes logistic regression and KNN through a few
neural-net basics (perceptron, forward pass, sigmoid activation, cross-entropy
loss), so `statlearn` includes those four as prerequisites.

## Practising locked topics

Upstream refuses to serve a topic until its prerequisites are mastered. In this
fork, a locked topic's page has a **Practice anyway** button, as in Math Academy.
It applies only to the topic you picked by hand; recommendations are unchanged.

Mastering a topic credits every prerequisite below it that you haven't mastered.
Credited topics count as done for your plan, so you aren't sent back to drill
them. Each correct answer on a topic also keeps its credited prerequisites
fresh, so they only come back for review if you stop working above them.
Topics you mastered yourself keep their own review schedule.

## Not covered (use ISLR / course notes)

- Multiple regression in matrix form (β̂ = (X'X)⁻¹X'y) and coefficient inference
- Lasso, bootstrap, bias–variance decomposition, LDA/QDA, SVMs
- Econ framing of dynamics: cobweb models, saddle paths, discrete-time stability
- Causal inference (DAGs, d-separation, adjustment sets)

## Editing the courses

These courses live in `modules/ma-econ/courses.json` (the destination in
`destinations.json` beside it) as lists of target concept IDs, read at startup.
Restart the server after editing. Concept IDs are in `data/concepts/*.json`.
Run `go run scripts/validate_graph.go` after any change.

## Modules

```
data/                 upstream Mathua's corpus (unchanged)
modules/              committed modules: original content, safe to push
  ma-econ/            the two courses and the MA Economics destination
  python-dsh/         the Python for Data Science track
modules-local/        your own modules: git-ignored, never pushed
modules.local.json    {"disabled": ["..."]}: switch modules off (git-ignored)
```

A module is a folder with a `module.json` and any of `concepts/`, `bank/`,
`readings.json`, `courses.json`, `destinations.json` and `files/`. The
server and the web build read every enabled module. A module can declare a
**track**, which adds an option to the header switch (Math | Python | ...).
`modules-local/README.md` shows the layout.

Use `modules-local/` for anything made from your own books, notes or Anki
decks. Its topics can point at the section to read with `readings.json`.
Learn then shows **Open section**, which opens your PDF from `files/` at that
page through your local server. The questions come from the module's bank.

Add, remove or switch off a module, then rebuild:

```bash
docker compose up --build
```

The rebuild is needed because the web app bundles the concept list.
`make local` includes your local modules too. It rewrites
`web/next-app/data/concepts.json`, so run `go run ./scripts/merge_concepts`
before you commit. A test fails if local module content is committed.

### Anki cards

`scripts/import_anki.py` turns an Anki export (`.colpkg` or `.apkg`) into a
local module of recall cards:

```bash
pip install zstandard
python3 scripts/import_anki.py collection.colpkg --config modules-local/anki/anki.json
```

The config chooses which decks to import and which domain and track each
becomes; the script's docstring shows the format. Each deck becomes a topic,
and large decks are split into sets of about 60 cards along your `topic::`
tags. Card images are copied into the module's `files/media/`. Your Anki
schedule is not imported: Mathua schedules the cards from your answers here.
Re-run the import after you export again; it replaces the cards and keeps
the rest of the module.

## Python for Data Science track

The `python-dsh` module adds the **Python** side of the header switch. Same
site, same login, same XP and review schedule. The switch only changes which
topics the Learn shelf, the topic browser, reviews and the quiz offer.

The track follows the five chapters of Jake VanderPlas's *Python Data Science
Handbook*. Every topic links to its book section. The questions and notes are
original: the book's text is CC-BY-NC-ND, so nothing here adapts it.

| Chapter | Domain | Topics | Questions |
|---|---|---|---|
| 1. IPython | `python_ipython` | 9 | 60 |
| 2. NumPy | `python_numpy` | 20 | 127 |
| 3. Pandas | `python_pandas` | 18 | 114 |
| 4. Matplotlib | `python_matplotlib` | 15 | 92 |
| 5. Machine Learning | `python_ml` | 16 | 114 |

Section 5.14 (image features) is not covered: it needs scikit-image. Section
4.14 (Seaborn) uses pandas plotting, because Seaborn is not installed.

Each topic mixes three kinds of question:

- **Multiple choice** on the concept.
- **What does this print?** Type the output. Spacing inside arrays does not
  matter; everything else does.
- **Code tasks.** Write code in the box. **Run** shows output and plots.
  **Check** grades it against hidden tests on the server.

A question bank can also hold **recall cards**, which work like flashcards.
You answer in your head, reveal the back, then mark whether you had it.

The Python track has no placement test. Its first topics have no
prerequisites. Mastering a topic credits the topics below it, as on the math
side.

### How code runs

The server starts one Python process per Run or Check
(`grading/pyrunner.py`). The process has a time limit (8 seconds of CPU by
default, `PYRUNNER_TIME_LIMIT`), a memory limit and a guard that refuses
network access, subprocesses and writes outside its own temporary folder. In
the Docker image it also runs as the `nobody` user. This protects a personal
study server from mistakes. It is not a sandbox for untrusted strangers: do
not expose the server to the public internet with the Python track on.

The image installs numpy, pandas, matplotlib and scikit-learn at the versions
in `grading/requirements-python.txt`.

### Editing the questions

Questions are written in `modules/<id>/src/<NN>_<chapter>.py`, one file per
chapter (`modules-local/` works the same way). Build them with the pinned
libraries installed:

```bash
python3 scripts/python_bank.py build             # every module
python3 scripts/python_bank.py build python-dsh  # one module
python3 scripts/python_bank.py build numpy       # one chapter
```

The build runs every snippet. It computes the expected output of each "what
does this print" question, checks that each reference solution passes its
hidden tests and that each starter fails them, and writes the module's
`bank/*.json` and `concepts/*.json`.

Then rebuild the image. The image build runs
`python3 scripts/python_bank.py check` and fails if a stored output no longer
matches what the pinned libraries print.
