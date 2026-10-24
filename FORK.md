# MA Economics fork

A fork of [Mathua](https://github.com/chuma-beep/mathua), an open-source
adaptive learning engine (Go backend, Next.js frontend, a concept graph with
prerequisites, spaced review, placement diagnostics), reshaped into a
course-centred study app for a graduate economics programme. The engine and
the mathematics corpus are upstream's; everything the fork adds lives in
modules that can be switched off, plus the pages that present them.

What the fork adds:

- **Courses, not destinations.** A course is a study plan (topics in teaching
  order, split into modules), with its own page (Home, Modules, Readings,
  Lectures, Grades), its texts and a due date. The **Study** tab is one queue
  across every course you follow: due reviews first, then the topics you pinned
  for this week, then the engine's picks, paced by each course's deadline.
  Placement is course-scoped: the diagnostic covers the prerequisites of the
  courses you follow.
- **Question banks as modules.** A module folder carries concepts, a question
  bank (multiple choice, typed answers, Python code graded by hidden tests,
  flip cards as a last resort), readings into your own PDFs or online books,
  lecture notes and videos. Public modules in this repository: `modules/ma-econ`
  (Math for Economists and a Statistical Learning companion, built on the
  upstream graph) and `modules/python-dsh` (the Python Data Science Handbook as
  a track). The owner's own modules (textbook-derived cards, course lecture
  notebooks, a political-economy reading course) live in `modules-local/`,
  which is git-ignored.
- **Library.** Every text the installed modules read from, as a shelf of books
  with tables of contents that link to the topics; PDFs open in an embedded
  reader at the right page.
- **Importers and authoring tools.** An Anki collection importer that files
  cards under topics and turns flip cards into answerable questions where it
  can; a Python authoring API for question banks (`scripts/python_bank.py`).

The `ma-econ` module adds two courses. It adds no new concepts or generators:
everything uses the existing graph.

| ID | Course | Targets | Concepts incl. prerequisites |
|---|---|---|---|
| `econ5153` | Math for Economists (ECON 5153) | 42 | 143 |
| `statlearn` | Statistical Learning (ISLR companion) | 25 | 129 |

Neither course pulls in any trigonometry. The only geometry prerequisite is
plotting a point. The placement diagnostic credits prerequisites when you
answer harder questions correctly, so arithmetic and algebra basics are
skipped unless you miss something.

## Licence

MIT, the same as upstream (see `LICENSE`). Use it, copy it, change it,
redistribute it, commercially or not, with attribution. The fork's own code
and the public modules are published under the same terms. Material derived
from copyrighted textbooks is deliberately kept out of the repository.

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

Mastering a topic gives **partial credit** to the prerequisites below it, as
in Math Academy:

- A direct prerequisite gets half a credit. A topic two links down gets a
  quarter, and one three links down gets an eighth. Anything further gets
  nothing.
- A topic that lists a prerequisite under `encompasses` passes it a full
  credit, because practising the topic practises all of that prerequisite.
- Credit adds up. A prerequisite reaches full credit when, for example, you
  master two topics directly above it. It then counts as done for your plan,
  so you are not sent back to drill it.

A correct answer keeps credited prerequisites fresh only through a strong link
(half a credit or more). Weakly linked credit comes back for review. Topics you
mastered yourself keep their own review schedule.

## Not covered (use ISLR / course notes)

- Multiple regression in matrix form (β̂ = (X'X)⁻¹X'y) and coefficient inference
- Lasso, bootstrap, bias–variance decomposition, LDA/QDA, SVMs
- Econ framing of dynamics: cobweb models, saddle paths, discrete-time stability
- Causal inference (DAGs, d-separation, adjustment sets)

## Editing the courses

These courses live in `modules/ma-econ/courses.json` (the destination in
`destinations.json` beside it) as lists of target concept IDs, read at startup.
Restart the server after editing. Concept IDs are in `data/concepts/*.json`.
Run `go run scripts/validate_graph.go` after any change. It checks the graph
the install runs with, modules included, for cycles, dangling prerequisites
and duplicate labels.

**Study plans.** A course can carry an ordered `plan` instead of (or as well
as) `targets`: its topics in the order they are taught. The course's path
keeps that order, putting each topic's unmet prerequisites just before it,
and with the course active the shelf offers untouched topics earlier in the
plan first. A course with only a plan targets its plan.

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

**One graph across tracks.** A track decides what the shelf offers. It does
not wall off prerequisites. A concept can come after a concept in any
domain or track: an econometrics topic after Mathua's coefficient of
determination, a code task after a pandas topic. The engine sequences the
graph as one:

- A concept unlocks from your whole record, in every track.
- When a track's topic waits on a concept from another track that is ready
  to learn, this track's shelf offers it, marked **Needed next**.
- When everything in a track waits on other tracks, the shelf offers the
  track's entry topics as **Jump in**. You can practise a locked topic.
- Mastering a concept gives partial credit to the concepts below it, in every
  track. The credit halves with each link.
- A goal (destination) plans and places across tracks. Its placement test
  covers the goal's prerequisites in every domain. It skips question-bank
  practice sets, such as flashcards and code tasks.

Use `modules-local/` for anything made from your own books, notes or Anki
decks. Its topics can point at the section to read with `readings.json`.
Learn then shows **Open section**, which opens your PDF from `files/` at that
page through your local server. The questions come from the module's bank.
A reading can also point at a base-graph concept (say `calc.multi.lagrange`
at your textbook's section); its lesson then starts with that step.

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
local module. Where a card allows it, the card becomes a Mathua question:

- **Typed answer.** The answer at the top of the back is a formula or a
  number, or a few labelled ones such as `x_0 = 60` and `y_0 = 40`. You type
  each one in Mathua's math editor. Mathua's own graders check it, so any
  equivalent form counts.
- **Choice.** The back starts with True, False, Yes or No. You pick one.
- **Flip card.** Every other card. You reveal the back and say whether you
  had it.

The import checks each typed answer before it keeps it. The book's answer,
typed into the math editor, must grade correct. A wrong answer must grade
wrong. A card that fails either check stays a flip card. An answer is also
kept as a flip card when the question already shows an equal expression,
for example "simplify" and "factor" questions.

```bash
pip install zstandard sympy "antlr4-python3-runtime==4.11.*"
(cd web/next-app && npm ci)
python3 scripts/import_anki.py collection.colpkg --config modules-local/anki/anki.json
```

**Sort by subject.** By default each deck becomes a domain. A config can
name a `subjects_file` instead. The file maps your cards to subjects, and
each subject to a Mathua domain and the concepts it follows:

```json
{"id": "derivative_rules", "domain": "calculus", "subdomain": "calculus.derivatives",
 "label": "Derivative rules", "after": ["calc.deriv.chain_rule"],
 "topics": ["derivative_and_the_rules_of_differentiation"],
 "decks": ["Module 2: One Variable Calculus (basic)"]}
```

A card goes to the subject that lists its `topic::` tag. A card without a
tag goes by its deck folder. A `routes` list can re-sort cards by their
text.

The practice sets then appear in Mathua's own domains, after the
concepts named in `after`. Use a new domain only where Mathua has none.
Placement tests skip these sets: they are practice, not placement
material.

The typed-answer check needs Node with the web app's packages, Go and SymPy.
Use `--no-typed` to import flip cards only. Use `--report rejects.json` to
see which typed answers failed the check, and why.

The config chooses which decks to import and which domain and track each
becomes; the script's docstring shows the format. Each deck becomes a topic,
and large decks are split into sets of about 60 cards along your `topic::`
tags. Card images are copied into the module's `files/media/`. Your Anki
schedule is not imported: Mathua schedules the cards from your answers here.
Re-run the import after you export again; it replaces the cards and keeps
the rest of the module.

### Code tasks from your own material

A local module can author code tasks the same way the Python track does:
`modules-local/<id>/src/NN_*.py`, built with
`python3 scripts/python_bank.py build <id>`. Use `topic()` instead of
`concept()` to name your own source. The runner image includes
`statsmodels`, the `wooldridge` data sets and `ISLP` (data and model helpers
only), so tasks can use real econometrics data with no network.

## Library, readings and course texts

- **Library** (`/library`): every book and source your topics come from,
  shelved by track. A book's contents list its sections and the topics each
  teaches; your PDFs open beside the contents at the section.
- **Embedded reader:** a topic with a reading in one of your PDFs opens it
  inside the lesson at the section's first page (pdf.js, so it works on
  phones too), with the pages to read.
- **Several texts per topic:** a module's `readings.json` may give a concept
  a list of readings, one per text. A course lists its authority texts in
  priority order (`"texts": ["ma-program/hoy-3e.pdf", "web:Statistical
  Learning"]`); a topic reached through that course (from Topics with the
  plan selected, or as one of the plans you follow) opens in that text, and
  the others are one click away. `web:` keys match online books and decks by
  name.
- A module's `books.json` names its PDFs: `{"file.pdf": {"title": ...,
  "author": ..., "source_prefix": ...}}`.

## Courses, the Study tab and your cards

The app opens on **Study** (`/today`): one queue across every course you
follow, with the mastery test and the review session surfaced when due.
**Courses** (`/courses`) shows each followed course as a card; a course's
own page (`/course?id=`) has Home, Modules, Readings, Videos and Grades.
A course's `courses.json` entry may carry `term` and `modules`
(`[{"title", "topics": [...]}]`, each a slice of its plan); `videos.json`
in a module maps a topic id or course id to `[{"title", "url"}]` (YouTube)
or `[{"title", "file": "lecture.mp4"}]` (a file in `files/`).

Recommendations put a followed course's path first (its plan and the
prerequisites the plan needs); a due review still comes before them, and a
topic missed on the last mastery test before new work. When a plan is still
locked, the course page offers a placement test scoped to it
(`/onboard?course=`), and a correct placement answer opens the topic's
prerequisites too.

Imported flashcards come up in the practice of the topics they are about.
A module's `cards.json` (`{"<topic>": [{"deck", "item"}, ...]}`) files each
card under a topic; a topic nothing is filed under takes the whole deck
that lists it as a prerequisite. About half of a topic's questions are
cards (`MATHUA_CARD_SHARE`). A recall card the learner grades themselves is
practice only: it never moves mastery, counts toward a halt or reschedules
a review. The mastery test covers only topics studied on followed courses.

## Learning reports

`scripts/learning_report.py` reads a copy of the app's database and writes a
report for one learner: activity and accuracy by week, study-plan pace
against a deadline, stuck topics with the prerequisite most likely behind
them, slow topics, the review backlog, and questions that look broken rather
than hard (for example a rounded decimal rejected against a fraction key).

The database lives in a Docker volume. To read it from outside Docker, copy
`docker-compose.override.example.yml` to `docker-compose.override.yml`; it
adds a small service that snapshots the database to `local-data/` every
hour. Then:

```bash
python3 scripts/learning_report.py local-data/mathua-snapshot.db \
  --config local-data/analytics.json --out local-data/reports/today.md
```

`analytics.json` is optional: `{"deadlines": {"<course id>": "2026-12-06"}}`
turns a plan's pace into on track or behind. The report changes nothing.

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
prerequisites. Mastering a topic gives partial credit to the topics below it,
as on the math side.

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
