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

**Scaffold steps.** Each card is put on one of the three steps of "The
order of practice inside a topic" by `scripts/anki_levels.py`, which the
import runs: step 1 for one fact, definition or step (a short front and
back, a cloze or fill-in of one item, a true/false check, a single-rule
computation), step 2 for worked cases (a `format::worked_example` card, a
computation of several steps or parts, a cloze of several items), step 3
for proofs, derivations, word problems and applications (`format::proof`,
"prove", "show that", "explain why", an economic setting worded as a
problem, a prompt in parts, a deck called Exercises, Problems or Proofs).
The import prints the steps per subject. Each card's step and the rule that
chose it are written to the module's `levels.json`; `level_overrides.json`
overrides them per card or per deck:

```json
{"cards": {"n1786236482047c0": 1}, "decks": {"Wooldridge - Introductory Econometrics 7e::Ch02 The Simple Regression Model": 2}}
```

After editing the overrides, `python3 scripts/anki_levels.py anki`
re-applies them to the bank without a re-import (`--check` reports only,
`--sample N` prints N cards with their step and rule, `--export X.colpkg`
re-reads the cards' decks and tags from the collection).

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

### Prerequisites across modules

The graph is one DAG over the corpus and every module, so a topic's
prerequisites may be anywhere in it: a Hoy heading rests on the corpus's
`calc.deriv.partial`, an econometrics code task on a pandas topic from
`python-dsh`, an Anki deck on the Hoy topic it practises. That is what lets
a learner struggling on a topic be sent back to the right concept whatever
book it came from, and what lets two courses that share a concept plan it
once.

Where a link is declared depends on how the topic was authored:

- **Authored banks** (`topic()` / `concept()` in `src/`): the third
  argument is the prerequisite list, written as concept ids. Add one with
  `python3 scripts/add_prereq.py <module> <topic> <prerequisite>...`, then
  rebuild the bank. The build refuses an id that is not in the graph, a
  topic that lists itself or an id twice, and a cycle, so a bad link never
  reaches a `concepts/` file. The Docker check runs the same test.
- **Anki modules**: each subject's `after` list in the subjects file.
  `python3 scripts/anki_prereqs.py <module>` re-applies the file to the
  module's `concepts/` without a re-import (`--check` only reports drift).
- **Base-corpus concepts** (`data/concepts/*.json`): upstream's; the fork
  links into them and does not edit them.

A link should name the specific concept at the right level (a chapter-7
Hoy topic depends on `calc.deriv.partial`, not on the calculus domain), and
practice sets (`Practice ·` decks) are never prerequisites of anything.

Checks, from quick to thorough:

- The server refuses to start on a dangling prerequisite, a cycle or a
  duplicate id (`internal/concepts`).
- `go run scripts/validate_graph.go` builds the install's graph and also
  rejects duplicate labels.
- `go run ./scripts/audit_graph` (`internal/graphaudit`) reviews the whole
  local graph: isolated topics, module topics whose prerequisites never
  leave their module, look-alike concepts across modules (tie them with
  `encompasses` or a link; do not delete), chains that wander back into a
  domain they left, key prerequisites, missing subdomains and labels, topics
  no course lists, and a shortlist of cross-module prerequisites by label
  similarity. `-review r.md` writes the shortlist to go through by hand;
  `-v` prints it. `TestPublicGraph` keeps the public graph clean in CI.

When a learner misses twice in a row on a topic, or misses it on a test,
the engine queues the topic's `key_prerequisites`, or, as no module topic
has any, its direct prerequisites from any module, unmastered and weakest
first, before serving the topic again at an easier variant.

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
A course's `courses.json` entry may carry `term`, `due` and `modules`
(`[{"title", "topics": [...]}]`, each a slice of its plan). A module with
`"after_due": true` is taught after the due date: it stays on the course
page, but pacing ("N topics/week to finish" and the queue's share for the
course) counts only the topics due by the date. `videos.json`
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

**The floor.** A learner who follows courses has a floor: the domains they
are past. Topics below it are never served to them: not as a review, not as
new work in Up next or the mixed session, not as remediation, not on the
mastery check and not in a placement test. A topic a followed course lists
as its own, or one the learner pins, is served as usual. A topic below the
floor still counts as met where a topic above it needs it, so a floor never
locks course material.

- The floor comes from the courses. A course above school grades (`grade`
  "MA", "college", "Pro"...) is past `arithmetic`, `fractions`, `prealgebra`
  and `geometry`; a school-grade course ("4th", "8th-9th") is past nothing.
  A course's `floor_domains` sets its floor (`[]` switches it off). With
  several courses, a domain is below the floor only when every one of them
  is past it.
- `MATHUA_REVIEW_FLOOR_DOMAINS` overrides the course floor with its own list;
  `none` switches the floor off. For a learner who follows no course it
  keeps its older meaning: their mastered topics in those domains are not
  reviewed, and nothing else changes.
- The server logs the floor at startup, and `GET /api/floor` (also the
  `floor` field of `GET /api/pins`) returns a learner's floor: its domains,
  its source (`courses`, `env` or `none`) and its scope.
- A course placement asks nothing more than two links below the course's
  own topics (`engine.CoursePlacementDepth`); a correct answer still credits
  everything below it. The onboarding placement of a learner who follows
  courses is a course placement too: it no longer starts the grade-school
  adaptive test over.

`TestNoFloorTopicsForCourseLearner` holds this on the whole base graph.

**How the queue is ordered.** Up next, the review session and the mixed
session (`/session`) follow the same rules:

1. **Reviews come most overdue first**, across every course and book: days
   past the review's due date (a decaying topic with no date: days past the
   decay horizon). They are not grouped by course, module or domain. A topic
   two courses share is one review. A review session serves a topic once; a
   review missed there is rescheduled, not asked again three questions on.
2. **Due reviews come before new topics.** Up next puts one review card
   ahead of new work. The one exception: a followed course due within seven
   days (`scheduler.UrgentDays`) puts its next plan topic first, and the
   review follows it.
3. **New topics come in the courses' paced plan order** (each course's plan
   spread over the days to its due date). A topic in an `after_due` module
   comes after the ones due by the date. Nothing off every followed course's
   path, no credit-only foundation and nothing below the floor is offered.
   The mixed session used to ignore all three and drop off-path topics from
   its view, which both offered off-path topics and locked topics above
   them; it now skips them and keeps the plan order.

Every recommendation carries a reason (`why` on `GET /api/next`, shown on
the Study tab) and every question from `/session` and the review session a
`reason`: "review due 3 days ago (8 due)", "next in ECON 5153 Math for
Economists · 7 Systems of Linear Equations", "needed for The bordered
Hessian in ...", "pinned", "missed in the mastery check", "Workday
integrations exam is due in 5 days · ...", "prerequisite of Implicit
differentiation after two misses".

**Replay audit.** `scripts/replay_audit` replays study sessions for one
learner against a copy of a database snapshot and logs every topic served
(Up next, the review session, a lesson, the mixed session, the mastery check
and its remediation, the course placements and the onboarding placement),
with the engine's reason, then classifies each serve: below the floor,
credit-only off the courses, off every course path, the same topic twice in
a review session or a list, unmet prerequisites, an after-due topic while a
due one is ready, remediation that is not a direct prerequisite, and a
placement question deeper than `CoursePlacementDepth`. It ages the copy one
day between sessions.

```bash
MATHUA_ROOT=$PWD go run ./scripts/replay_audit -db snapshot.db -student <id> -sessions 30 -json serves.jsonl
```

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

- In practice, a topic serves its questions in scaffold order (below), the
  book's exercises first within each step, up to the book step cap. An exercise you missed comes
  back about half the time, so the rest of the topic is not crowded out.
  The practice versions that unlock after an exercise join its step once
  it has a correct answer.
- Items filed under a topic through `cards.json` follow the same rules, so a
  book exercise can belong to a generated topic. The items can live in a
  deck that is not itself a topic: an authoring file whose `CHAPTER` sets
  `"graph": False` gets a bank but no `concepts/` node.
- Placement never serves a book exercise: it stays objective and quick, and
  a book page needs the PDF and reading time. A course review test may ask
  one. Neither counts a book exercise or an item that unlocks after another
  when deciding whether a topic has enough questions for a test.

### The order of practice inside a topic

A topic's questions climb three scaffold steps, as in Math Academy: simple
practice first, then worked cases, then word problems and applications.

| Step | What it is | Bank `level` | Generated questions |
|------|------------|--------------|---------------------|
| 1 | Apply the rule: one step, one idea ("differentiate x^5") | 1 | difficulty up to 0.45 |
| 2 | Worked cases: the book's examples and exercises, several steps | 2 (and unset) | 0.45 to 0.75 |
| 3 | Applications: word problems, economic settings, proofs | 3 | 0.75 and up |

Rules (`internal/generator/scaffold.go`):

1. The learner works the lowest step that is not done. A step is done when
   one of its questions (or a harder one) has a correct answer and the
   learner has tried the step's book exercises, up to the **book step cap**
   (default 3): a topic with nine book exercises at one step moves on after
   three have been tried and something at the step is right. No step-3
   question comes before a correct answer at step 1, unless the topic has
   only step 3.
2. Within a step the book comes first: its exercises (`book_ref`) in book
   order, then the bank's other items (untried ones first, at random, so
   variants interleave) and the topic's generated questions at that step's
   difficulty. Where the book has nothing for a step, generated questions
   and bank items carry it. On a generated topic with cards filed under it,
   about `MATHUA_CARD_SHARE` of a step's questions are cards.
3. An item that unlocks after another (`unlocks_after`) is not served until
   that one has a correct answer, and its level is never below that item's
   (the bank build and the server refuse one that is: a practice version
   practises the same skill as its exercise).
4. A lesson that asks for several questions at once gets them in order:
   when a step runs out, the lesson continues with the next step. The step
   comes from the learner's record, not from the Learn page's difficulty,
   which starts over at each visit: a learner who left off at step 3
   resumes there, and one who misses stays at the step (two misses in a row
   send them to the prerequisites, as before).
5. Once every step is done, practice samples across the steps by
   difficulty, and so does every review: a review is a check on the whole
   topic, not a climb.
6. Book exercises past the cap are not dropped. Once the cap's worth at a
   step have been tried they stop coming first and join the step's untried
   items; after the steps are done, and in reviews, a leftover exercise is
   served about half the time (the next untried one in book order, else a
   missed one), so the rest of the topic is not crowded out.

The cap is `MATHUA_BOOK_STEP_CAP` (default 3; `0` switches it off, so every
book exercise at a step comes before the next step, as before). A course in
a `courses.json` can set its own with `"book_step_cap": N` for the topics it
lists (plan, targets and modules); a topic on several courses takes the
smallest cap they set, and `0` turns the cap off for that course's topics.
Book proofs (`proofs/*.json` with a `book_ref`) are not scaffold items and
still come first until proved.

Placement and tests are unchanged: they take objective questions from any
step.

## Which grader a question gets

Every question is graded one way, chosen by what its prompt asks for. The
rules, which `scripts/audit_grading_style.py` checks each bank against:

| The prompt asks for | Kind | Grader |
| --- | --- | --- |
| A derivation, "show that", "prove", "verify that", "explain why", "justify", a critique, a summary | a proof or writing problem (`proofs/*.json`), or a recall card the judge grades (`written.go`) | the language-model judge against a marking scheme; self-graded flip card without a key |
| One closed-form value: a number, an expression, a derivative, a solution | `math` with `numeric` or `expression` parts | the numeric grader (a fraction, a decimal to the key's places, a rounded decimal for a fraction key) or SymPy equivalence |
| A set of values, a domain, an interval, a union of intervals | `math` with `set` or `interval` parts | the set and interval graders: order, repeats, spacing and the spellings of ∞, ∪ and ∅ do not matter |
| A name, a definition's term, what something is called | `term` | the term grader: case, accents, a leading article and punctuation do not matter |
| Yes or no, true or false | `mc` with the two options (served Yes/No, Sí/No, True/False) | exact choice |
| "Which of these", one option among several | `mc` | exact choice, as clicked or as a typed letter |
| Code that does something | `code` with hidden tests | the Python runner |
| What a snippet prints | `output` | printed text, spacing aside |
| Something to recall and mark yourself | `recall` | the learner's own verdict |

A typed-math key must grade correct through the same path a learner's
answer takes, and so must its plausible spellings (1/2 and 0.5, 3x^2 and
3x², e^x and exp(x), (2, oo) and (2, ∞)), while a nearby wrong answer must
not. `go test ./internal/gradecheck` grades every item of the committed
modules and a sample of every generator that way; `make gradecheck` does
the same over `modules-local/` (`GRADECHECK_FLAGS=-code` runs every code
task's solution and starter too, slowly), and
`python3 scripts/audit_grading_style.py` lists the items whose prompt and
kind disagree, by rule. It skips what `internal/pytrack/written.go` also
skips: a method question ("how does one prove"), a purpose clause ("to
verify that A is idempotent, it is enough to..."), a subject that shows or
describes, and "compute it, or show it is impossible". The clear cases are
fixed in the authoring sources; the rest are listed for the author.

For imported cards the source is the import. A card whose front asks for an
argument keeps its flip card: `scripts/import_anki.py` does not turn it into
a true/false choice or a typed value (the judge grades it, or you mark it),
and `python3 scripts/anki_grading.py anki` applies the same rule to a bank
imported before. The author's per-card decisions go in the module's
`grading_overrides.json`, which both apply: `replace` rewrites a prompt
("Derive the equation" to "Find the equation" where the equation is the
whole task), and `kind` makes a card a flip card, a choice with its options
or a typed value with its parts, each with a `why`.

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
