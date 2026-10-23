# MA Economics fork

This fork adds two courses and one destination on top of upstream Mathua. It
adds no new concepts or generators: everything uses the existing graph.

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

Courses live in `data/courses.json` as a list of target concept IDs, and are read
at startup. Restart the server after editing. Concept IDs are in
`data/concepts/*.json`. Run `go run scripts/validate_graph.go` after any change.

## Python for Data Science track

The header has a **Math | Python** switch. Same site, same login, same XP and
review schedule. The switch only changes which topics the Learn shelf, the
topic browser, reviews and the quiz offer.

The Python track follows the five chapters of Jake VanderPlas's *Python Data
Science Handbook*. Every topic links to its book section. The questions and
notes are original: the book's text is CC-BY-NC-ND, so nothing here adapts it.

| Chapter | Domain | Status |
|---|---|---|
| 1. IPython | `python_ipython` | 9 topics, 60 questions |
| 2. NumPy | `python_numpy` | 20 topics, 127 questions |
| 3. Pandas | `python_pandas` | planned |
| 4. Matplotlib | `python_matplotlib` | planned |
| 5. Machine Learning | `python_ml` | planned |

Each topic mixes three kinds of question:

- **Multiple choice** on the concept.
- **What does this print?** Type the output. Spacing inside arrays does not
  matter; everything else does.
- **Code tasks.** Write code in the box. **Run** shows output and plots.
  **Check** grades it against hidden tests on the server.

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

Questions are written in `data/python/<NN>_<chapter>.py`, one file per
chapter. Build them with the pinned libraries installed:

```bash
python3 scripts/python_bank.py build           # all chapters
python3 scripts/python_bank.py build numpy     # one chapter
```

The build runs every snippet. It computes the expected output of each "what
does this print" question, checks that each reference solution passes its
hidden tests and that each starter fails them, and writes:

- `internal/pytrack/bank/*.json`: the question bank, embedded in the server.
- `data/concepts/python_*.json`: the topics and their prerequisites.

Then rebuild the image. The image build runs
`python3 scripts/python_bank.py check` and fails if a stored output no longer
matches what the pinned libraries print.
