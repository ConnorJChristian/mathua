# MA Economics fork

A fork of [Mathua](https://github.com/chuma-beep/mathua), an open-source
adaptive learning engine (Go backend, Next.js frontend, a concept graph with
prerequisites, spaced review, placement diagnostics), reshaped into a
course-centred study app for graduate economics. The engine and
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
  (mathematics for economists and a statistical-learning companion, built
  on the upstream graph) and `modules/python-dsh` (the Python Data Science Handbook as
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
| `econ5153` | Mathematics for economists | 42 | 143 |
| `statlearn` | Statistical learning (ISLR companion) | 25 | 129 |

Neither course pulls in any trigonometry. The only geometry prerequisite is
plotting a point. The placement diagnostic credits prerequisites when you
answer harder questions correctly, so arithmetic and algebra basics are
skipped unless you miss something.

## Licence

MIT, the same as upstream (see `LICENSE`). Use it, copy it, change it,
redistribute it, commercially or not, with attribution. The fork's own code
and the public modules are published under the same terms. Material derived
from copyrighted textbooks is deliberately kept out of the repository.

## Quality disclaimer

The fork's changes were written with an LLM coding assistant (Claude) under
the owner's direction, and tested only as far as the owner's own use needed.
It is published for sharing, because others may find it helpful, not as a
product: there is no guarantee, and no intention, of production-quality code
or functionality, no promise that it is a finished or final piece of work,
and no support or maintenance commitment. Use at your own risk; the MIT
licence's warranty disclaimer applies in full.

## What you get, and what you bring

The repository is the engine plus empty slots. Out of the box it has upstream
Mathua's mathematics corpus and the two public modules above. Your own
material goes in `modules-local/` (git-ignored): PDFs of your books in a
module's `files/` with `readings.json` and `books.json` pointing into them,
courses in `courses.json`, question banks written with the authoring API or
imported from an Anki collection, lecture notes and videos. Nothing in that
folder is ever committed, so the repository stays shareable while your
install carries whatever you have the right to read.

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

**Mathematics for economists:** derivatives and basic integration (substitution, FTC) incl. exp/log and implicit differentiation, convexity
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
`proofs/`, `readings.json`, `courses.json`, `destinations.json` and `files/`. The
server and the web build read every enabled module. A module can declare a
**track**, a named group of domains (the fork has no Math | Python switch
any more: one queue spans every track; a track still scopes a request that
names it by id).
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
It comes due after 50 XP, 4 topics worked and a gap since the last one: 20
hours by default, or `MATHUA_QUIZ_GAP_HOURS` (168 for a weekly test).

### Book exercises and practice that unlocks

Two optional fields of a bank item (`bank/*.json`, written with
`scripts/python_bank.py`) serve a textbook's exercises from your own copy of
the book without putting the book's text in the bank.

`book_ref` makes a page of a PDF in a module's `files/` the item's prompt:

```json
"book_ref": {"file": "hoy-3e.pdf", "module": "ma-program", "page": 187,
             "printed_page": 173, "label": "Hoy Exercise 5.4, #3"}
```

`page` is the PDF page the exercise starts on; `end_page` (when it runs
over a page break) and `printed_page` (the number printed in the book) are
optional. The item keeps its own answer kind (typed math, a term, multiple
choice or a recall card) and its `prompt` says only what to enter. The
served question names the exercise and its pages, and Learn, review and the
tests show the page in the embedded PDF reader directly above the answer
box. When the PDF is not in `files/`, the label and both page numbers are
shown instead, so you can work from a paper copy. A proof problem
(`proofs/*.json`) takes the same field; its `statement` then only says what
to write, and its reference proof states the claim in the module's own
words, which is what the judge grades against.

`unlocks_after` names another item of the same concept (for a proof, another
problem of the same topic). The item is not served to you until you have
answered that one correctly at least once. The attempts table is the
record: a bank item's answer key names the item (`math:<concept>#<item>`,
`proof:<id>`), and for multiple choice the question text is compared too.

Serving rules:

- In practice, a topic serves its book exercises first, in bank order, until
  each has a correct answer. An exercise you missed comes back about half
  the time, so the rest of the topic is not crowded out. After that, the
  practice versions that unlock after them mix in with everything else.
- Items filed under a topic through `cards.json` follow the same rules, so a
  book exercise can belong to a generated topic. The items can live in a
  deck that is not itself a topic: an authoring file whose `CHAPTER` sets
  `"graph": False` gets a bank but no `concepts/` node.
- Placement never serves a book exercise: it stays objective and quick, and
  a book page needs the PDF and reading time. A course review test may ask
  one. Neither counts a book exercise or an item that unlocks after another
  when deciding whether a topic has enough questions for a test.

## Proof grading

A module may hold written-proof problems in `proofs/*.json`. Each problem
names the topic whose practice serves it (`concept`), and carries its
statement, a reference proof and a marking scheme on a 0-7 scale: milestones
with points, caps for fatal errors ("assumes the conclusion: at most 1") and
accepted alternative routes. The marking scheme is written once, offline.
`internal/proof` describes the format. Proofs follow a textbook, so they
belong in a `modules-local/` module; the committed modules hold none.

A proof is graded by an Anthropic model (the Messages API) run several times
at once; the grade is the median, and a proof counts as correct only when
every run gives it at least 6 of 7. The rubric block is sent with prompt
caching, an identical resubmitted proof is answered from a cache, and the
month's spend is kept in the database against a ceiling. A missing key, a
failed call or a spent budget leaves the proof ungraded, never wrong, and no
proof is served while it could not be graded. Proofs are not served in
placement or in quizzes.

| Variable | Default | Meaning |
| --- | --- | --- |
| `ANTHROPIC_API_KEY` | unset | Turns proof grading on. |
| `MATHUA_PROOF_MODEL` | `claude-sonnet-5-5` | The judge model (`claude-opus-5-5` is stricter and dearer). |
| `MATHUA_PROOF_RUNS` | 5 | Judge runs per proof (1-9). |
| `MATHUA_PROOF_MONTHLY_USD` | 60 | Monthly spend ceiling in US dollars. |
| `MATHUA_PROOF_PRICE_IN`, `MATHUA_PROOF_PRICE_OUT` | 3 / 15 (Sonnet), 5 / 25 (Opus) | Dollars per million input and output tokens; the defaults err high. |
| `MATHUA_PROOF_MAX_TOKENS` | 2000 | Output budget of one run. |
| `MATHUA_PROOF_TIMEOUT_SECONDS` | 120 | Bound on one grading. |
| `MATHUA_PROOF_SHARE` | 0.2 | Share of a topic's practice that is proofs. |
| `MATHUA_PROOF_COUNTS_FOR_MASTERY` | off | Let a fully correct proof count as mastery evidence. |

Until you turn on `MATHUA_PROOF_COUNTS_FOR_MASTERY`, a proof result is
feedback only, like a self-graded card. Turn it on after calibration:
`mathua proof calibrate FILE` grades a CSV of `problem_id,proof_text,human_score`
and reports the mean absolute error, the share within one point, the bias and
the false-pass rate; `mathua proof calibrate -synthetic` grades the synthetic
proofs stored with each problem. `mathua proof spend` shows the month's spend
and `mathua proof list` the problems. In Docker, run them with
`docker compose exec mathua /mathua proof ...`.

### Writing problems

A problem in `proofs/*.json` may set `"kind": "writing"` (the default,
`"proof"`, may be left out). It is then a piece of academic writing rather
than a proof: a definition, a general-specific paragraph, a data commentary,
a summary, a critique, a research-paper introduction. The file format is the
same, read differently:

| Field | For a writing problem |
| --- | --- |
| `statement` | The task, with any source text or data it works from. |
| `given` | Optional notes shown under the task (length, audience). |
| `reference_proof` | A model answer written to the standard asked for. |
| `milestones` | The rubric, 7 points, from the section's own criteria (moves, register, accuracy to the source, flow). |
| `caps` | Fatal problems, such as off-task, copying the source, the wrong genre. |
| `alternatives` | Other acceptable approaches. |
| `samples` | Synthetic texts with intended scores, by convention `strong`, `weak` and `off-task`. |

The judge's instructions switch from checking a proof to grading academic
writing against the rubric (genre and moves, register, accuracy to the
source or data, clarity and flow). It is as strict and returns the same
verdict, so everything else is shared: k runs and the median, the grade
cache, the spend guard, ungraded rather than wrong on any failure,
calibration (`mathua proof calibrate -synthetic` includes writing samples)
and `MATHUA_PROOF_COUNTS_FOR_MASTERY`, under which a writing problem counts
exactly as a proof does. The served question begins **Write.** and carries
`<!--writing-answer-->` after the proof marker; the app labels the box "Your
writing", counts words, and shows the model answer after grading. `mathua
proof list` shows each problem's kind.

### Written-answer flashcards

About a quarter of imported flashcards ask for a proof, a derivation or a
justification ("prove", "show that", "derive", "explain why", "justify",
"verify"). Marking those yourself after reading a one-line back is easy to
get wrong in your own favour, so when proof grading has a key, such a card
is served with the proof box instead of the reveal button and graded by the
same judge: k runs, the median, and fully right only when every run gives at
least 6 of 7. After grading you see the score out of 7, the milestone
checklist, the first error, the feedback and the card's back. "Just show me
the answer (self-grade)" under the box reveals the back and turns the card
back into an ordinary flashcard.

Which cards: `internal/pytrack/written.go` classifies each recall card from
its front. Cards whose task is done on a graph, sketch, table or diagram,
cloze cards, computer exercises, method questions ("how do you prove") and
checks of a computed answer ("verify your answer", "compute it or show it
does not exist") stay self-graded, as does any card with a picture on the
front. A card's `"grading": "llm"` or `"grading": "self"` in the bank file
overrides the classifier. `mathua proof classify [-bank B] [-sample N]`
reports the counts per bank and a sample with the reason for each verdict.

Marking schemes are written lazily. The first time an answer to a card is
graded, the judge's model writes a scheme for it from the front and back
(milestones worth 7 points, caps for fatal errors, alternative routes, and a
full reference answer when the back is terse). The scheme is stored in the
database (the `proof_grades` table, under the card's id and a hash of its
front and back, so editing a card gets it a new scheme) and reused for every
later answer. Writing schemes counts against the same monthly budget. If a
scheme cannot be written, the card stays self-graded: for a day after a
failed call (`MATHUA_LLM_RECALL_RETRY_HOURS`), and until the card is edited
when the model judged it ungradable from text. With no key, with
`MATHUA_LLM_RECALL=0` or with the budget spent, every card is an ordinary
flashcard.

A graded card is recorded exactly as a self-graded card is, with the judge's
verdict as your mark: practice, never mastery evidence, never a miss toward
a halt, whatever `MATHUA_PROOF_COUNTS_FOR_MASTERY` says. An answer that could
not be graded is never a miss: nothing is recorded and you can submit again
or self-grade. A quiz serves these cards as plain flashcards; placement and
course tests serve no flashcards at all.

| Variable | Default | Meaning |
| --- | --- | --- |
| `MATHUA_LLM_RECALL` | on when `ANTHROPIC_API_KEY` is set | `0` serves every flashcard self-graded. |
| `MATHUA_LLM_RECALL_BANKS` | unset (all banks) | Comma-separated bank files whose cards are graded, e.g. `calculus,econometrics`. The `anki_` prefix and `.json` are optional. |
| `MATHUA_LLM_RECALL_EXCLUDE_BANKS` | unset | Bank files whose cards stay self-graded. `linear_algebra` leaves out a linear algebra deck whose proofs are above your level. |
| `MATHUA_LLM_RECALL_SCHEME_MAX_TOKENS` | 3000 | Output budget of one marking scheme. |
| `MATHUA_LLM_RECALL_RETRY_HOURS` | 24 | How long a card whose scheme failed stays self-graded before another try. |

`mathua proof schemes [-bank B] [-concept C] [-limit N] [-dry-run] [-yes]`
writes schemes ahead of time, so a card's first answer does not wait for
its scheme. It shows the number of schemes and the estimated cost and asks
before spending (`-yes` skips the question). `-bank` overrides the
exclusions. With Sonnet at list prices and 5 runs, a scheme costs about 2
cents (at most 5) and grading an answer about 7 cents (at most 18); 1,200
schemes come to about $25, at most $58.

`mathua proof calibrate -recall [-n N] [-bank B]` samples N written-answer
cards and grades each card's own back, which should pass, and a wrong
answer the model writes against the card's scheme, which should fail. It
prints the confusion matrix (false-fail and false-pass rates) and the cost.

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

The `python-dsh` module adds the **Python** track: same site, same login,
same XP and review schedule; its topics sit in the same Study queue, topic
browser, reviews and mastery check as everything else.

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

A bank can also hold **term items**: you type the name of a concept as the
source uses it. An item has kind `term`, the term in `correct` and other
spellings of it in `accept`. Case, accents, a leading article and surrounding
punctuation do not matter. A two-option choice can be True/False, Yes/No,
Sí/No or Verdadero/Falso.

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
MATHUA_ROOT=../mathua python3 scripts/python_bank.py build ma-program
                                                 # another install's modules-local/
```

The build runs every snippet. It computes the expected output of each "what
does this print" question, checks that each reference solution passes its
hidden tests and that each starter fails them, and writes the module's
`bank/*.json` and `concepts/*.json`.

Then rebuild the image. The image build runs
`python3 scripts/python_bank.py check` and fails if a stored output no longer
matches what the pinned libraries print.
