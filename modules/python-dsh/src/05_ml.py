# Chapter 5 of the Python Data Science Handbook: Machine Learning.
#
# Every question and note here is original. Each concept links to the book
# section it follows; the book's own text is CC-BY-NC-ND and is never adapted.
# Build with: python3 scripts/python_bank.py build ml
#
# Section 5.14 (image features) is not covered: its method (HOG) lives in
# scikit-image, which the grading image does not install. Feature extraction
# and pipelines are practised in py.ml.categorical_text and py.ml.pipelines.

CHAPTER = {
    "chapter": "ml",
    "title": "Machine Learning",
    "domain": "python_ml",
    "concepts": [

# ---------------------------------------------------------------- 5.01
concept(
    "py.ml.kinds", "Kinds of machine learning", ["py.np.attributes"],
    "5.01", "What Is Machine Learning?", "05.01-what-is-machine-learning",
    """
    Machine learning fits a model with adjustable parameters to data, then
    uses the fitted model on data it has not seen. The tasks split two ways.

    **Supervised** learning has a label for every row. If the label is a
    category (defaulted or not, sector of a firm) it is **classification**; if
    it is a number (a wage, next quarter's sales) it is **regression**.

    **Unsupervised** learning has no labels. **Clustering** groups similar
    rows (counties with similar economies); **dimensionality reduction**
    compresses many columns into a few that keep most of the structure.

    ```python
    import numpy as np
    income = np.array([28.0, 41.5, 63.0, 19.2])   # regression target
    poor = (income < 30).astype(int)              # classification target
    ```

    An economist already knows half of this: OLS is supervised regression.
    The new emphasis is on prediction out of sample rather than on
    interpreting coefficients.
    """,
    [
        mc("m1", "A bank wants to predict whether each loan applicant will default (yes or no) from past applicants whose outcomes are known. What kind of task is this?",
           "Supervised classification", ["Supervised regression", "Clustering", "Dimensionality reduction"],
           """
           Past outcomes are known, so it is supervised; the outcome is a
           category, so it is classification.
           """, level=1),
        mc("m2", "You predict each household's monthly electricity spending in currency units from its size and region. What kind of task is this?",
           "Supervised regression", ["Supervised classification", "Clustering", "Unsupervised regression"],
           "The target is a continuous number, which makes it regression.", level=1),
        mc("m3", "You have 40 economic indicators for 3,000 counties and no labels, and you want to find groups of counties with similar profiles. What kind of task is this?",
           "Clustering (unsupervised)", ["Classification (supervised)", "Regression (supervised)", "Dimensionality reduction (supervised)"],
           """
           With no labels to predict, the goal is to discover structure. Finding
           groups of similar rows is clustering.
           """, level=1),
        mc("m4", "What do dimensionality-reduction methods do?",
           "Represent each row with fewer columns while keeping as much of the data's structure as possible",
           ["Remove rows that look like outliers", "Predict a label with fewer parameters", "Drop every column that has missing values"],
           """
           They map many features to a few new ones, which helps with plotting,
           compression and noise removal. No label is used.
           """, level=2),
        out("o1", "What does this print?",
            """
            import numpy as np
            rent = np.array([900.0, 1450.0, 1200.0, 700.0, 1600.0])
            pricey = (rent > 1000).astype(int)
            print(pricey, pricey.sum())
            """,
            """
            Thresholding a continuous variable turns a regression target into a
            two-class classification target: 1 where rent exceeds 1000.
            """, level=1),
        task("c1",
             """
             `income` (already defined) is an array of household incomes in
             thousands. Make `low`, an integer array that is 1 where income is
             below the median of `income` and 0 otherwise. This turns a regression
             target into a classification target.
             """,
             """
             import numpy as np
             low = (income < np.median(income)).astype(int)
             """,
             [
                 check("labels", """
                     import numpy as np
                     check(getattr(low, "shape", None) == income.shape, "`low` should have one entry per household.")
                     check(same_array(low, [1, 0, 0, 1, 1, 0]), "`low` should be 1 exactly where income is below the median.")
                 """),
                 check("dtype", 'check(low.dtype.kind == "i", "`low` should be an integer array of 0s and 1s, not booleans.")'),
             ],
             """
             Compare against `np.median(income)` to get a boolean mask, then
             `.astype(int)` converts True and False to 1 and 0.
             """,
             starter="""
             import numpy as np
             low = ...
             """,
             setup="import numpy as np\nincome = np.array([22.0, 58.5, 41.0, 30.2, 19.9, 75.0])", level=1),
    ]),

# ---------------------------------------------------------------- 5.02
concept(
    "py.ml.estimator_api", "Features matrix, target and the Estimator API",
    ["py.ml.kinds", "py.np.reshape", "py.pd.dataframe"],
    "5.02", "Introducing Scikit-Learn", "05.02-introducing-scikit-learn",
    """
    Scikit-Learn wants data in two pieces. The **features matrix** `X` is
    two-dimensional with shape `(n_samples, n_features)`: one row per
    observation, one column per variable. The **target** `y` is
    one-dimensional, one value per row. A DataFrame works as `X`.

    Every model follows the same steps:

    ```python
    from sklearn.linear_model import LinearRegression
    model = LinearRegression(fit_intercept=True)  # 1. class, 2. hyperparameters
    X = schooling[:, None]                         # 3. arrange X as 2-D
    model.fit(X, wage)                             # 4. fit
    model.coef_, model.intercept_                  # learned values end in _
    model.predict([[12], [16]])                    # 5. predict new rows
    ```

    Hyperparameters are chosen by you when you build the object; parameters
    are learned by `fit` and stored in attributes with a trailing underscore.
    Unsupervised models use `fit(X)` with no `y`, then `transform` or
    `predict`.
    """,
    [
        mc("m1", "`x` is a 1-D array of 50 years of schooling. Why does `LinearRegression().fit(x, wage)` raise an error?",
           "`X` must be 2-D with shape (n_samples, n_features); `x[:, None]` or `x.reshape(-1, 1)` fixes it",
           ["LinearRegression only accepts DataFrames", "The target must be 2-D as well", "Integer features are not allowed"],
           """
           Scikit-Learn refuses to guess whether a 1-D array is one feature or
           one sample. Making it a single column removes the ambiguity.
           """, level=1),
        mc("m2", "Which of these is a hyperparameter rather than a learned parameter?",
           "`n_clusters` passed to `KMeans(...)`", ["`coef_` of a fitted LinearRegression", "`intercept_`", "`cluster_centers_` of a fitted KMeans"],
           """
           Hyperparameters are set before fitting, in the constructor. Names
           ending in an underscore are only created by `fit`.
           """, level=2),
        mc("m3", "In what order does the Estimator API workflow go?",
           "Choose a model class, set hyperparameters, arrange X and y, fit, then predict or transform",
           ["Arrange X and y, predict, then fit", "Fit, choose hyperparameters, then choose a class",
            "Choose hyperparameters, predict, then arrange X and y"],
           "You build the object first, fit it on data, then apply it.", level=1),
        out("o1", "What does this print?",
            """
            import numpy as np
            from sklearn.linear_model import LinearRegression
            x = np.array([8, 10, 12, 14, 16])
            wage = 3 + 1.5 * x
            model = LinearRegression().fit(x[:, None], wage)
            print(round(model.coef_[0], 3), round(model.intercept_, 3))
            """,
            """
            The data lie exactly on wage = 3 + 1.5 x, so least squares recovers
            slope 1.5 and intercept 3.
            """, level=1),
        out("o2", "What does this print?",
            """
            from sklearn.datasets import load_wine
            from sklearn.decomposition import PCA
            X, y = load_wine(return_X_y=True)
            Z = PCA(n_components=2).fit_transform(X)
            print(X.shape, y.shape, Z.shape)
            """,
            """
            The wine data has 178 rows and 13 features. PCA is unsupervised: it
            ignores `y` and maps every row to 2 new columns.
            """, level=2),
        task("c1",
             """
             `exper` (years of experience, 1-D) and `wage` are already defined.
             Fit a `LinearRegression` called `model` of wage on experience, then
             store in `pred` its predicted wages at 5, 10 and 20 years.
             """,
             """
             import numpy as np
             from sklearn.linear_model import LinearRegression
             model = LinearRegression()
             model.fit(exper[:, None], wage)
             pred = model.predict(np.array([[5], [10], [20]]))
             """,
             [
                 check("fitted", 'check(hasattr(model, "coef_"), "`model` should be a fitted LinearRegression.")'),
                 check("pred", """
                     import numpy as np
                     ref = 11.8 + 0.42 * np.array([5, 10, 20])
                     check(getattr(pred, "shape", None) == (3,), "`pred` should hold three predictions.")
                     check(close(pred, ref, rtol=1e-3), "`pred` should be the model's predictions at 5, 10 and 20 years.")
                 """),
             ],
             """
             `exper[:, None]` turns the 1-D array into one column. New inputs
             must have the same layout: three rows, one column.
             """,
             starter="""
             import numpy as np
             from sklearn.linear_model import LinearRegression
             model = ...
             pred = ...
             """,
             setup="import numpy as np\nexper = np.arange(0, 31, dtype=float)\nwage = 11.8 + 0.42 * exper", level=1),
        task("c2",
             """
             The DataFrame `df` (already defined) has columns `hours`, `age` and
             `earnings`. Set `X` to a DataFrame of the two feature columns,
             `y` to the `earnings` column, and fit `model`, a
             `LinearRegression`, on them.
             """,
             """
             from sklearn.linear_model import LinearRegression
             X = df[["hours", "age"]]
             y = df["earnings"]
             model = LinearRegression().fit(X, y)
             """,
             [
                 check("shapes", """
                     check(getattr(X, "shape", None) == (40, 2), "`X` should have 40 rows and the two feature columns.")
                     check(getattr(y, "ndim", None) == 1 and len(y) == 40, "`y` should be the 1-D earnings column.")
                 """),
                 check("coef", 'check(hasattr(model, "coef_") and close(model.coef_, [20.0, 3.0], rtol=1e-3), "`model` should be fitted on hours and age, in that order.")'),
             ],
             """
             Double brackets select a DataFrame of columns (2-D); single brackets
             select one Series (1-D), which is what a target should be.
             """,
             starter="""
             from sklearn.linear_model import LinearRegression
             X = ...
             y = ...
             model = ...
             """,
             setup="""
             import numpy as np, pandas as pd
             rng = np.random.default_rng(3)
             hours = rng.integers(10, 50, 40).astype(float)
             age = rng.integers(20, 60, 40).astype(float)
             df = pd.DataFrame({"hours": hours, "age": age, "earnings": 100 + 20 * hours + 3 * age})
             """, level=2),
    ]),

# ---------------------------------------------------------------- 5.03
concept(
    "py.ml.holdout_cv", "Holdout sets and cross-validation", ["py.ml.estimator_api"],
    "5.03", "Hyperparameters and Model Validation", "05.03-hyperparameters-and-model-validation",
    """
    Scoring a model on the rows it was fitted on measures memory, not
    prediction. A one-nearest-neighbour classifier scores 100% on its own
    training data because each row's nearest neighbour is itself. The fix is
    the out-of-sample test an econometrician runs on a forecast: hold data
    back.

    ```python
    from sklearn.model_selection import train_test_split, cross_val_score
    X_tr, X_te, y_tr, y_te = train_test_split(X, y, test_size=0.25, random_state=0)
    model.fit(X_tr, y_tr)
    model.score(X_te, y_te)          # accuracy (classifiers) or R^2 (regressors)
    cross_val_score(model, X, y, cv=5)   # five scores, one per fold
    ```

    A single split wastes data and depends on luck. **k-fold
    cross-validation** splits the rows into k folds, fits k times, and tests
    on each fold once, so every row is used for testing exactly once.
    """,
    [
        mc("m1", "Why is a model's score on its own training data a poor guide to how it will do on new data?",
           "The model has already seen those rows, so the score rewards memorising noise as well as learning the pattern",
           ["Training scores are always exactly 1.0", "`score` cannot be called on training data", "Training data are always smaller than test data"],
           """
           Flexible models can fit the noise in the training rows. Only rows held
           out of the fit reveal whether that fit generalises.
           """, level=1),
        mc("m2", "In 5-fold cross-validation on 200 rows, how often is each row used as a test row?",
           "Exactly once", ["Five times", "Never; test rows are separate", "About 40 times"],
           """
           The 200 rows are split into 5 folds of 40. Each fit leaves out one
           fold, so each row is in the test fold of exactly one fit.
           """, level=2),
        mc("m3", "What does `model.score(X_te, y_te)` return for a `LinearRegression`?",
           "The R-squared on the test rows", ["The mean squared error", "The share of exactly correct predictions", "The adjusted R-squared"],
           """
           Regressors score with R-squared and classifiers with accuracy by
           default. Out of sample, R-squared can even be negative.
           """, level=2),
        out("o1", "What does this print?",
            """
            from sklearn.datasets import load_breast_cancer
            from sklearn.neighbors import KNeighborsClassifier
            X, y = load_breast_cancer(return_X_y=True)
            knn = KNeighborsClassifier(n_neighbors=1).fit(X, y)
            print(knn.score(X, y))
            """,
            """
            With one neighbour, every training row is classified by itself, so
            in-sample accuracy is perfect. It says nothing about new patients.
            """, level=1),
        out("o2", "What does this print?",
            """
            import numpy as np
            from sklearn.model_selection import train_test_split
            X = np.zeros((120, 4))
            y = np.zeros(120)
            X_tr, X_te, y_tr, y_te = train_test_split(X, y, test_size=0.25, random_state=0)
            print(X_tr.shape, X_te.shape, y_te.shape)
            """,
            "A quarter of 120 rows (30) goes to the test set and 90 stay for training.", level=1),
        out("o3", "What does this print?",
            """
            from sklearn.datasets import load_iris
            from sklearn.naive_bayes import GaussianNB
            from sklearn.model_selection import cross_val_score
            X, y = load_iris(return_X_y=True)
            scores = cross_val_score(GaussianNB(), X, y, cv=4)
            print(len(scores), round(scores.mean(), 2))
            """,
            """
            `cv=4` gives one accuracy per fold, four in all; the mean is the
            usual summary.
            """, level=2),
        task("c1",
             """
             Using `load_diabetes(return_X_y=True)`, split the data with
             `train_test_split(..., test_size=0.3, random_state=0)`, fit a
             `LinearRegression` on the training part, and store its test-set
             R-squared in `test_r2` and its training R-squared in `train_r2`.
             """,
             """
             from sklearn.datasets import load_diabetes
             from sklearn.linear_model import LinearRegression
             from sklearn.model_selection import train_test_split
             X, y = load_diabetes(return_X_y=True)
             X_tr, X_te, y_tr, y_te = train_test_split(X, y, test_size=0.3, random_state=0)
             model = LinearRegression().fit(X_tr, y_tr)
             train_r2 = model.score(X_tr, y_tr)
             test_r2 = model.score(X_te, y_te)
             """,
             [
                 check("scores", """
                     from sklearn.datasets import load_diabetes
                     from sklearn.linear_model import LinearRegression
                     from sklearn.model_selection import train_test_split
                     X_, y_ = load_diabetes(return_X_y=True)
                     a, b, c, d = train_test_split(X_, y_, test_size=0.3, random_state=0)
                     m = LinearRegression().fit(a, c)
                     check(close(train_r2, m.score(a, c)), "`train_r2` should be R-squared on the training rows.")
                     check(close(test_r2, m.score(b, d)), "`test_r2` should be R-squared on the 30% test rows, with random_state=0.")
                 """),
             ],
             """
             Fit only on the training rows. `score` on the test rows is the honest
             number; it is lower than the training R-squared, as expected.
             """,
             starter="""
             from sklearn.datasets import load_diabetes
             from sklearn.linear_model import LinearRegression
             from sklearn.model_selection import train_test_split
             X, y = load_diabetes(return_X_y=True)
             train_r2 = ...
             test_r2 = ...
             """, level=2),
        task("c2",
             """
             `X` and `y` (a small classification data set) are already defined.
             Use `cross_val_score` with 5 folds to evaluate
             `KNeighborsClassifier(n_neighbors=5)`. Store the array of fold
             scores in `scores` and their mean in `cv_mean`.
             """,
             """
             from sklearn.neighbors import KNeighborsClassifier
             from sklearn.model_selection import cross_val_score
             scores = cross_val_score(KNeighborsClassifier(n_neighbors=5), X, y, cv=5)
             cv_mean = scores.mean()
             """,
             [
                 check("scores", 'check(len(scores) == 5, "`scores` should hold one score per fold (5).")'),
                 check("mean", """
                     from sklearn.neighbors import KNeighborsClassifier
                     from sklearn.model_selection import cross_val_score
                     ref = cross_val_score(KNeighborsClassifier(n_neighbors=5), X, y, cv=5).mean()
                     check(close(cv_mean, ref), "`cv_mean` should be the mean of the five fold accuracies.")
                 """),
             ],
             "`cross_val_score` does the splitting, fitting and scoring for you.",
             starter="""
             from sklearn.neighbors import KNeighborsClassifier
             from sklearn.model_selection import cross_val_score
             scores = ...
             cv_mean = ...
             """,
             setup="from sklearn.datasets import make_classification\nX, y = make_classification(n_samples=150, n_features=5, random_state=1)", level=2),
    ]),

concept(
    "py.ml.model_selection", "Bias-variance, validation curves and grid search", ["py.ml.holdout_cv"],
    "5.03", "Hyperparameters and Model Validation", "05.03-hyperparameters-and-model-validation",
    """
    A model that is too rigid **underfits** (high bias): it scores badly on
    training and test data alike. One that is too flexible **overfits** (high
    variance): it scores well on training data and badly on new data. Think
    of predicting GDP with a straight line versus a degree-15 polynomial.

    A **validation curve** plots training and validation scores against a
    complexity hyperparameter; the best setting is where the validation
    score peaks. A **learning curve** plots both against training-set size;
    once the curves meet, more data will not help that model.

    `GridSearchCV` automates the search with cross-validation:

    ```python
    from sklearn.model_selection import GridSearchCV
    pipe = make_pipeline(PolynomialFeatures(), LinearRegression())
    grid = GridSearchCV(pipe, {"polynomialfeatures__degree": [1, 2, 3, 4]}, cv=5)
    grid.fit(X, y)
    grid.best_params_, grid.best_estimator_
    ```

    Parameters inside a pipeline are named `step__param`.
    """,
    [
        mc("m1", "A model scores R-squared 0.31 on training data and 0.29 on validation data, and the target is known to be very predictable. What is the likely problem?",
           "High bias: the model is too simple (underfitting)", ["High variance: the model is overfitting", "Data leakage from the test set", "Nothing; close scores mean the model is ideal"],
           """
           Training and validation scores are close but both low. A more
           flexible model is needed; overfitting would show a large gap.
           """, level=2),
        mc("m2", "On a validation curve, model complexity rises from left to right. What shape does the validation score usually have?",
           "It rises, peaks, then falls as the model starts to overfit",
           ["It rises steadily, like the training score", "It falls steadily", "It is flat"],
           """
           The training score keeps rising with complexity; the validation score
           peaks at the right balance of bias and variance.
           """, level=2),
        mc("m3", "A learning curve shows the training and validation scores have converged at a level you find too low. What will help?",
           "A more flexible model; more data of the same kind will not raise the score",
           ["Collecting many more rows", "Using fewer cross-validation folds", "Shuffling the rows again"],
           """
           Once the curves meet, the model has learned all it can. Its bias, not
           the sample size, is the limit.
           """, level=3),
        mc("m4", "In `GridSearchCV(make_pipeline(PolynomialFeatures(), Ridge()), param_grid, cv=5)`, what is the correct key for Ridge's `alpha`?",
           "`'ridge__alpha'`", ["`'alpha'`", "`'Ridge.alpha'`", "`'ridge_alpha'`"],
           """
           `make_pipeline` names each step after its class in lowercase, and
           a step's parameter is reached with a double underscore.
           """, level=2),
        out("o1", "What does this print?",
            """
            import numpy as np
            from sklearn.linear_model import Ridge
            from sklearn.model_selection import validation_curve
            rng = np.random.default_rng(0)
            X = rng.normal(size=(60, 3))
            y = X @ [1.0, -2.0, 0.5] + rng.normal(size=60)
            train, valid = validation_curve(Ridge(), X, y, param_name="alpha",
                                            param_range=[0.1, 1, 10, 100], cv=3)
            print(train.shape, valid.shape)
            """,
            """
            One row per hyperparameter value (4) and one column per fold (3),
            for both the training and the validation scores.
            """, level=2),
        out("o2", "What does this print?",
            """
            import numpy as np
            from sklearn.pipeline import make_pipeline
            from sklearn.preprocessing import PolynomialFeatures
            from sklearn.linear_model import LinearRegression
            from sklearn.model_selection import GridSearchCV
            rng = np.random.default_rng(1)
            x = rng.uniform(-3, 3, 80)
            y = 0.5 * x**3 - x + rng.normal(0, 1, 80)
            pipe = make_pipeline(PolynomialFeatures(), LinearRegression())
            grid = GridSearchCV(pipe, {"polynomialfeatures__degree": [1, 2, 3]}, cv=5)
            grid.fit(x[:, None], y)
            print(grid.best_params_)
            """,
            """
            The data come from a cubic, so degree 3 has the best
            cross-validated score among degrees 1 to 3.
            """, level=2),
        task("c1",
             """
             `x` (1-D) and `y` are already defined. Build a pipeline of
             `PolynomialFeatures` and `LinearRegression`, and run
             `GridSearchCV` with `cv=5` over degrees 1 through 6. Call the fitted
             search `grid` and store the chosen degree (an int) in `best_degree`.
             """,
             """
             from sklearn.pipeline import make_pipeline
             from sklearn.preprocessing import PolynomialFeatures
             from sklearn.linear_model import LinearRegression
             from sklearn.model_selection import GridSearchCV
             pipe = make_pipeline(PolynomialFeatures(), LinearRegression())
             grid = GridSearchCV(pipe, {"polynomialfeatures__degree": list(range(1, 7))}, cv=5)
             grid.fit(x[:, None], y)
             best_degree = grid.best_params_["polynomialfeatures__degree"]
             """,
             [
                 check("grid", 'check(hasattr(grid, "best_params_"), "`grid` should be a fitted GridSearchCV.")'),
                 check("range", """
                     degs = list(grid.param_grid["polynomialfeatures__degree"]) if "polynomialfeatures__degree" in grid.param_grid else []
                     check(sorted(degs) == [1, 2, 3, 4, 5, 6], "Search the degrees 1 to 6 under the key 'polynomialfeatures__degree'.")
                 """),
                 check("best", 'check(best_degree == 2, "`best_degree` should be the degree in `grid.best_params_`.")'),
             ],
             """
             The search fits each degree five times and keeps the one with the
             best mean validation score. The data are quadratic, so degree 2 wins.
             """,
             starter="""
             from sklearn.pipeline import make_pipeline
             from sklearn.preprocessing import PolynomialFeatures
             from sklearn.linear_model import LinearRegression
             from sklearn.model_selection import GridSearchCV
             grid = ...
             best_degree = ...
             """,
             setup="import numpy as np\nrng = np.random.default_rng(5)\nx = rng.uniform(0, 10, 120)\ny = 2 + 3 * x - 0.4 * x**2 + rng.normal(0, 1.5, 120)", level=3),
        task("c2",
             """
             `X` and `y` are already defined. Use `learning_curve` with a
             `LinearRegression`, `cv=5` and `train_sizes=[0.2, 0.5, 1.0]`. Store
             the three mean validation scores (one per size, averaged over
             folds) in `valid_mean`.
             """,
             """
             from sklearn.linear_model import LinearRegression
             from sklearn.model_selection import learning_curve
             sizes, train, valid = learning_curve(LinearRegression(), X, y, cv=5, train_sizes=[0.2, 0.5, 1.0])
             valid_mean = valid.mean(axis=1)
             """,
             [
                 check("shape", 'check(getattr(valid_mean, "shape", None) == (3,), "`valid_mean` should hold one mean per training size (3 values). Average across folds, axis=1.")'),
                 check("values", """
                     from sklearn.linear_model import LinearRegression
                     from sklearn.model_selection import learning_curve
                     _, _, v = learning_curve(LinearRegression(), X, y, cv=5, train_sizes=[0.2, 0.5, 1.0])
                     check(close(valid_mean, v.mean(axis=1)), "`valid_mean` should be the validation scores averaged over the 5 folds.")
                 """),
             ],
             """
             `learning_curve` returns the sizes used, then training and
             validation score arrays of shape (sizes, folds).
             """,
             starter="""
             from sklearn.linear_model import LinearRegression
             from sklearn.model_selection import learning_curve
             valid_mean = ...
             """,
             setup="import numpy as np\nrng = np.random.default_rng(2)\nX = rng.normal(size=(200, 4))\ny = X @ [1.0, 0.5, -1.0, 2.0] + rng.normal(size=200)", level=2),
    ]),

# ---------------------------------------------------------------- 5.04
concept(
    "py.ml.categorical_text", "Encoding categories and text", ["py.ml.estimator_api", "py.pd.series"],
    "5.04", "Feature Engineering", "05.04-feature-engineering",
    """
    Models need numbers. A region column with values North, South and West
    should not become 1, 2, 3: that claims West is "three times" North. Use
    **one-hot encoding** instead, one 0/1 column per category, exactly the
    dummy variables of a regression.

    ```python
    from sklearn.feature_extraction import DictVectorizer
    from sklearn.preprocessing import OneHotEncoder
    rows = [{"region": "North", "size": 3}, {"region": "West", "size": 1}]
    DictVectorizer(sparse=False).fit_transform(rows)
    OneHotEncoder(sparse_output=False).fit_transform([["North"], ["West"]])
    ```

    Text becomes numbers by counting words. `CountVectorizer` builds a
    vocabulary and gives each document a row of word counts.
    `TfidfVectorizer` reweights those counts so that words appearing in
    almost every document (like "the") count for less.

    Both return sparse matrices by default, since most entries are zero; call
    `.toarray()` to see a dense version.
    """,
    [
        mc("m1", "Why is coding a `sector` column as Agriculture=1, Manufacturing=2, Services=3 a poor choice for a linear model?",
           "It imposes an order and equal spacing between sectors that do not exist",
           ["Linear models cannot use integers", "The codes would be dropped as constants", "It makes the features matrix sparse"],
           """
           The model would treat Services as Agriculture plus two steps. One-hot
           dummies let each sector have its own effect.
           """, level=1),
        mc("m2", "With its default settings, which tokens does `CountVectorizer` keep from the text `'A tax on X'`?",
           "`'on'` and `'tax'`", ["`'A'`, `'tax'`, `'on'`, `'X'`", "`'a'`, `'tax'`, `'on'`, `'x'`", "`'tax'` only"],
           """
           It lowercases the text and its default token pattern keeps only
           tokens of two or more word characters, so 'a' and 'x' are dropped.
           """, level=2),
        mc("m3", "What does TF-IDF do compared with raw word counts?",
           "It down-weights words that appear in many documents, so distinctive words count for more",
           ["It removes all words shorter than four letters", "It counts each word at most once", "It sorts words by frequency"],
           """
           The inverse-document-frequency factor is small for words found in
           nearly every document and large for rare ones.
           """, level=2),
        out("o1", "What does this print?",
            """
            from sklearn.feature_extraction import DictVectorizer
            rows = [{"state": "NV", "beds": 120}, {"state": "UT", "beds": 80}, {"state": "NV", "beds": 45}]
            dv = DictVectorizer(sparse=False)
            M = dv.fit_transform(rows)
            print(dv.get_feature_names_out())
            print(M.shape)
            """,
            """
            Numeric values pass through as one column; each string value becomes
            its own `key=value` column. Names are sorted alphabetically.
            """, level=2),
        out("o2", "What does this print?",
            """
            from sklearn.feature_extraction.text import CountVectorizer
            docs = ["rates rise", "rates fall then rates rise"]
            cv = CountVectorizer()
            M = cv.fit_transform(docs)
            print(cv.get_feature_names_out())
            print(M.toarray())
            """,
            """
            The vocabulary is sorted. The second document mentions 'rates' twice,
            so its count in that column is 2.
            """, level=2),
        task("c1",
             """
             `region` is a list of region names (already defined). Use
             `OneHotEncoder(sparse_output=False)` to make `R`, a dense array with
             one 0/1 column per region, and store the encoder (fitted) in `enc`.
             The encoder expects a 2-D input: one column.
             """,
             """
             import numpy as np
             from sklearn.preprocessing import OneHotEncoder
             enc = OneHotEncoder(sparse_output=False)
             R = enc.fit_transform(np.array(region).reshape(-1, 1))
             """,
             [
                 check("shape", 'check(getattr(R, "shape", None) == (6, 3), "`R` should have 6 rows and one column per distinct region (3).")'),
                 check("values", """
                     check(close(R.sum(axis=1), [1] * 6), "Each row of `R` should have exactly one 1.")
                     check(close(R.sum(axis=0), [2, 3, 1]), "Column sums should be the counts of each region, in sorted order (East, North, West).")
                 """),
                 check("enc", 'check(hasattr(enc, "categories_"), "`enc` should be the fitted OneHotEncoder.")'),
             ],
             """
             Reshape the list to one column, then `fit_transform`. The categories
             are sorted, so the columns are East, North, West.
             """,
             starter="""
             import numpy as np
             from sklearn.preprocessing import OneHotEncoder
             enc = ...
             R = ...
             """,
             setup='region = ["North", "East", "North", "West", "East", "North"]', level=2),
        task("c2",
             """
             `notes` is a list of short survey answers (already defined). Use
             `TfidfVectorizer` to make `T`, a dense array of TF-IDF weights, and
             `vocab`, the array of feature names.
             """,
             """
             from sklearn.feature_extraction.text import TfidfVectorizer
             tf = TfidfVectorizer()
             T = tf.fit_transform(notes).toarray()
             vocab = tf.get_feature_names_out()
             """,
             [
                 check("vocab", 'check(list(vocab) == ["high", "low", "rent", "too", "wages"], "`vocab` should be the sorted vocabulary from the vectorizer.")'),
                 check("T", """
                     import numpy as np
                     check(getattr(T, "shape", None) == (3, 5), "`T` should be a dense array with one row per answer and one column per word. Use .toarray().")
                     check(close(np.linalg.norm(T, axis=1), [1, 1, 1]), "Each row of `T` should be TF-IDF weights (unit length by default).")
                 """),
             ],
             """
             `TfidfVectorizer` counts words and reweights them in one step; by
             default each row is scaled to unit length.
             """,
             starter="""
             from sklearn.feature_extraction.text import TfidfVectorizer
             T = ...
             vocab = ...
             """,
             setup='notes = ["rent too high", "wages too low", "high rent low wages"]', level=2),
    ]),

concept(
    "py.ml.pipelines", "Derived features, imputation and pipelines", ["py.ml.categorical_text", "py.ml.holdout_cv"],
    "5.04", "Feature Engineering", "05.04-feature-engineering",
    """
    Some features are built from others. `PolynomialFeatures` adds powers and
    products, so a linear model can fit a curve, much like adding
    experience-squared to a wage equation. `SimpleImputer` fills missing
    values with a column's mean, median or most frequent value.

    Chaining these steps by hand invites mistakes, so wrap them in a
    **pipeline**, which behaves like a single estimator:

    ```python
    from sklearn.pipeline import make_pipeline
    from sklearn.impute import SimpleImputer
    from sklearn.preprocessing import PolynomialFeatures
    from sklearn.linear_model import LinearRegression
    model = make_pipeline(SimpleImputer(strategy="median"),
                          PolynomialFeatures(degree=2),
                          LinearRegression())
    model.fit(X_train, y_train)
    model.predict(X_test)     # imputes and expands with training-set values
    ```

    Inside cross-validation, a pipeline refits the imputer on each training
    fold, so test-fold values never leak into the fill values.
    """,
    [
        mc("m1", "Why put a `SimpleImputer` inside the pipeline passed to `cross_val_score`, rather than imputing the whole data set first?",
           "So each fold's fill values are computed from its training rows only, keeping test rows out of the fit",
           ["Because SimpleImputer cannot run outside a pipeline", "Because it makes imputation faster", "So missing rows are dropped instead of filled"],
           """
           Imputing first uses the test rows' values to compute the means, a mild
           form of leakage. The pipeline refits every step per fold.
           """, level=2),
        mc("m2", "`PolynomialFeatures(degree=2, include_bias=False)` is applied to a single column x. Which columns come out?",
           "x and x squared", ["1, x and x squared", "x squared only", "x, x squared and x cubed"],
           "With `include_bias=False` the constant column of ones is left out.", level=2),
        mc("m3", "How many output columns does `PolynomialFeatures(degree=2)` produce from 2 input columns a and b (bias included)?",
           "6: 1, a, b, a squared, a times b, b squared", ["4", "5", "3"],
           "Degree 2 includes the interaction term a times b as well as the squares.", level=3),
        out("o1", "What does this print?",
            """
            from sklearn.preprocessing import PolynomialFeatures
            print(PolynomialFeatures(degree=3).fit_transform([[2]]))
            """,
            "The bias column 1, then 2, 2 squared and 2 cubed.", level=1),
        out("o2", "What does this print?",
            """
            import numpy as np
            from sklearn.impute import SimpleImputer
            X = np.array([[1.0, 10.0], [np.nan, 20.0], [3.0, np.nan], [8.0, 30.0]])
            print(SimpleImputer(strategy="median").fit_transform(X))
            """,
            """
            Each NaN is replaced by its column's median of the observed values:
            3 for the first column (of 1, 3, 8) and 20 for the second.
            """, level=2),
        out("o3", "What does this print?",
            """
            from sklearn.pipeline import make_pipeline
            from sklearn.impute import SimpleImputer
            from sklearn.linear_model import LinearRegression
            pipe = make_pipeline(SimpleImputer(), LinearRegression())
            print([name for name, step in pipe.steps])
            """,
            "`make_pipeline` names each step after its class, in lowercase.", level=1),
        task("c1",
             """
             `X` (with some NaN entries) and `y` are already defined. Build `pipe`,
             a pipeline that fills missing values with the column median and then
             fits a `LinearRegression`, and fit it on `X` and `y`.
             """,
             """
             from sklearn.pipeline import make_pipeline
             from sklearn.impute import SimpleImputer
             from sklearn.linear_model import LinearRegression
             pipe = make_pipeline(SimpleImputer(strategy="median"), LinearRegression())
             pipe.fit(X, y)
             """,
             [
                 check("steps", """
                     names = [n for n, s in getattr(pipe, "steps", [])]
                     check(names == ["simpleimputer", "linearregression"], "`pipe` should have an imputer followed by a linear regression.")
                     check(pipe.steps[0][1].strategy == "median", "Fill missing values with the median.")
                 """),
                 check("fitted", 'check(hasattr(pipe.steps[-1][1], "coef_"), "Fit the pipeline on X and y.")'),
             ],
             "A fitted pipeline fits each step in turn, passing transformed data forward.",
             starter="""
             from sklearn.pipeline import make_pipeline
             from sklearn.impute import SimpleImputer
             from sklearn.linear_model import LinearRegression
             pipe = ...
             """,
             setup="""
             import numpy as np
             rng = np.random.default_rng(4)
             X = rng.normal(size=(50, 3))
             y = X @ [2.0, -1.0, 0.5] + rng.normal(0, 0.1, 50)
             X[rng.random((50, 3)) < 0.1] = np.nan
             """, level=2),
        task("c2",
             """
             `exper` (1-D, years of experience) and `logwage` are already defined.
             The relationship is concave. Fit `model`, a pipeline of
             `PolynomialFeatures(degree=2)` and `LinearRegression`, on experience,
             and store its predicted log wage at 30 years in `at30` (a float).
             """,
             """
             from sklearn.pipeline import make_pipeline
             from sklearn.preprocessing import PolynomialFeatures
             from sklearn.linear_model import LinearRegression
             model = make_pipeline(PolynomialFeatures(degree=2), LinearRegression())
             model.fit(exper[:, None], logwage)
             at30 = float(model.predict([[30]])[0])
             """,
             [
                 check("fitted", 'check(hasattr(model, "predict") and hasattr(model.steps[-1][1], "coef_"), "`model` should be a fitted pipeline.")'),
                 check("pred", 'check(close(at30, 1.5 + 0.06 * 30 - 0.001 * 900, rtol=1e-4), "`at30` should be the model\'s prediction at 30 years. Did you include the squared term?")'),
             ],
             """
             This is the experience profile of a Mincer wage equation: a
             positive linear term and a negative squared term.
             """,
             starter="""
             from sklearn.pipeline import make_pipeline
             from sklearn.preprocessing import PolynomialFeatures
             from sklearn.linear_model import LinearRegression
             model = ...
             at30 = ...
             """,
             setup="import numpy as np\nexper = np.arange(0, 41, dtype=float)\nlogwage = 1.5 + 0.06 * exper - 0.001 * exper**2", level=2),
    ]),

# ---------------------------------------------------------------- 5.05
concept(
    "py.ml.naive_bayes", "Naive Bayes classification", ["py.ml.categorical_text", "py.ml.holdout_cv"],
    "5.05", "In Depth: Naive Bayes Classification", "05.05-naive-bayes",
    """
    Naive Bayes classifiers apply Bayes' rule with a bold simplification: given
    the class, every feature is assumed independent of the others. Each class
    then needs only a simple model per feature.

    **Gaussian naive Bayes** models each feature within each class as a normal
    distribution with its own mean and variance. **Multinomial naive Bayes**
    models counts, which suits word-count features from text.

    ```python
    from sklearn.naive_bayes import GaussianNB, MultinomialNB
    from sklearn.pipeline import make_pipeline
    from sklearn.feature_extraction.text import CountVectorizer
    gnb = GaussianNB().fit(X_train, y_train)
    gnb.predict_proba(X_test)        # class probabilities per row
    text_clf = make_pipeline(CountVectorizer(), MultinomialNB())
    ```

    They fit almost instantly and have few hyperparameters, so they make a
    good first baseline. Their probabilities are often poorly calibrated even
    when their predicted classes are good.
    """,
    [
        mc("m1", "What is the 'naive' assumption in naive Bayes?",
           "Features are independent of each other within each class",
           ["Every class is equally likely", "Features are normally distributed in every model", "The classes do not overlap"],
           """
           Independence given the class lets the model multiply one simple
           likelihood per feature. Equal priors are not assumed; they are
           estimated from class frequencies.
           """, level=1),
        mc("m2", "Which naive Bayes variant suits word counts from `CountVectorizer`?",
           "`MultinomialNB`", ["`GaussianNB`", "Neither; text cannot be classified with naive Bayes", "`GaussianNB` after taking logs"],
           "Multinomial NB models non-negative counts; Gaussian NB models continuous measurements.", level=1),
        mc("m3", "Why is naive Bayes a common first model?",
           "It is very fast, has few hyperparameters, and gives a quick baseline to beat",
           ["It is always the most accurate classifier", "It needs no training data", "It never overfits"],
           "A baseline tells you whether a more complex model is worth the cost.", level=2),
        out("o1", "What does this print?",
            """
            import numpy as np
            from sklearn.naive_bayes import GaussianNB
            X = np.array([[1.0], [1.5], [2.0], [8.0], [8.5], [9.0]])
            y = np.array([0, 0, 0, 1, 1, 1])
            gnb = GaussianNB().fit(X, y)
            print(gnb.predict([[1.2], [5.6], [10.0]]))
            """,
            """
            Class 0 is centred at 1.5 and class 1 at 8.5 with equal spread, so the
            boundary sits near 5: 5.6 goes to class 1.
            """, level=2),
        out("o2", "What does this print?",
            """
            from sklearn.pipeline import make_pipeline
            from sklearn.feature_extraction.text import CountVectorizer
            from sklearn.naive_bayes import MultinomialNB
            docs = ["late payment fee", "fee for late rent", "great service fast", "fast friendly service"]
            labels = ["complaint", "complaint", "praise", "praise"]
            clf = make_pipeline(CountVectorizer(), MultinomialNB()).fit(docs, labels)
            print(clf.predict(["rent fee", "friendly and fast"]))
            """,
            """
            'rent' and 'fee' were only seen in complaints; 'friendly' and 'fast'
            only in praise.
            """, level=2),
        task("c1",
             """
             Load `load_wine(return_X_y=True)`, split it with
             `train_test_split(..., test_size=0.3, random_state=1)`, fit a
             `GaussianNB` called `gnb` on the training part, and store its test
             accuracy in `acc`.
             """,
             """
             from sklearn.datasets import load_wine
             from sklearn.model_selection import train_test_split
             from sklearn.naive_bayes import GaussianNB
             X, y = load_wine(return_X_y=True)
             X_tr, X_te, y_tr, y_te = train_test_split(X, y, test_size=0.3, random_state=1)
             gnb = GaussianNB().fit(X_tr, y_tr)
             acc = gnb.score(X_te, y_te)
             """,
             [
                 check("model", 'check(hasattr(gnb, "theta_"), "`gnb` should be a fitted GaussianNB.")'),
                 check("acc", """
                     from sklearn.datasets import load_wine
                     from sklearn.model_selection import train_test_split
                     from sklearn.naive_bayes import GaussianNB
                     X_, y_ = load_wine(return_X_y=True)
                     a, b, c, d = train_test_split(X_, y_, test_size=0.3, random_state=1)
                     check(close(acc, GaussianNB().fit(a, c).score(b, d)), "`acc` should be the accuracy on the 30% test split with random_state=1.")
                 """),
             ],
             "For classifiers, `score` returns the share of correct predictions.",
             starter="""
             from sklearn.datasets import load_wine
             from sklearn.model_selection import train_test_split
             from sklearn.naive_bayes import GaussianNB
             gnb = ...
             acc = ...
             """, level=1),
        task("c2",
             """
             `texts` and `topics` (already defined) are short headlines and their
             topic labels. Build `clf`, a pipeline of `TfidfVectorizer` and
             `MultinomialNB`, fit it, and store its predictions for the two
             headlines in `new` in `guess`.
             """,
             """
             from sklearn.pipeline import make_pipeline
             from sklearn.feature_extraction.text import TfidfVectorizer
             from sklearn.naive_bayes import MultinomialNB
             clf = make_pipeline(TfidfVectorizer(), MultinomialNB()).fit(texts, topics)
             guess = clf.predict(new)
             """,
             [
                 check("steps", 'check([n for n, s in getattr(clf, "steps", [])] == ["tfidfvectorizer", "multinomialnb"], "`clf` should be TfidfVectorizer followed by MultinomialNB.")'),
                 check("guess", 'check(list(guess) == ["budget", "jobs"], "`guess` should be the predicted topics for the two headlines in `new`.")'),
             ],
             "A text pipeline takes raw strings, so you can call `predict` on new headlines directly.",
             starter="""
             from sklearn.pipeline import make_pipeline
             from sklearn.feature_extraction.text import TfidfVectorizer
             from sklearn.naive_bayes import MultinomialNB
             clf = ...
             guess = ...
             """,
             setup="""
             texts = ["deficit grows as tax revenue falls", "council passes budget with tax rise",
                      "spending cuts close budget gap", "unemployment falls as hiring rises",
                      "factory hiring slows", "wages rise in tight labour market"]
             topics = ["budget", "budget", "budget", "jobs", "jobs", "jobs"]
             new = ["tax revenue and spending", "hiring and wages"]
             """, level=2),
    ]),

# ---------------------------------------------------------------- 5.06
concept(
    "py.ml.linear_regression", "Linear regression and basis functions", ["py.ml.pipelines"],
    "5.06", "In Depth: Linear Regression", "05.06-linear-regression",
    """
    `LinearRegression` is ordinary least squares. With several features it
    fits one coefficient per column in `coef_` plus `intercept_`. Unlike a
    statistics package it reports no standard errors: the focus is
    prediction.

    "Linear" refers to the coefficients, not the features. Transform `x` into
    **basis functions** first and a linear model can follow curves:

    ```python
    model = make_pipeline(PolynomialFeatures(5), LinearRegression())
    ```

    Polynomials are one basis. Another is a set of **Gaussian bumps** centred
    along the x range, each column measuring how close x is to one centre:

    ```python
    centres = np.linspace(0, 10, 8)
    B = np.exp(-((x[:, None] - centres) ** 2) / (2 * 1.0 ** 2))
    LinearRegression().fit(B, y)
    ```

    More basis functions mean more flexibility and more risk of overfitting,
    which is where regularization (next) comes in.
    """,
    [
        mc("m1", "Why is `make_pipeline(PolynomialFeatures(3), LinearRegression())` still called a linear model?",
           "It is linear in its coefficients; only the inputs are transformed",
           ["It only fits straight lines", "PolynomialFeatures removes the curvature", "Because the intercept is linear"],
           "Least squares still solves for weights on fixed columns; the columns just happen to be powers of x.", level=2),
        mc("m2", "After fitting `LinearRegression` on `X` with columns `[educ, exper]`, what is `model.coef_[1]`?",
           "The predicted change in the target for one more unit of exper, holding educ fixed",
           ["The intercept", "The correlation between exper and the target", "The p-value for exper"],
           "Coefficients are partial effects in column order. No p-values are computed.", level=1),
        mc("m3", "What does each Gaussian basis column measure?",
           "How close x is to one centre: near 1 at the centre, falling towards 0 far away",
           ["The share of rows below one centre", "The squared distance from the mean of x", "A random draw around x"],
           "A weighted sum of bumps can trace almost any smooth curve.", level=2),
        out("o1", "What does this print?",
            """
            import numpy as np
            from sklearn.linear_model import LinearRegression
            rng = np.random.default_rng(0)
            X = rng.uniform(0, 10, size=(30, 3))
            y = 4 + X @ [2.0, -1.0, 0.5]
            model = LinearRegression().fit(X, y)
            print(np.round(model.coef_, 2), round(model.intercept_, 2))
            """,
            "With no noise, least squares recovers the true coefficients and intercept exactly.", level=1),
        out("o2", "What does this print?",
            """
            import numpy as np
            x = np.array([0.0, 2.0, 5.0])
            centres = np.array([0.0, 5.0])
            B = np.exp(-((x[:, None] - centres) ** 2) / 2)
            print(B.shape, np.round(B[:, 1], 2))
            """,
            """
            One row per x value and one column per centre. The second column is 1
            at x = 5, small at 2, and essentially 0 at 0.
            """, level=2),
        task("c1",
             """
             The DataFrame `df` has columns `educ`, `exper` and `logwage`. Fit a
             Mincer-style regression `model` of `logwage` on `educ`, `exper` and
             experience squared. Build `X` as a DataFrame with columns `educ`,
             `exper`, `exper2` (in that order), then store the return to a year of
             education (the `educ` coefficient) in `ret_educ`.
             """,
             """
             from sklearn.linear_model import LinearRegression
             X = df[["educ", "exper"]].copy()
             X["exper2"] = df["exper"] ** 2
             model = LinearRegression().fit(X, df["logwage"])
             ret_educ = model.coef_[0]
             """,
             [
                 check("X", 'check(list(getattr(X, "columns", [])) == ["educ", "exper", "exper2"], "`X` should have columns educ, exper, exper2 in that order.")'),
                 check("coef", 'check(hasattr(model, "coef_") and close(model.coef_, [0.08, 0.04, -0.0007], rtol=1e-3, atol=1e-6), "`model` should be fitted on X and logwage.")'),
                 check("ret", 'check(close(ret_educ, 0.08, rtol=1e-3), "`ret_educ` should be the coefficient on educ.")'),
             ],
             """
             The squared term lets earnings rise with experience at a decreasing
             rate. The educ coefficient is the classic return to schooling.
             """,
             starter="""
             from sklearn.linear_model import LinearRegression
             X = ...
             model = ...
             ret_educ = ...
             """,
             setup="""
             import numpy as np, pandas as pd
             rng = np.random.default_rng(7)
             educ = rng.integers(8, 21, 200).astype(float)
             exper = rng.integers(0, 41, 200).astype(float)
             df = pd.DataFrame({"educ": educ, "exper": exper,
                                "logwage": 1.0 + 0.08 * educ + 0.04 * exper - 0.0007 * exper**2})
             """, level=2),
        task("c2",
             """
             `x` and `y` (1-D) are already defined. Build `B`, a Gaussian basis
             with 10 centres from `np.linspace(0, 10, 10)` and width 1, where
             column j is `exp(-(x - c_j)**2 / 2)`. Fit `LinearRegression` on `B`
             as `model` and store its in-sample R-squared in `r2`.
             """,
             """
             import numpy as np
             from sklearn.linear_model import LinearRegression
             centres = np.linspace(0, 10, 10)
             B = np.exp(-((x[:, None] - centres) ** 2) / 2)
             model = LinearRegression().fit(B, y)
             r2 = model.score(B, y)
             """,
             [
                 check("B", """
                     import numpy as np
                     ref = np.exp(-((x[:, None] - np.linspace(0, 10, 10)) ** 2) / 2)
                     check(getattr(B, "shape", None) == (100, 10), "`B` should have one row per x value and 10 columns.")
                     check(close(B, ref), "Column j of `B` should be exp(-(x - c_j)**2 / 2). Broadcast x[:, None] against the centres.")
                 """),
                 check("r2", 'check(hasattr(model, "coef_") and close(r2, model.score(B, y)) and r2 > 0.9, "`r2` should be the fitted model\'s R-squared on B and y.")'),
             ],
             "Broadcasting a column of x against a row of centres builds the whole basis in one step.",
             starter="""
             import numpy as np
             from sklearn.linear_model import LinearRegression
             B = ...
             model = ...
             r2 = ...
             """,
             setup="import numpy as np\nrng = np.random.default_rng(8)\nx = np.sort(rng.uniform(0, 10, 100))\ny = np.sin(x) + 0.1 * rng.normal(size=100)", level=3),
    ]),

concept(
    "py.ml.regularization", "Ridge and lasso regularization", ["py.ml.linear_regression", "py.ml.model_selection"],
    "5.06", "In Depth: Linear Regression", "05.06-linear-regression",
    """
    With many basis functions or many regressors, least squares can chase
    noise and produce huge, offsetting coefficients. **Regularization** adds a
    penalty on coefficient size to the least-squares loss.

    - **Ridge** penalises the sum of squared coefficients. It shrinks all of
      them towards zero but rarely makes any exactly zero.
    - **Lasso** penalises the sum of absolute values. It sets some
      coefficients exactly to zero, so it also selects variables.

    ```python
    from sklearn.linear_model import Ridge, Lasso
    Ridge(alpha=1.0).fit(X, y)
    Lasso(alpha=0.1).fit(X, y).coef_     # some entries are 0.0
    ```

    `alpha` sets the penalty strength: 0 is plain OLS, larger is more
    shrinkage. Choose it by cross-validation. Because the penalty treats all
    coefficients alike, put features on a common scale first (for example
    with `StandardScaler` in a pipeline).
    """,
    [
        mc("m1", "Which penalty can set coefficients exactly to zero, acting as variable selection?",
           "Lasso (absolute values)", ["Ridge (squared values)", "Both equally often", "Neither"],
           """
           The corner of the absolute-value penalty at zero makes exact zeros
           common. Ridge shrinks smoothly and almost never lands on zero.
           """, level=1),
        mc("m2", "What happens to ridge coefficients as `alpha` grows very large?",
           "They shrink towards zero", ["They approach the OLS estimates", "They grow without bound", "They become exactly equal to each other"],
           "Large alpha makes the penalty dominate the fit; alpha near 0 gives OLS.", level=1),
        mc("m3", "Why standardise features before ridge or lasso?",
           "The penalty depends on coefficient size, which depends on each feature's units",
           ["Ridge cannot run on unscaled data", "Standardising removes the need for an intercept", "It makes the penalty smaller"],
           """
           Income in currency units and in thousands would get very different
           penalties otherwise. Scaling puts them on equal footing.
           """, level=2),
        mc("m4", "In econometric terms, what does regularization trade?",
           "A little bias in the coefficients for a reduction in their variance",
           ["Variance for consistency", "Out-of-sample fit for in-sample fit", "Nothing; it is unbiased"],
           """
           Penalised estimates are biased towards zero, but they vary less from
           sample to sample, which often improves out-of-sample prediction.
           """, level=2),
        out("o1", "What does this print?",
            """
            import numpy as np
            from sklearn.linear_model import Lasso
            rng = np.random.default_rng(0)
            X = rng.normal(size=(100, 8))
            y = 3 * X[:, 0] - 2 * X[:, 4] + rng.normal(size=100)
            lasso = Lasso(alpha=0.3).fit(X, y)
            print(np.flatnonzero(lasso.coef_))
            """,
            """
            Only columns 0 and 4 carry signal. The lasso penalty zeroes the
            other six coefficients.
            """, level=2),
        out("o2", "What does this print?",
            """
            import numpy as np
            from sklearn.linear_model import Ridge
            rng = np.random.default_rng(1)
            X = rng.normal(size=(50, 5))
            y = X @ [1.0, 2.0, -1.0, 0.5, 0.0] + rng.normal(size=50)
            sizes = [np.abs(Ridge(alpha=a).fit(X, y).coef_).sum() for a in [0.1, 10, 1000]]
            print([round(float(s), 2) for s in sizes])
            """,
            "The total size of the coefficients falls as alpha grows.", level=2),
        task("c1",
             """
             `X` (20 candidate regressors) and `y` are already defined. Fit
             `lasso = Lasso(alpha=0.2)` and store in `kept` the array of column
             indices whose coefficient is not zero.
             """,
             """
             import numpy as np
             from sklearn.linear_model import Lasso
             lasso = Lasso(alpha=0.2).fit(X, y)
             kept = np.flatnonzero(lasso.coef_)
             """,
             [
                 check("fitted", 'check(hasattr(lasso, "coef_") and lasso.alpha == 0.2, "`lasso` should be a Lasso with alpha=0.2, fitted on X and y.")'),
                 check("kept", 'check(same_array(kept, [1, 7, 12]), "`kept` should be the indices of the nonzero coefficients. np.flatnonzero helps.")'),
             ],
             "Lasso keeps the three columns that drive y and zeroes the other seventeen.",
             starter="""
             import numpy as np
             from sklearn.linear_model import Lasso
             lasso = ...
             kept = ...
             """,
             setup="import numpy as np\nrng = np.random.default_rng(11)\nX = rng.normal(size=(150, 20))\ny = 2 * X[:, 1] - 1.5 * X[:, 7] + X[:, 12] + rng.normal(0, 0.5, 150)", level=2),
        task("c2",
             """
             `X` and `y` are already defined. Make `pipe`, a pipeline of
             `StandardScaler` and `Ridge`, and use `GridSearchCV` with `cv=5`
             to choose `alpha` from `[0.01, 0.1, 1, 10, 100]`. Call the fitted
             search `grid` and store the chosen alpha in `best_alpha`.
             """,
             """
             from sklearn.pipeline import make_pipeline
             from sklearn.preprocessing import StandardScaler
             from sklearn.linear_model import Ridge
             from sklearn.model_selection import GridSearchCV
             pipe = make_pipeline(StandardScaler(), Ridge())
             grid = GridSearchCV(pipe, {"ridge__alpha": [0.01, 0.1, 1, 10, 100]}, cv=5).fit(X, y)
             best_alpha = grid.best_params_["ridge__alpha"]
             """,
             [
                 check("grid", """
                     check(hasattr(grid, "best_params_"), "`grid` should be a fitted GridSearchCV.")
                     check(sorted(grid.param_grid.get("ridge__alpha", [])) == [0.01, 0.1, 1, 10, 100], "Search alpha under the key 'ridge__alpha' over the five given values.")
                 """),
                 check("best", """
                     from sklearn.pipeline import make_pipeline
                     from sklearn.preprocessing import StandardScaler
                     from sklearn.linear_model import Ridge
                     from sklearn.model_selection import GridSearchCV
                     ref = GridSearchCV(make_pipeline(StandardScaler(), Ridge()), {"ridge__alpha": [0.01, 0.1, 1, 10, 100]}, cv=5).fit(X, y)
                     check(best_alpha == ref.best_params_["ridge__alpha"], "`best_alpha` should be the alpha in `grid.best_params_`.")
                 """),
             ],
             "Scaling inside the pipeline means each fold is scaled with its own training statistics.",
             starter="""
             from sklearn.pipeline import make_pipeline
             from sklearn.preprocessing import StandardScaler
             from sklearn.linear_model import Ridge
             from sklearn.model_selection import GridSearchCV
             grid = ...
             best_alpha = ...
             """,
             setup="import numpy as np\nrng = np.random.default_rng(12)\nX = rng.normal(size=(60, 30)) * rng.uniform(1, 50, 30)\ny = X[:, :5] @ [0.1, 0.05, -0.02, 0.03, 0.01] + rng.normal(size=60)", level=3),
    ]),

# ---------------------------------------------------------------- 5.07
concept(
    "py.ml.svm", "Support vector machines", ["py.ml.holdout_cv"],
    "5.07", "In Depth: Support Vector Machines", "05.07-support-vector-machines",
    """
    A support vector classifier draws the boundary between two classes that
    leaves the widest possible **margin** to the nearest points. Only those
    nearest points, the **support vectors**, decide where the boundary goes;
    moving any other point (without crossing the margin) changes nothing.

    ```python
    from sklearn.svm import SVC
    clf = SVC(kernel="linear", C=1.0).fit(X, y)
    clf.support_vectors_          # the points that pin the margin
    SVC(kernel="rbf", C=10).fit(X, y)   # curved boundaries
    ```

    The **kernel** lets the boundary bend: an `"rbf"` kernel works as if each
    point were mapped into a much richer feature space, without building it.
    **`C`** controls softness. Small `C` tolerates points inside the margin
    (a wider, smoother boundary); large `C` punishes them (a tighter fit that
    can overfit). Scale features first, since distances drive everything.
    """,
    [
        mc("m1", "Which training points determine a fitted SVM's decision boundary?",
           "Only the support vectors, the points on or inside the margin",
           ["All training points equally", "The points farthest from the boundary", "The class means"],
           "Points well clear of the margin have no effect on the fit.", level=1),
        mc("m2", "What does lowering `C` in `SVC` do?",
           "Softens the margin: more points may sit inside it, giving a smoother boundary",
           ["Hardens the margin so no point may violate it", "Changes the kernel to linear", "Removes the intercept"],
           """
           C is the cost of a margin violation. Cheaper violations allow a wider
           margin and usually more support vectors.
           """, level=2),
        mc("m3", "Two classes form concentric rings. Which `SVC` setting can separate them?",
           "`kernel='rbf'`", ["`kernel='linear'` with a large C", "`kernel='linear'` with a small C", "None; SVMs need linearly separable data"],
           "No straight line separates rings; a radial-basis kernel can draw a circle.", level=2),
        out("o1", "What does this print?",
            """
            import numpy as np
            from sklearn.svm import SVC
            X = np.array([[0, 0], [0, 1], [1, 0], [3, 3], [3, 4], [4, 3]], dtype=float)
            y = np.array([0, 0, 0, 1, 1, 1])
            clf = SVC(kernel="linear", C=100).fit(X, y)
            print(len(clf.support_vectors_), clf.predict([[1, 1], [3.5, 3.5]]))
            """,
            """
            Only the points nearest the gap between the groups become support
            vectors; the rest do not affect the boundary.
            """, level=2),
        out("o2", "What does this print?",
            """
            from sklearn.datasets import make_circles
            from sklearn.svm import SVC
            X, y = make_circles(n_samples=200, factor=0.4, noise=0.05, random_state=0)
            lin = SVC(kernel="linear").fit(X, y).score(X, y)
            rbf = SVC(kernel="rbf").fit(X, y).score(X, y)
            print(lin < 0.8, rbf > 0.95)
            """,
            """
            A line cannot separate an inner ring from an outer one; the RBF
            kernel can.
            """, level=2),
        task("c1",
             """
             `X` and `y` (two interleaved half-moons) are already defined. Split
             them with `train_test_split(X, y, random_state=0)`, fit `clf`, an
             `SVC` with an RBF kernel and `C=10`, on the training part, and store
             its test accuracy in `acc`.
             """,
             """
             from sklearn.svm import SVC
             from sklearn.model_selection import train_test_split
             X_tr, X_te, y_tr, y_te = train_test_split(X, y, random_state=0)
             clf = SVC(kernel="rbf", C=10).fit(X_tr, y_tr)
             acc = clf.score(X_te, y_te)
             """,
             [
                 check("model", 'check(getattr(clf, "kernel", None) == "rbf" and getattr(clf, "C", None) == 10 and hasattr(clf, "support_vectors_"), "`clf` should be a fitted SVC with kernel=\'rbf\' and C=10.")'),
                 check("acc", """
                     from sklearn.svm import SVC
                     from sklearn.model_selection import train_test_split
                     a, b, c, d = train_test_split(X, y, random_state=0)
                     check(close(acc, SVC(kernel="rbf", C=10).fit(a, c).score(b, d)), "`acc` should be the test accuracy on the split with random_state=0.")
                 """),
             ],
             "The curved RBF boundary follows the moons closely.",
             starter="""
             from sklearn.svm import SVC
             from sklearn.model_selection import train_test_split
             clf = ...
             acc = ...
             """,
             setup="from sklearn.datasets import make_moons\nX, y = make_moons(n_samples=200, noise=0.2, random_state=3)", level=2),
        task("c2",
             """
             `X` and `y` are already defined. For each C in `[0.01, 1, 100]` fit
             `SVC(kernel="linear", C=C)` on all the data and record its number of
             support vectors. Store the three counts, in that order, in a list
             `n_sv`.
             """,
             """
             from sklearn.svm import SVC
             n_sv = [len(SVC(kernel="linear", C=C).fit(X, y).support_vectors_) for C in [0.01, 1, 100]]
             """,
             [
                 check("n_sv", """
                     from sklearn.svm import SVC
                     ref = [len(SVC(kernel="linear", C=C).fit(X, y).support_vectors_) for C in [0.01, 1, 100]]
                     check(list(n_sv) == ref, "`n_sv` should hold the support-vector counts for C = 0.01, 1, 100.")
                 """),
             ],
             "A softer margin (small C) lets more points inside it, so more of them become support vectors.",
             starter="""
             from sklearn.svm import SVC
             n_sv = ...
             """,
             setup="from sklearn.datasets import make_blobs\nX, y = make_blobs(n_samples=100, centers=2, cluster_std=1.5, random_state=2)", level=2),
    ]),

# ---------------------------------------------------------------- 5.08
concept(
    "py.ml.forests", "Decision trees and random forests", ["py.ml.model_selection"],
    "5.08", "In Depth: Decision Trees and Random Forests", "05.08-random-forests",
    """
    A **decision tree** asks a sequence of yes/no questions about single
    features ("income above 40?") and predicts from the leaf a row lands in.
    Grown without limit, a tree keeps splitting until each leaf is pure, so it
    fits the training data perfectly and overfits badly.

    A **random forest** averages many trees, each grown on a bootstrap sample
    of rows and choosing among random subsets of features at each split. The
    individual trees overfit in different ways, and averaging cancels much of
    that variance, the same logic as averaging many noisy estimators.

    ```python
    from sklearn.ensemble import RandomForestClassifier, RandomForestRegressor
    rf = RandomForestRegressor(n_estimators=200, random_state=0)
    rf.fit(X_train, y_train)
    rf.feature_importances_       # sums to 1
    ```

    Forests need little tuning, handle nonlinearity and interactions without
    you specifying them, and require no scaling.
    """,
    [
        mc("m1", "A `DecisionTreeClassifier()` with default settings scores 1.0 on its training data and 0.78 on test data. What is happening?",
           "The unrestricted tree has overfit; limiting `max_depth` or using a forest would help",
           ["The tree is underfitting", "The test set is mislabelled", "Trees always score 1.0 on test data when tuned"],
           "Without a depth limit, the tree keeps splitting until it memorises the training rows.", level=2),
        mc("m2", "Why does a random forest generalise better than a single deep tree?",
           "Averaging many differently-overfit trees cancels much of their variance",
           ["Each tree in the forest is shallower than a single tree", "Forests use linear models in the leaves", "Forests never use bootstrap samples"],
           """
           Bootstrap rows and random feature subsets make the trees differ;
           their errors partly offset when averaged.
           """, level=2),
        mc("m3", "Which preprocessing step do random forests NOT need?",
           "Scaling features to a common range", ["Encoding text as numbers", "Arranging data as a 2-D features matrix", "Handling the target's format"],
           "Splits compare one feature with a threshold, so a feature's units do not matter.", level=2),
        out("o1", "What does this print?",
            """
            from sklearn.datasets import load_breast_cancer
            from sklearn.tree import DecisionTreeClassifier
            X, y = load_breast_cancer(return_X_y=True)
            for depth in [1, None]:
                tree = DecisionTreeClassifier(max_depth=depth, random_state=0).fit(X, y)
                print(depth, round(tree.score(X, y), 3))
            """,
            """
            A single split (a 'stump') gets most rows right; an unlimited tree
            fits every training row.
            """, level=2),
        out("o2", "What does this print?",
            """
            from sklearn.datasets import make_classification
            from sklearn.ensemble import RandomForestClassifier
            X, y = make_classification(n_samples=100, n_features=4, random_state=0)
            rf = RandomForestClassifier(n_estimators=25, random_state=0).fit(X, y)
            print(len(rf.estimators_), round(rf.feature_importances_.sum(), 3))
            """,
            "The forest stores one fitted tree per estimator, and importances are normalised to sum to 1.", level=1),
        task("c1",
             """
             `X_tr`, `X_te`, `y_tr`, `y_te` are a train/test split of a
             nonlinear regression problem (already defined). Fit `rf`, a
             `RandomForestRegressor(n_estimators=100, random_state=0)`, and `ols`,
             a `LinearRegression`, on the training data. Store their test
             R-squared values in `rf_r2` and `ols_r2`.
             """,
             """
             from sklearn.ensemble import RandomForestRegressor
             from sklearn.linear_model import LinearRegression
             rf = RandomForestRegressor(n_estimators=100, random_state=0).fit(X_tr, y_tr)
             ols = LinearRegression().fit(X_tr, y_tr)
             rf_r2 = rf.score(X_te, y_te)
             ols_r2 = ols.score(X_te, y_te)
             """,
             [
                 check("models", 'check(hasattr(rf, "estimators_") and hasattr(ols, "coef_"), "Fit both `rf` and `ols` on the training data.")'),
                 check("scores", """
                     check(close(rf_r2, rf.score(X_te, y_te)) and close(ols_r2, ols.score(X_te, y_te)), "Store each model's R-squared on the test rows.")
                     check(rf_r2 > ols_r2, "With this threshold-shaped data the forest should beat OLS out of sample. Check you scored on the test rows.")
                 """),
             ],
             """
             The target jumps at a threshold and has an interaction. A forest finds
             both without being told; OLS needs you to specify them.
             """,
             starter="""
             from sklearn.ensemble import RandomForestRegressor
             from sklearn.linear_model import LinearRegression
             rf = ...
             ols = ...
             rf_r2 = ...
             ols_r2 = ...
             """,
             setup="""
             import numpy as np
             from sklearn.model_selection import train_test_split
             rng = np.random.default_rng(9)
             X = rng.uniform(0, 10, size=(400, 2))
             y = 5 * (X[:, 0] > 6) + X[:, 0] * X[:, 1] / 5 + rng.normal(0, 0.5, 400)
             X_tr, X_te, y_tr, y_te = train_test_split(X, y, random_state=0)
             """, level=2),
        task("c2",
             """
             Using `load_wine(return_X_y=True)`, fit `rf`, a
             `RandomForestClassifier(n_estimators=100, random_state=0)`, on all
             the data. Store in `top` the index of the most important feature
             according to `feature_importances_`.
             """,
             """
             import numpy as np
             from sklearn.datasets import load_wine
             from sklearn.ensemble import RandomForestClassifier
             X, y = load_wine(return_X_y=True)
             rf = RandomForestClassifier(n_estimators=100, random_state=0).fit(X, y)
             top = int(np.argmax(rf.feature_importances_))
             """,
             [
                 check("rf", 'check(hasattr(rf, "feature_importances_") and rf.n_estimators == 100, "`rf` should be a fitted RandomForestClassifier with 100 trees.")'),
                 check("top", 'check(top == int(rf.feature_importances_.argmax()), "`top` should be the index of the largest feature importance. Try np.argmax.")'),
             ],
             "`argmax` gives the position of the largest importance.",
             starter="""
             import numpy as np
             from sklearn.datasets import load_wine
             from sklearn.ensemble import RandomForestClassifier
             rf = ...
             top = ...
             """, level=1),
    ]),

# ---------------------------------------------------------------- 5.09
concept(
    "py.ml.pca", "Principal component analysis", ["py.ml.estimator_api", "py.np.aggregates"],
    "5.09", "In Depth: Principal Component Analysis", "05.09-principal-component-analysis",
    """
    PCA finds new axes for the data, ordered by how much variance each
    captures. The first **principal component** is the direction along which
    the rows spread out most; the second is the most spread-out direction
    perpendicular to it, and so on. Keeping only the first few compresses
    the data with little loss.

    ```python
    from sklearn.decomposition import PCA
    pca = PCA(n_components=3).fit(X)
    pca.explained_variance_ratio_      # share of total variance per component
    Z = pca.transform(X)               # coordinates on the new axes
    X_hat = pca.inverse_transform(Z)   # back to the original columns
    PCA(n_components=0.90)             # keep enough components for 90%
    ```

    Projecting onto a few components and back is a way to **filter noise**:
    the leading components carry the shared signal, and the many small ones
    are mostly noise. Many correlated price indices, for instance, collapse
    onto one or two common factors. Standardise first when columns have
    different units.
    """,
    [
        mc("m1", "What does the first principal component represent?",
           "The direction in feature space along which the data vary the most",
           ["The feature most correlated with the target", "The column with the largest mean", "The first column of X"],
           "PCA is unsupervised; it looks only at the spread of X.", level=1),
        mc("m2", "What does `PCA(n_components=0.95)` do?",
           "Keeps the smallest number of components that together explain at least 95% of the variance",
           ["Keeps 95 components", "Drops 5% of the rows", "Keeps components whose individual share exceeds 0.95"],
           "A float between 0 and 1 is read as a target share of variance.", level=2),
        mc("m3", "One column is income in currency units (variance in the millions) and the rest are rates between 0 and 1. What happens if you run PCA without scaling?",
           "The first component will point almost entirely along income, because it dominates the variance",
           ["PCA scales the columns automatically", "Income is dropped", "The rates dominate because there are more of them"],
           "PCA maximises raw variance, so units matter. Standardise first.", level=2),
        out("o1", "What does this print?",
            """
            import numpy as np
            from sklearn.decomposition import PCA
            rng = np.random.default_rng(0)
            t = rng.normal(size=200)
            X = np.column_stack([t, 2 * t, -t])
            print(np.round(PCA().fit(X).explained_variance_ratio_, 3))
            """,
            """
            All three columns are multiples of one variable, so the data lie on a
            line: the first component explains everything.
            """, level=2),
        out("o2", "What does this print?",
            """
            from sklearn.datasets import load_digits
            from sklearn.decomposition import PCA
            X, _ = load_digits(return_X_y=True)
            pca = PCA(n_components=0.9).fit(X)
            print(X.shape[1], pca.n_components_)
            """,
            """
            The 64 pixel columns are highly correlated, so far fewer components
            cover 90% of the variance.
            """, level=2),
        task("c1",
             """
             Load `load_wine(return_X_y=True)` and standardise `X` with
             `StandardScaler`. Fit `pca = PCA()` on the scaled data and store in
             `k80` the smallest number of components whose cumulative explained
             variance ratio reaches 0.8.
             """,
             """
             import numpy as np
             from sklearn.datasets import load_wine
             from sklearn.preprocessing import StandardScaler
             from sklearn.decomposition import PCA
             X, y = load_wine(return_X_y=True)
             Xs = StandardScaler().fit_transform(X)
             pca = PCA().fit(Xs)
             k80 = int(np.argmax(np.cumsum(pca.explained_variance_ratio_) >= 0.8) + 1)
             """,
             [
                 check("pca", 'check(hasattr(pca, "explained_variance_ratio_") and len(pca.explained_variance_ratio_) == 13, "`pca` should be a PCA with all components, fitted on the 13 scaled wine features.")'),
                 check("scaled", 'check(close(pca.mean_, [0] * 13, atol=1e-8), "Fit PCA on the standardised data (each column mean 0).")'),
                 check("k80", 'check(k80 == 5, "`k80` should be the count of components needed for a cumulative share of at least 0.8. Use np.cumsum.")'),
             ],
             """
             `np.cumsum` of the ratios gives running totals; the first position
             reaching 0.8, plus one, is the count.
             """,
             starter="""
             import numpy as np
             from sklearn.datasets import load_wine
             from sklearn.preprocessing import StandardScaler
             from sklearn.decomposition import PCA
             pca = ...
             k80 = ...
             """, level=2),
        task("c2",
             """
             `noisy` (already defined) is a 100 x 12 panel of price indices: a
             low-dimensional common signal plus noise. Filter it by fitting
             `pca = PCA(n_components=2)`, projecting and mapping back. Store the
             reconstructed array (same shape as `noisy`) in `smooth`.
             """,
             """
             from sklearn.decomposition import PCA
             pca = PCA(n_components=2).fit(noisy)
             smooth = pca.inverse_transform(pca.transform(noisy))
             """,
             [
                 check("shape", 'check(getattr(smooth, "shape", None) == (100, 12), "`smooth` should have the same shape as `noisy`. Use inverse_transform.")'),
                 check("closer", """
                     import numpy as np
                     check(getattr(pca, "n_components_", None) == 2, "Use 2 components.")
                     check(np.abs(smooth - clean).mean() < np.abs(noisy - clean).mean(), "`smooth` should be closer to the underlying signal than `noisy` is.")
                 """),
             ],
             "Projecting onto two components keeps the common factors; mapping back drops the small, noisy directions.",
             starter="""
             from sklearn.decomposition import PCA
             pca = ...
             smooth = ...
             """,
             setup="""
             import numpy as np
             rng = np.random.default_rng(10)
             factors = rng.normal(size=(100, 2))
             loadings = rng.normal(size=(2, 12))
             clean = factors @ loadings
             noisy = clean + rng.normal(0, 0.5, size=(100, 12))
             """, level=2),
    ]),

# ---------------------------------------------------------------- 5.10
concept(
    "py.ml.manifold", "Manifold learning", ["py.ml.pca"],
    "5.10", "In Depth: Manifold Learning", "05.10-manifold-learning",
    """
    PCA finds flat directions. When data lie on a curved surface, like a
    sheet of paper rolled up in three dimensions, no flat projection unrolls
    it. **Manifold learning** methods look for a low-dimensional layout that
    respects the data's own geometry.

    - **MDS** (multidimensional scaling) places points so that their
      distances match the original pairwise distances as well as possible.
    - **LLE** (locally linear embedding) keeps only each point's relationship
      to its nearest neighbours, so it can unroll curved sheets.
    - **Isomap** measures distance along the surface, through chains of
      neighbours, and then lays those distances out flat.

    ```python
    from sklearn.manifold import Isomap
    Z = Isomap(n_neighbors=10, n_components=2).fit_transform(X)
    ```

    These methods are slower than PCA, sensitive to `n_neighbors` and noise,
    and their axes have no direct meaning. Use them mainly to explore and
    visualise.
    """,
    [
        mc("m1", "What does metric MDS try to preserve?",
           "The pairwise distances between points", ["The variance of each column", "Each point's class label", "The order of the rows"],
           "MDS searches for a low-dimensional layout whose distances match the originals.", level=1),
        mc("m2", "Why can Isomap unroll a rolled-up sheet when PCA cannot?",
           "It measures distance along the sheet through chains of neighbours, not straight through the roll",
           ["It uses more components than PCA", "It standardises the data first", "It is supervised"],
           """
           Two points on adjacent layers of the roll are close in a straight line
           but far apart along the surface; Isomap uses the latter.
           """, level=2),
        mc("m3", "In scikit-learn 1.9, which of these estimators has no `transform` method for embedding new points after fitting?",
           "`MDS`", ["`Isomap`", "`LocallyLinearEmbedding`", "`PCA`"],
           """
           MDS only offers `fit_transform` on the data it was given. Isomap,
           LLE and PCA can map new rows with `transform`.
           """, level=3),
        out("o1", "What does this print?",
            """
            from sklearn.datasets import make_s_curve
            from sklearn.manifold import Isomap
            X, t = make_s_curve(n_samples=300, random_state=0)
            Z = Isomap(n_neighbors=10, n_components=2).fit_transform(X)
            print(X.shape, Z.shape)
            """,
            "Isomap maps each 3-D point to 2 coordinates on the unrolled sheet.", level=1),
        out("o2", "What does this print?",
            """
            import numpy as np
            from scipy.spatial.distance import pdist
            from sklearn.manifold import MDS
            rng = np.random.default_rng(0)
            P = rng.normal(size=(8, 2)) @ np.array([[1.0, 0.0, 2.0], [0.0, 1.0, 1.0]])
            Z = MDS(n_components=2, init="classical_mds", random_state=0).fit_transform(P)
            print(Z.shape, round(float(np.abs(pdist(Z) - pdist(P)).max()), 3))
            """,
            """
            These 3-D points lie exactly on a plane, so a 2-D layout can match
            every pairwise distance: the largest error is 0.
            """, level=3),
        task("c1",
             """
             `X` (points on an S-shaped sheet) and `t` (each point's position
             along the sheet) are already defined. Embed `X` in 2 dimensions with
             `Isomap(n_neighbors=12, n_components=2)` and store the result in `Z`.
             Then store in `corr` the absolute correlation between `t` and the
             first column of `Z` (use `np.corrcoef`).
             """,
             """
             import numpy as np
             from sklearn.manifold import Isomap
             Z = Isomap(n_neighbors=12, n_components=2).fit_transform(X)
             corr = abs(np.corrcoef(t, Z[:, 0])[0, 1])
             """,
             [
                 check("Z", 'check(getattr(Z, "shape", None) == (300, 2), "`Z` should be the 2-D Isomap embedding (300 rows, 2 columns).")'),
                 check("corr", """
                     import numpy as np
                     check(close(corr, abs(np.corrcoef(t, Z[:, 0])[0, 1])), "`corr` should be the absolute correlation of t with Z[:, 0].")
                     check(corr > 0.9, "Isomap's first coordinate should track position along the sheet closely.")
                 """),
             ],
             """
             Isomap's first coordinate follows the sheet's long direction, so it
             nearly reproduces `t` (up to sign).
             """,
             starter="""
             import numpy as np
             from sklearn.manifold import Isomap
             Z = ...
             corr = ...
             """,
             setup="from sklearn.datasets import make_s_curve\nX, t = make_s_curve(n_samples=300, random_state=1)", level=2),
    ]),

# ---------------------------------------------------------------- 5.11
concept(
    "py.ml.kmeans", "k-means clustering in scikit-learn", ["py.ml.estimator_api"],
    "5.11", "In Depth: k-Means Clustering", "05.11-k-means",
    """
    k-means splits rows into k clusters, each summarised by its centre. It
    alternates two steps until nothing changes: assign every point to the
    nearest centre, then move each centre to the mean of its points.

    ```python
    from sklearn.cluster import KMeans
    km = KMeans(n_clusters=3, random_state=0).fit(X)
    km.labels_              # cluster of each row
    km.cluster_centers_     # one row per centre
    km.inertia_             # total squared distance to centres
    ```

    You must choose k. Inertia always falls as k rises (more centres are
    always closer), so look for an "elbow" where it stops falling quickly,
    or compare silhouette scores.

    k-means assumes round clusters of similar size, because it only looks at
    distance to a centre. Elongated or nested groups defeat it, and since it
    can stop at a local optimum it is run from several starts. The cluster
    numbers themselves are arbitrary labels.
    """,
    [
        mc("m1", "Why can't you choose k by picking the value with the lowest inertia?",
           "Inertia always falls as k grows, reaching 0 when every point is its own cluster",
           ["Inertia is random and cannot be compared", "Inertia rises with k", "KMeans does not compute inertia"],
           "Look for the elbow where extra clusters stop buying much reduction.", level=2),
        mc("m2", "Which data set is k-means most likely to cluster badly?",
           "Two concentric rings", ["Three well-separated round blobs", "Two blobs of equal size", "Four tight, round groups"],
           """
           Each k-means cluster is the region closest to a centre, so the
           boundaries are straight. Rings need a nonlinear method.
           """, level=2),
        mc("m3", "You run KMeans twice with different `random_state` values and the labels come out as [0, 0, 1, 1] and [1, 1, 0, 0]. What does this mean?",
           "The clusters are the same; only the arbitrary label numbers are swapped",
           ["The two runs found different clusters", "One run failed to converge", "The data have no cluster structure"],
           "Cluster numbers have no meaning; compare which points are grouped together.", level=1),
        out("o1", "What does this print?",
            """
            import numpy as np
            from sklearn.datasets import make_blobs
            from sklearn.cluster import KMeans
            X, _ = make_blobs(n_samples=[50, 30, 20], random_state=4, cluster_std=0.6)
            km = KMeans(n_clusters=3, random_state=0).fit(X)
            print(sorted(np.bincount(km.labels_).tolist()))
            """,
            """
            The three well-separated blobs have 50, 30 and 20 points; sorting the
            counts hides the arbitrary label numbers.
            """, level=2),
        out("o2", "What does this print?",
            """
            from sklearn.datasets import make_blobs
            from sklearn.cluster import KMeans
            X, _ = make_blobs(n_samples=150, centers=3, random_state=1)
            inertia = [KMeans(n_clusters=k, random_state=0).fit(X).inertia_ for k in range(1, 6)]
            print(all(a > b for a, b in zip(inertia, inertia[1:])))
            """,
            "Adding a centre can only bring points closer, so inertia falls with every k.", level=1),
        task("c1",
             """
             `X` (already defined) holds two indicators for 200 regions. Fit
             `km = KMeans(n_clusters=4, random_state=0)` and store the cluster
             centres in `centres` and the number of regions in the largest cluster
             in `biggest`.
             """,
             """
             import numpy as np
             from sklearn.cluster import KMeans
             km = KMeans(n_clusters=4, random_state=0).fit(X)
             centres = km.cluster_centers_
             biggest = int(np.bincount(km.labels_).max())
             """,
             [
                 check("km", 'check(getattr(km, "n_clusters", None) == 4 and hasattr(km, "labels_"), "`km` should be a KMeans with 4 clusters, fitted on X.")'),
                 check("centres", 'check(getattr(centres, "shape", None) == (4, 2), "`centres` should hold 4 centres with 2 coordinates each.")'),
                 check("biggest", 'check(biggest == 80, "`biggest` should count the rows in the largest cluster. np.bincount on the labels helps.")'),
             ],
             "`np.bincount(labels)` counts rows per cluster.",
             starter="""
             import numpy as np
             from sklearn.cluster import KMeans
             km = ...
             centres = ...
             biggest = ...
             """,
             setup="from sklearn.datasets import make_blobs\nX, _ = make_blobs(n_samples=[80, 50, 40, 30], cluster_std=0.5, random_state=6)", level=2),
        task("c2",
             """
             `X` is already defined. Compute `inertias`, a list with the
             KMeans inertia for k = 1 to 8 (use `random_state=0`). Then store in
             `drop` a list of the 7 successive decreases (inertia at k minus
             inertia at k+1).
             """,
             """
             from sklearn.cluster import KMeans
             inertias = [KMeans(n_clusters=k, random_state=0).fit(X).inertia_ for k in range(1, 9)]
             drop = [a - b for a, b in zip(inertias, inertias[1:])]
             """,
             [
                 check("inertias", """
                     from sklearn.cluster import KMeans
                     check(len(inertias) == 8, "`inertias` should have one value for each k from 1 to 8.")
                     check(close(inertias[2], KMeans(n_clusters=3, random_state=0).fit(X).inertia_), "Use KMeans(n_clusters=k, random_state=0).inertia_ for each k.")
                 """),
                 check("drop", """
                     import numpy as np
                     check(len(drop) == 7 and close(drop, -np.diff(inertias)), "`drop[i]` should be inertias[i] - inertias[i + 1].")
                 """),
             ],
             """
             The drops are large up to the true number of clusters and small after
             it: that change in slope is the elbow.
             """,
             starter="""
             from sklearn.cluster import KMeans
             inertias = ...
             drop = ...
             """,
             setup="from sklearn.datasets import make_blobs\nX, _ = make_blobs(n_samples=200, centers=3, random_state=7)", level=2),
    ]),

# ---------------------------------------------------------------- 5.12
concept(
    "py.ml.gmm", "Gaussian mixture models", ["py.ml.kmeans"],
    "5.12", "In Depth: Gaussian Mixture Models", "05.12-gaussian-mixtures",
    """
    A **Gaussian mixture model** says each row was drawn from one of k normal
    distributions, each with its own mean, covariance and weight. Fitting by
    expectation-maximisation gives **soft** assignments: a probability of
    belonging to each component, not a single label.

    ```python
    from sklearn.mixture import GaussianMixture
    gmm = GaussianMixture(n_components=3, covariance_type="full", random_state=0).fit(X)
    gmm.predict_proba(X)       # one row of probabilities per point
    gmm.weights_, gmm.means_
    gmm.bic(X)                 # lower is better
    new_rows, _ = gmm.sample(100)
    ```

    `covariance_type` sets each component's shape: `"spherical"` (round),
    `"diag"` (axis-aligned ellipses) or `"full"` (any ellipse). Full handles
    elongated clusters that defeat k-means.

    Because it is a full probability model, a GMM is also a **density
    estimator**: it can generate new data. Choose the number of components
    with AIC or BIC, which reward fit and penalise extra parameters, much as
    in model selection for time-series lags.
    """,
    [
        mc("m1", "What does a GMM give for each point that k-means does not?",
           "A probability of belonging to each component", ["A distance to the nearest centre", "A single hard label", "A predicted target value"],
           "`predict_proba` returns soft assignments; points between clusters get split probabilities.", level=1),
        mc("m2", "You compare GMMs with 1 to 6 components using BIC. Which do you pick?",
           "The one with the lowest BIC", ["The one with the highest BIC", "The one with the most components", "The one where BIC first goes negative"],
           "BIC adds a penalty for parameters to the negative log-likelihood, so smaller is better.", level=1),
        mc("m3", "Which `covariance_type` lets each component be an ellipse tilted at any angle?",
           "`'full'`", ["`'spherical'`", "`'diag'`", "None; GMM components are always round"],
           """
           Full covariance matrices allow correlation between features within a
           component. Diagonal ones keep ellipses aligned with the axes.
           """, level=2),
        out("o1", "What does this print?",
            """
            import numpy as np
            from sklearn.mixture import GaussianMixture
            rng = np.random.default_rng(0)
            X = np.concatenate([rng.normal(0, 1, 300), rng.normal(6, 1, 100)])[:, None]
            gmm = GaussianMixture(n_components=2, random_state=0).fit(X)
            print(np.round(np.sort(gmm.weights_), 2), np.round(np.sort(gmm.means_.ravel()), 1) + 0.0)
            """,
            """
            Three quarters of the draws come from the component at 0 and one
            quarter from the one at 6. Sorting hides the arbitrary component order.
            """, level=2),
        out("o2", "What does this print?",
            """
            from sklearn.datasets import make_blobs
            from sklearn.mixture import GaussianMixture
            X, _ = make_blobs(n_samples=300, centers=3, cluster_std=0.8, random_state=2)
            bic = [GaussianMixture(n_components=k, random_state=0).fit(X).bic(X) for k in range(1, 6)]
            print(bic.index(min(bic)) + 1)
            """,
            "BIC is lowest at the true number of components, 3.", level=2),
        task("c1",
             """
             `X` (already defined) has two elongated, tilted clusters. Fit
             `gmm = GaussianMixture(n_components=2, covariance_type="full",
             random_state=0)` and store in `probs` the array of membership
             probabilities for every row. Then store in `unsure` the number of
             rows whose largest probability is below 0.9.
             """,
             """
             from sklearn.mixture import GaussianMixture
             gmm = GaussianMixture(n_components=2, covariance_type="full", random_state=0).fit(X)
             probs = gmm.predict_proba(X)
             unsure = int((probs.max(axis=1) < 0.9).sum())
             """,
             [
                 check("gmm", 'check(getattr(gmm, "covariance_type", None) == "full" and hasattr(gmm, "means_"), "`gmm` should be a fitted GaussianMixture with full covariances.")'),
                 check("probs", """
                     check(getattr(probs, "shape", None) == (len(X), 2), "`probs` should have one row per point and one column per component.")
                     check(close(probs.sum(axis=1), [1] * len(X)), "Each row of `probs` should sum to 1. Use predict_proba.")
                 """),
                 check("unsure", 'check(unsure == int((probs.max(axis=1) < 0.9).sum()), "`unsure` should count rows whose largest probability is under 0.9.")'),
             ],
             "Points in the overlap between the ellipses get split probabilities.",
             starter="""
             from sklearn.mixture import GaussianMixture
             gmm = ...
             probs = ...
             unsure = ...
             """,
             setup="""
             import numpy as np
             rng = np.random.default_rng(13)
             A = rng.normal(size=(150, 2)) @ np.array([[2.0, 0.0], [1.5, 0.4]])
             B = rng.normal(size=(150, 2)) @ np.array([[2.0, 0.0], [1.5, 0.4]]) + [0.0, 3.0]
             X = np.vstack([A, B])
             """, level=2),
        task("c2",
             """
             `incomes` (a 1-D array of log incomes) is already defined. Fit
             GaussianMixture models with 1 to 5 components (`random_state=0`) on
             it as a single column, and store in `best_k` the number of
             components with the lowest BIC. Then fit the chosen model as `gmm`
             and draw `fake`, 500 new incomes from it with `gmm.sample(500)`,
             keeping just the sampled values.
             """,
             """
             from sklearn.mixture import GaussianMixture
             X = incomes[:, None]
             bics = [GaussianMixture(n_components=k, random_state=0).fit(X).bic(X) for k in range(1, 6)]
             best_k = bics.index(min(bics)) + 1
             gmm = GaussianMixture(n_components=best_k, random_state=0).fit(X)
             fake, _ = gmm.sample(500)
             """,
             [
                 check("best_k", 'check(best_k == 2, "`best_k` should be the number of components (1 to 5) with the lowest BIC.")'),
                 check("gmm", 'check(getattr(gmm, "n_components", None) == best_k and hasattr(gmm, "means_"), "Fit `gmm` with the chosen number of components.")'),
                 check("fake", 'check(getattr(fake, "shape", None) == (500, 1), "`fake` should be the first item returned by gmm.sample(500), shape (500, 1).")'),
             ],
             """
             The incomes come from two groups, so BIC picks 2. `sample` returns
             the draws and their component labels; keep the draws.
             """,
             starter="""
             from sklearn.mixture import GaussianMixture
             best_k = ...
             gmm = ...
             fake = ...
             """,
             setup="import numpy as np\nrng = np.random.default_rng(14)\nincomes = np.concatenate([rng.normal(9.8, 0.3, 400), rng.normal(11.2, 0.4, 200)])", level=3),
    ]),

# ---------------------------------------------------------------- 5.13
concept(
    "py.ml.kde", "Kernel density estimation", ["py.ml.gmm", "py.ml.model_selection"],
    "5.13", "In Depth: Kernel Density Estimation", "05.13-kernel-density-estimation",
    """
    A histogram estimates a density, but it depends on where the bins start.
    **Kernel density estimation** puts a small smooth bump (the kernel) on
    every data point and adds them up. Think of it as a mixture model with one
    component per observation.

    ```python
    from sklearn.neighbors import KernelDensity
    kde = KernelDensity(kernel="gaussian", bandwidth=0.5).fit(x[:, None])
    log_dens = kde.score_samples(grid[:, None])   # log of the density
    dens = np.exp(log_dens)
    ```

    Note that `score_samples` returns the **log** density, so exponentiate it.

    The **bandwidth** is the bump width, the analogue of bin width. Too small
    gives a spiky curve that chases individual points; too large blurs real
    features such as two peaks in an income distribution. Choose it by
    cross-validation: `GridSearchCV` over `bandwidth` uses
    `kde.score`, the total log-likelihood of held-out points.
    """,
    [
        mc("m1", "What does `KernelDensity.score_samples(X)` return?",
           "The log of the estimated density at each row of X", ["The density itself", "The probability of each row's class", "A cluster label per row"],
           "Exponentiate with `np.exp` to get the density.", level=1),
        mc("m2", "A KDE of hourly wages looks like a row of narrow spikes at the observed values. What should you change?",
           "Increase the bandwidth", ["Decrease the bandwidth", "Switch to a linear kernel", "Use fewer data points"],
           "A bandwidth that is too small fits each point separately, the density analogue of overfitting.", level=2),
        mc("m3", "When `GridSearchCV` tunes a `KernelDensity` with no `scoring` argument, what does it maximise?",
           "The log-likelihood of the held-out points under the fitted density",
           ["Classification accuracy", "R-squared", "The number of peaks"],
           "KernelDensity's `score` method returns the total log-likelihood, which the search uses by default.", level=3),
        out("o1", "What does this print?",
            """
            import numpy as np
            from sklearn.neighbors import KernelDensity
            rng = np.random.default_rng(0)
            x = rng.normal(0, 1, 200)
            kde = KernelDensity(bandwidth=0.4).fit(x[:, None])
            grid = np.linspace(-6, 6, 1201)
            dens = np.exp(kde.score_samples(grid[:, None]))
            print(round(dens.sum() * (grid[1] - grid[0]), 3))
            """,
            "A density integrates to 1; the Riemann sum over a wide grid confirms it.", level=2),
        out("o2", "What does this print?",
            """
            import numpy as np
            from sklearn.neighbors import KernelDensity
            x = np.array([0.0, 0.0, 4.0])[:, None]
            kde = KernelDensity(bandwidth=0.5).fit(x)
            d = np.exp(kde.score_samples([[0.0], [4.0], [2.0]]))
            print(d[0] > d[1] > d[2])
            """,
            """
            Two bumps sit at 0 and one at 4, so the density is highest at 0,
            lower at 4, and near zero halfway between.
            """, level=2),
        task("c1",
             """
             `logwage` (1-D, already defined) is a sample of log wages. Use
             `GridSearchCV` with `cv=5` over `KernelDensity` bandwidths
             `np.linspace(0.05, 1.0, 20)`. Store the fitted search in `grid` and
             the chosen bandwidth in `bw`.
             """,
             """
             import numpy as np
             from sklearn.neighbors import KernelDensity
             from sklearn.model_selection import GridSearchCV
             grid = GridSearchCV(KernelDensity(), {"bandwidth": np.linspace(0.05, 1.0, 20)}, cv=5)
             grid.fit(logwage[:, None])
             bw = grid.best_params_["bandwidth"]
             """,
             [
                 check("grid", 'check(hasattr(grid, "best_params_") and len(grid.param_grid.get("bandwidth", [])) == 20, "`grid` should be a fitted GridSearchCV over 20 bandwidth values.")'),
                 check("bw", """
                     import numpy as np
                     from sklearn.neighbors import KernelDensity
                     from sklearn.model_selection import GridSearchCV
                     ref = GridSearchCV(KernelDensity(), {"bandwidth": np.linspace(0.05, 1.0, 20)}, cv=5).fit(logwage[:, None])
                     check(close(bw, ref.best_params_["bandwidth"]), "`bw` should be the bandwidth in `grid.best_params_`. Did you fit on logwage as a single column?")
                 """),
             ],
             "No target is needed: the search scores each bandwidth by held-out log-likelihood.",
             starter="""
             import numpy as np
             from sklearn.neighbors import KernelDensity
             from sklearn.model_selection import GridSearchCV
             grid = ...
             bw = ...
             """,
             setup="import numpy as np\nrng = np.random.default_rng(15)\nlogwage = np.concatenate([rng.normal(2.6, 0.25, 150), rng.normal(3.4, 0.3, 100)])", level=2),
        task("c2",
             """
             `x` (1-D, already defined) has two groups of values. Fit
             `kde = KernelDensity(bandwidth=0.3)` on it, evaluate the density
             (not the log density) on `grid = np.linspace(0, 10, 501)` as `dens`,
             and store the grid value where the density is highest in `mode`.
             """,
             """
             import numpy as np
             from sklearn.neighbors import KernelDensity
             kde = KernelDensity(bandwidth=0.3).fit(x[:, None])
             grid = np.linspace(0, 10, 501)
             dens = np.exp(kde.score_samples(grid[:, None]))
             mode = grid[np.argmax(dens)]
             """,
             [
                 check("dens", """
                     import numpy as np
                     check(getattr(dens, "shape", None) == (501,), "`dens` should have one value per grid point.")
                     check(dens.min() >= 0 and close(dens.sum() * 0.02, 1, rtol=0.02), "`dens` should be the density itself. score_samples returns its log.")
                 """),
                 check("mode", 'check(abs(mode - 3.0) < 0.3, "`mode` should be the grid value with the highest density.")'),
             ],
             "Exponentiate the log density, then `argmax` finds the peak.",
             starter="""
             import numpy as np
             from sklearn.neighbors import KernelDensity
             kde = ...
             dens = ...
             mode = ...
             """,
             setup="import numpy as np\nrng = np.random.default_rng(16)\nx = np.concatenate([rng.normal(3, 0.5, 300), rng.normal(7, 0.8, 120)])", level=2),
    ]),

    ],
}
