# Chapter 3 of the Python Data Science Handbook: Data Manipulation with Pandas.
#
# Every question and note here is original. Each concept links to the book
# section it follows; the book's own text is CC-BY-NC-ND and is never adapted.
# Written for pandas 3.1: copy-on-write is on, text columns get the `str`
# dtype by default, and offsets use the new aliases ("ME", "QE", "h").
# Build with: python3 scripts/python_bank.py build pandas

CHAPTER = {
    "chapter": "pandas",
    "title": "Data Manipulation with Pandas",
    "domain": "python_pandas",
    "concepts": [

# ---------------------------------------------------------------- 3.01
concept(
    "py.pd.series", "The Series object", ["py.np.create", "py.np.indexing"],
    "3.01", "Introducing Pandas Objects", "03.01-introducing-pandas-objects",
    """
    A **Series** is a one-dimensional NumPy array with a label attached to
    every value. The labels form the **index**; the values keep a single
    dtype, just like an array.

    ```python
    import pandas as pd
    wage = pd.Series([18.5, 22.0, 31.0], index=["clerk", "driver", "nurse"])
    wage["driver"]      # 22.0, looked up by label
    wage.values         # the underlying data
    wage.index          # Index(['clerk', 'driver', 'nurse'], dtype='str')
    ```

    You can read a Series two ways. It behaves like an array (slicing,
    vectorised arithmetic, a dtype), and it behaves like a dictionary (look up
    a value by its key, test `"nurse" in wage`). Building one from a dict uses
    the keys as the index. Without an explicit index, pandas numbers the
    entries 0, 1, 2, ...
    """,
    [
        mc("m1", "What does `pd.Series({'Q1': 4.0, 'Q2': 5.5})` use as its index?",
           "The dict keys, `'Q1'` and `'Q2'`",
           ["The integers 0 and 1", "The values 4.0 and 5.5", "No index until you call `set_index`"],
           """
           A dict maps keys to values, and a Series does the same, so the keys
           become the labels.
           """, level=1),
        mc("m2", "How is a Series different from a plain NumPy array?",
           "Each value carries a label, and the labels can be any hashable values, not only 0 to n-1",
           ["A Series can mix dtypes freely while an array cannot",
            "A Series cannot be used in arithmetic",
            "A Series has no dtype"],
           """
           Under the hood the values are still a typed array. The extra piece
           is the explicit index, which lets you look values up by name.
           """, level=1),
        mc("m3", "`s = pd.Series(7, index=['a', 'b', 'c'])`. What is `s`?",
           "Three entries, each equal to 7", ["One entry, 7, labelled 'a'", "An error: the data must be a list", "Three missing values"],
           """
           A scalar is repeated to fill every label in the index you give.
           """, level=2),
        out("o1", "What does this print?",
            """
            import pandas as pd
            rent = pd.Series({"Reno": 1450, "Elko": 980, "Ely": 720})
            print(rent["Elko"], len(rent), "Ely" in rent)
            """,
            """
            Lookup by label, the number of entries, and `in` checks the
            index (the keys), just as it does for a dict.
            """, level=1),
        out("o2", "What does this print?",
            """
            import pandas as pd
            s = pd.Series([3, 1, 4])
            print(s.index.tolist(), s.dtype)
            """,
            """
            Without an index argument pandas uses 0, 1, 2. Three integers give
            an int64 Series.
            """, level=1),
        task("c1",
             """
             Make a Series `cpi` holding the price index values 100.0, 102.1 and
             104.9 labelled by the years 2021, 2022 and 2023. Then set `latest`
             to the 2023 value, looked up by its label.
             """,
             """
             import pandas as pd
             cpi = pd.Series([100.0, 102.1, 104.9], index=[2021, 2022, 2023])
             latest = cpi[2023]
             """,
             [
                 check("cpi", """
                     import pandas as pd
                     check(isinstance(cpi, pd.Series), "`cpi` should be a pandas Series.")
                     check(list(cpi.index) == [2021, 2022, 2023], "The index of `cpi` should be the years 2021, 2022, 2023.")
                     check(close(cpi.values, [100.0, 102.1, 104.9]), "`cpi` should hold 100.0, 102.1 and 104.9.")
                 """),
                 check("latest", 'check(close(latest, 104.9), "`latest` should be the value labelled 2023.")'),
             ],
             """
             Pass the values and an `index=` list of the same length. A dict
             `{2021: 100.0, ...}` works too.
             """,
             starter="""
             import pandas as pd
             cpi = ...
             latest = ...
             """, level=1),
    ]),

concept(
    "py.pd.dataframe", "The DataFrame object", ["py.pd.series"],
    "3.01", "Introducing Pandas Objects", "03.01-introducing-pandas-objects",
    """
    A **DataFrame** is a table: several Series that share one row index, each
    stored as a named column. Columns may have different dtypes.

    ```python
    counties = pd.DataFrame({
        "pop":   [264, 20, 4],
        "wage":  [24.1, 27.3, 21.8],
    }, index=["Washoe", "Elko", "White Pine"])
    counties.shape     # (3, 2)
    counties.columns   # Index(['pop', 'wage'], dtype='str')
    counties["wage"]   # one column, as a Series
    ```

    The usual ways in: a dict of lists or Series (keys become columns), a
    list of dicts (one dict per row; missing keys become NaN), or a 2-D
    NumPy array plus `columns=` and `index=` labels. A dict of Series is
    aligned on the Series' indexes, so rows line up by label, not by position.
    """,
    [
        mc("m1", "`pd.DataFrame([{'a': 1, 'b': 2}, {'a': 3}])` has what in row 1, column `b`?",
           "`NaN`", ["`0`", "`2`", "It raises an error because the dicts differ"],
           """
           Each dict is a row. A key missing from a row is filled with NaN,
           which also makes column `b` floating point.
           """, level=2),
        mc("m2", "For a DataFrame `df`, what does `df['price']` return?",
           "The `price` column, as a Series sharing df's row index",
           ["The row labelled `price`", "A one-column DataFrame", "A NumPy array of prices with no labels"],
           """
           Square brackets with a single column name give that column as a
           Series. Use `df[['price']]` for a one-column DataFrame.
           """, level=1),
        mc("m3", "Which statement about DataFrame columns is true?",
           "Each column has a single dtype, but different columns can have different dtypes",
           ["All columns must share one dtype", "Each cell stores its own dtype", "Columns can only hold numbers"],
           """
           Think of each column as its own Series. That is what lets a table
           hold a text column next to a float column.
           """, level=1),
        out("o1", "What does this print?",
            """
            import numpy as np
            import pandas as pd
            df = pd.DataFrame(np.arange(6).reshape(3, 2), columns=["x", "y"])
            print(df.shape, df["y"].sum())
            """,
            """
            Three rows and two columns. Column `y` holds 1, 3 and 5.
            """, level=1),
        out("o2", "What does this print?",
            """
            import pandas as pd
            a = pd.Series([1, 2], index=["p", "q"])
            b = pd.Series([10, 30], index=["q", "r"])
            df = pd.DataFrame({"a": a, "b": b})
            print(df)
            """,
            """
            The row index is the union p, q, r. Each column is aligned by
            label, so gaps become NaN and both columns turn float.
            """, level=3),
        task("c1",
             """
             Build a DataFrame `firms` with columns `sector` and `employees`
             and the row index `["F1", "F2", "F3"]`. The sectors are
             "retail", "mining" and "retail"; the employee counts are 40, 310
             and 12. Then set `total` to the sum of the `employees` column.
             """,
             """
             import pandas as pd
             firms = pd.DataFrame({"sector": ["retail", "mining", "retail"],
                                   "employees": [40, 310, 12]},
                                  index=["F1", "F2", "F3"])
             total = firms["employees"].sum()
             """,
             [
                 check("shape", """
                     import pandas as pd
                     check(isinstance(firms, pd.DataFrame), "`firms` should be a DataFrame.")
                     check(list(firms.columns) == ["sector", "employees"], "Columns should be sector then employees.")
                     check(list(firms.index) == ["F1", "F2", "F3"], "The row index should be F1, F2, F3.")
                 """),
                 check("values", 'check(list(firms["employees"]) == [40, 310, 12] and list(firms["sector"]) == ["retail", "mining", "retail"], "Check the values in each column.")'),
                 check("total", 'check(total == 362, "`total` should be the sum of the employees column.")'),
             ],
             """
             A dict of lists gives one column per key; `index=` sets the row
             labels. Selecting a column gives a Series, which has `.sum()`.
             """,
             starter="""
             import pandas as pd
             firms = ...
             total = ...
             """, level=1),
    ]),

concept(
    "py.pd.index", "The Index object", ["py.pd.dataframe"],
    "3.01", "Introducing Pandas Objects", "03.01-introducing-pandas-objects",
    """
    The labels of a Series or DataFrame live in an **Index** object. It looks
    like an array (it has `shape`, `dtype`, slicing), with two twists.

    First, it is **immutable**: `idx[0] = "x"` raises a `TypeError`. That
    lets several objects share one index safely.

    Second, it behaves like an **ordered set**, with set operations that keep
    a sensible order:

    ```python
    a = pd.Index(["AL", "NV", "UT"])
    b = pd.Index(["NV", "UT", "WY"])
    a.intersection(b)          # Index(['NV', 'UT'])
    a.union(b)                 # Index(['AL', 'NV', 'UT', 'WY'])
    a.difference(b)            # Index(['AL'])
    a.symmetric_difference(b)  # Index(['AL', 'WY'])
    ```

    These are handy for questions like "which states appear in both
    surveys?" before you join two tables.
    """,
    [
        mc("m1", "`idx = pd.Index([2019, 2020, 2021])`. What happens on `idx[1] = 2025`?",
           "A `TypeError`: an Index cannot be changed in place",
           ["The middle label becomes 2025", "A new Index is returned", "Nothing happens silently"],
           """
           Index objects are immutable. To relabel, build a new index (for
           example with `rename` or by assigning a whole new list to `.index`).
           """, level=1),
        mc("m2", "`a = pd.Index([1, 2, 3, 4])`, `b = pd.Index([3, 4, 5])`. What is `a.difference(b)`?",
           "`Index([1, 2])`", ["`Index([5])`", "`Index([1, 2, 5])`", "`Index([3, 4])`"],
           """
           The difference keeps labels in `a` that are not in `b`.
           """, level=1),
        mc("m3", "Why is it useful that two DataFrames can share the same Index object?",
           "Because an Index cannot be modified, sharing it cannot cause one table to change the other's labels",
           ["Because sharing makes the data columns immutable too",
            "Because pandas requires every DataFrame to have a unique Index object",
            "Because shared indexes sort faster"],
           """
           Immutability is what makes sharing safe: nobody can edit the labels
           out from under another object.
           """, level=2),
        out("o1", "What does this print?",
            """
            import pandas as pd
            a = pd.Index(["AL", "NV", "UT"])
            b = pd.Index(["NV", "UT", "WY"])
            print(a.symmetric_difference(b).tolist())
            """,
            """
            Labels in exactly one of the two: AL (only in a) and WY (only in b).
            """, level=1),
        out("o2", "What does this print?",
            """
            import pandas as pd
            idx = pd.Index([10, 20, 30, 40])
            print(idx[1:3].tolist(), idx.size)
            """,
            """
            An Index slices like an array: positions 1 and 2.
            """, level=1),
        task("c1",
             """
             `survey_a` and `survey_b` are DataFrames indexed by county name.
             Set `both` to the Index of counties present in both surveys, and
             `only_a` to those in `survey_a` but not `survey_b`.
             """,
             """
             both = survey_a.index.intersection(survey_b.index)
             only_a = survey_a.index.difference(survey_b.index)
             """,
             [
                 check("both", 'check(sorted(both) == ["Clark", "Washoe"], "`both` should hold the counties in both indexes.")'),
                 check("only_a", 'check(sorted(only_a) == ["Elko", "Lyon"], "`only_a` should hold counties in survey_a only.")'),
             ],
             """
             Each DataFrame's labels are in `.index`, and Index objects have
             set methods: `intersection`, `union`, `difference`.
             """,
             starter="""
             print(survey_a.index, survey_b.index)
             both = ...
             only_a = ...
             """,
             setup="import pandas as pd\nsurvey_a = pd.DataFrame({'x': range(4)}, index=['Clark', 'Elko', 'Lyon', 'Washoe'])\nsurvey_b = pd.DataFrame({'y': range(3)}, index=['Washoe', 'Nye', 'Clark'])", level=1),
    ]),

# ---------------------------------------------------------------- 3.02
concept(
    "py.pd.series_select", "Selecting from a Series: loc and iloc", ["py.pd.series", "py.np.slicing", "py.np.masks"],
    "3.02", "Data Indexing and Selection", "03.02-data-indexing-and-selection",
    """
    A Series supports dictionary-style lookups (`s["a"]`), array-style
    slices and boolean masks. Because labels can themselves be integers,
    "position 1" and "label 1" can be different things. Two indexers remove
    the guesswork:

    - `s.loc[...]` always means **labels**. Label slices include the end.
    - `s.iloc[...]` always means **integer positions**. Slices exclude the
      end, as in Python.

    ```python
    s = pd.Series([5.0, 6.5, 8.0, 9.5], index=[10, 20, 30, 40])
    s.loc[20]          # 6.5   (label 20)
    s.iloc[1]          # 6.5   (second position)
    s.loc[20:30]       # labels 20 and 30, both included
    s.iloc[1:3]        # positions 1 and 2
    s[s > 7]           # boolean mask: 8.0 and 9.5
    ```

    Get in the habit of writing `loc` and `iloc` explicitly; it makes the
    intent clear to the reader and to pandas.
    """,
    [
        mc("m1", "`s = pd.Series([1, 2, 3, 4], index=['a', 'b', 'c', 'd'])`. What is `s.loc['b':'c']`?",
           "The values 2 and 3", ["Only the value 2", "The values 2, 3 and 4", "An error: labels cannot be sliced"],
           """
           Label slices with `loc` include both endpoints, unlike position
           slices.
           """, level=1),
        mc("m2", "`s = pd.Series([7, 8, 9], index=[3, 2, 1])`. What is `s.iloc[0]`?",
           "7", ["9", "8", "A `KeyError`"],
           """
           `iloc` ignores the labels and counts positions from 0, so it takes
           the first entry, 7. `s.loc[1]` would be 9.
           """, level=2),
        mc("m3", "Which expression keeps only the entries of a Series `wage` above 20?",
           "`wage[wage > 20]`", ["`wage > 20`", "`wage.loc[20:]`", "`wage.iloc[wage > 20]`"],
           """
           `wage > 20` is a boolean Series; using it as an index keeps the
           True rows. `wage.loc[20:]` slices labels, not values.
           """, level=1),
        out("o1", "What does this print?",
            """
            import pandas as pd
            s = pd.Series([5, 6, 7, 8], index=[1, 3, 5, 7])
            print(s.loc[3:5].tolist(), s.iloc[3:5].tolist())
            """,
            """
            `loc[3:5]` takes labels 3 through 5 inclusive. `iloc[3:5]` takes
            positions 3 and 4, but there is only position 3.
            """, level=2),
        out("o2", "What does this print?",
            """
            import pandas as pd
            price = pd.Series([2.5, 4.0, 1.25], index=["eggs", "milk", "salt"])
            price.loc["salt"] = 1.5
            print(price[price < 3].index.tolist())
            """,
            """
            The `loc` assignment updates salt to 1.5. The mask then keeps eggs
            and salt.
            """, level=1),
        task("c1",
             """
             `rate` is a Series of unemployment rates indexed by year (integers
             2015 to 2024). Set `mid` to the rates for 2018 through 2020
             inclusive (by label), `first3` to the first three entries (by
             position), and `high` to the entries above 5.0.
             """,
             """
             mid = rate.loc[2018:2020]
             first3 = rate.iloc[:3]
             high = rate[rate > 5.0]
             """,
             [
                 check("mid", 'check(list(mid.index) == [2018, 2019, 2020], "`mid` should hold the labels 2018 to 2020, both included. Use loc.")'),
                 check("first3", 'check(list(first3.index) == [2015, 2016, 2017], "`first3` should be the first three entries by position. Use iloc.")'),
                 check("high", 'check(list(high.index) == [2020, 2021], "`high` should keep the entries whose value is above 5.0.")'),
             ],
             """
             `loc` slices by label and includes the end; `iloc` slices by
             position and excludes it; a boolean Series filters by value.
             """,
             starter="""
             print(rate)
             mid = rate[2018:2020]
             first3 = ...
             high = ...
             """,
             setup="import pandas as pd\nrate = pd.Series([5.0, 4.7, 4.1, 3.9, 3.7, 8.1, 5.4, 3.6, 3.6, 4.0], index=range(2015, 2025))"),
    ]),

concept(
    "py.pd.frame_select", "Selecting from a DataFrame", ["py.pd.series_select", "py.pd.dataframe"],
    "3.02", "Data Indexing and Selection", "03.02-data-indexing-and-selection",
    """
    A DataFrame is dictionary-like over its **columns** and array-like over
    its **rows**. That gives a few conventions worth memorising:

    ```python
    df["wage"]                  # one column (Series)
    df[["wage", "hours"]]       # several columns (DataFrame)
    df["pay"] = df.wage * df.hours   # add a column
    df.loc["Elko", "wage"]      # one cell by labels
    df.loc[df.hours > 40, ["wage"]]  # mask on rows, list of columns
    df.iloc[:2, 1:]             # first two rows, columns from position 1
    df[1:3]                     # slicing in [] slices ROWS by position
    df[df.wage > 25]            # a mask in [] filters ROWS
    ```

    With copy-on-write in pandas 3, chained assignment such as
    `df["wage"]["Elko"] = 30` never changes `df`. Do it in one step:
    `df.loc["Elko", "wage"] = 30`.
    """,
    [
        mc("m1", "For a DataFrame `df`, what does `df[0:2]` select?",
           "The first two rows", ["The first two columns", "The column labelled 0 and 1", "A `KeyError`"],
           """
           A plain name in `[]` picks a column, but a slice in `[]` slices
           rows. It is a convenience convention worth knowing, though
           `df.iloc[0:2]` says the same thing more clearly.
           """, level=2),
        mc("m2", "With pandas 3 (copy-on-write on), what does `df['wage'][0] = 99` do to `df`?",
           "Nothing: the assignment lands on a temporary copy, and pandas warns about chained assignment",
           ["It sets the first wage in `df` to 99",
            "It raises a `KeyError`",
            "It sets every wage to 99"],
           """
           Under copy-on-write, `df['wage']` behaves as a copy, so the second
           assignment never reaches `df`. Use `df.loc[row, 'wage'] = 99`.
           """, level=2),
        mc("m3", "Which expression gives the `wage` and `hours` columns for rows where `hours` is over 40?",
           "`df.loc[df['hours'] > 40, ['wage', 'hours']]`",
           ["`df.iloc[df['hours'] > 40, ['wage', 'hours']]`",
            "`df[df['hours'] > 40, ['wage', 'hours']]`",
            "`df.loc[['wage', 'hours'], df['hours'] > 40]`"],
           """
           `loc[rows, columns]`: the mask selects rows and the list selects
           columns. `iloc` wants positions, and plain `[]` takes one key.
           """, level=2),
        out("o1", "What does this print?",
            """
            import pandas as pd
            df = pd.DataFrame({"wage": [21.0, 30.5, 18.0], "hours": [38, 45, 20]},
                              index=["Ana", "Ben", "Cy"])
            print(df.loc["Ben", "hours"], df.iloc[2, 0])
            """,
            """
            `loc` uses the labels Ben and hours. `iloc[2, 0]` is the third row
            and first column: Cy's wage.
            """, level=1),
        out("o2", "What does this print?",
            """
            import pandas as pd
            df = pd.DataFrame({"wage": [21.0, 30.5, 18.0], "hours": [38, 45, 20]},
                              index=["Ana", "Ben", "Cy"])
            df["pay"] = df["wage"] * df["hours"]
            print(df[df["pay"] > 500].index.tolist())
            """,
            """
            Pay is 798, 1372.5 and 360. A mask inside `[]` filters rows.
            """, level=2),
        task("c1",
             """
             `jobs` has columns `wage` and `hours`, indexed by worker id. Add a
             column `weekly` equal to wage times hours. Then set `full_time` to
             the `wage` and `weekly` columns for workers with at least 35
             hours, and raise the wage of worker `"w3"` to 20.0 in `jobs`
             itself.
             """,
             """
             jobs["weekly"] = jobs["wage"] * jobs["hours"]
             full_time = jobs.loc[jobs["hours"] >= 35, ["wage", "weekly"]]
             jobs.loc["w3", "wage"] = 20.0
             """,
             [
                 check("weekly", 'check("weekly" in jobs.columns and close(jobs.loc["w1", "weekly"], 720.0), "Add a `weekly` column equal to wage * hours.")'),
                 check("full_time", 'check(list(full_time.index) == ["w1", "w4"] and list(full_time.columns) == ["wage", "weekly"], "`full_time` should keep rows with hours >= 35 and only the wage and weekly columns.")'),
                 check("w3", 'check(close(jobs.loc["w3", "wage"], 20.0), "Worker w3 should have wage 20.0 in `jobs`. Chained assignment does not modify the frame; use loc[row, col].")'),
             ],
             """
             Assigning to a new column name adds it. `loc[mask, list]` picks
             rows and columns together, and `loc[row, col] = value` writes in
             one step, which is the form that works under copy-on-write.
             """,
             starter="""
             jobs["weekly"] = ...
             full_time = ...
             jobs["wage"]["w3"] = 20.0
             """,
             setup="import pandas as pd\njobs = pd.DataFrame({'wage': [18.0, 25.0, 16.5, 30.0], 'hours': [40, 20, 30, 38]}, index=['w1', 'w2', 'w3', 'w4'])"),
    ]),

# ---------------------------------------------------------------- 3.03
concept(
    "py.pd.alignment", "Operations and index alignment", ["py.pd.frame_select", "py.np.ufuncs", "py.np.broadcast_rules"],
    "3.03", "Operating on Data in Pandas", "03.03-operations-in-pandas",
    """
    NumPy ufuncs work on pandas objects and keep the labels: `np.log(s)`
    returns a Series with the same index.

    Binary operations go further: pandas **aligns on labels** before
    computing. Labels present in only one operand produce NaN.

    ```python
    q1 = pd.Series({"NV": 120, "UT": 90})
    q2 = pd.Series({"UT": 95, "AZ": 200})
    q1 + q2                     # AZ NaN, NV NaN, UT 185.0
    q1.add(q2, fill_value=0)    # AZ 200.0, NV 120.0, UT 185.0
    ```

    DataFrames align on both rows and columns. Between a DataFrame and a
    Series, the Series lines up with the **columns** by default (like NumPy
    broadcasting a row). To line it up with the rows, use a method with
    `axis=0`: `df.sub(df["base"], axis=0)`.
    """,
    [
        mc("m1", "`a = pd.Series([1, 2], index=['x', 'y'])`, `b = pd.Series([10, 20], index=['y', 'z'])`. What is `(a + b)['y']`?",
           "12.0", ["21", "NaN", "11"],
           """
           Values are matched by label: a's y (2) plus b's y (10). The result
           is float because x and z become NaN.
           """, level=1),
        mc("m2", "What does `a.add(b, fill_value=0)` change compared with `a + b`?",
           "A label missing from one side is treated as 0 there instead of producing NaN",
           ["It fills NaN in the result with 0 after adding", "It drops labels that are not in both", "Nothing; it is an alias"],
           """
           `fill_value` replaces the missing operand before adding. So a label
           only in `b` keeps b's value.
           """, level=2),
        mc("m3", "`df` has columns `['a', 'b']` and 3 rows. What does `df - df.iloc[0]` do?",
           "Subtracts the first row from every row, matching by column",
           ["Subtracts the first column from every column", "Raises an error because shapes differ", "Subtracts only from the first row"],
           """
           `df.iloc[0]` is a Series indexed by column names, so it aligns with
           the columns and is applied to each row.
           """, level=2),
        out("o1", "What does this print?",
            """
            import pandas as pd
            q1 = pd.Series({"NV": 120, "UT": 90})
            q2 = pd.Series({"UT": 95, "AZ": 200})
            print((q1 + q2).tolist())
            """,
            """
            The result index is the sorted union AZ, NV, UT. Only UT is in
            both, so the other two are NaN.
            """, level=2),
        out("o2", "What does this print?",
            """
            import numpy as np
            import pandas as pd
            s = pd.Series([1.0, 100.0], index=["lo", "hi"])
            print(np.log10(s))
            """,
            """
            A ufunc applied to a Series keeps the index and order.
            """, level=1),
        task("c1",
             """
             `sales` has one row per store and columns `base`, `q1`, `q2`.
             Make `growth`, a DataFrame with columns `q1` and `q2` holding each
             quarter's sales minus that store's `base` value.
             """,
             """
             growth = sales[["q1", "q2"]].sub(sales["base"], axis=0)
             """,
             [
                 check("shape", 'check(getattr(growth, "shape", None) == (3, 2) and list(growth.columns) == ["q1", "q2"], "`growth` should have the 3 stores and columns q1 and q2.")'),
                 check("values", 'check(close(growth.values, [[5, 15], [-10, 0], [2, 4]]), "Each quarter should have that row\'s base subtracted. A Series lines up with the rows only with axis=0.")'),
             ],
             """
             `df - series` lines the Series up with the columns, which would
             produce NaN here. `.sub(series, axis=0)` matches it to the rows.
             """,
             starter="""
             growth = sales[["q1", "q2"]] - sales["base"]
             """,
             setup="import pandas as pd\nsales = pd.DataFrame({'base': [100, 80, 50], 'q1': [105, 70, 52], 'q2': [115, 80, 54]}, index=['s1', 's2', 's3'])"),
    ]),

# ---------------------------------------------------------------- 3.04
concept(
    "py.pd.missing", "Handling missing data", ["py.pd.alignment", "py.np.aggregates"],
    "3.04", "Handling Missing Data", "03.04-missing-values",
    """
    Pandas marks missing values with **sentinels**. In float columns the
    sentinel is `NaN`, and `None` is converted to it, so an integer column
    with a gap becomes float64. The nullable dtypes (`"Int64"`, `"Float64"`,
    `"boolean"`, note the capital letters) use a separate mask and print the
    gap as `<NA>`, keeping integers as integers.

    ```python
    s = pd.Series([3, None, 5])           # float64: 3.0, NaN, 5.0
    pd.Series([3, None, 5], dtype="Int64")  # 3, <NA>, 5
    s.isna()          # True where missing (alias: isnull)
    s.notna()         # the opposite
    s.dropna()        # drop missing entries
    s.fillna(0)       # replace them
    s.ffill()         # carry the last value forward
    ```

    On DataFrames, `dropna()` drops any row with a gap; `how="all"` drops
    only rows that are entirely missing, `axis=1` drops columns, and
    `thresh=k` keeps rows with at least k values. Aggregates like `mean`
    skip missing values by default.
    """,
    [
        mc("m1", "What is the dtype of `pd.Series([4, None, 6])`?",
           "`float64`", ["`int64`", "`object`", "`Int64`"],
           """
           `None` becomes NaN, a float sentinel, so the integers are upcast to
           float64. Ask for `dtype="Int64"` to keep them as integers.
           """, level=1),
        mc("m2", "`s = pd.Series([1.0, None, 3.0])`. What is `s.mean()`?",
           "2.0", ["NaN", "1.333...", "An error"],
           """
           Pandas aggregates skip missing values by default (`skipna=True`),
           so the mean is of 1 and 3.
           """, level=1),
        mc("m3", "Which call drops only the rows of `df` in which every value is missing?",
           "`df.dropna(how='all')`", ["`df.dropna()`", "`df.dropna(axis=1)`", "`df.fillna(how='all')`"],
           """
           The default `how='any'` drops a row with even one gap. `axis=1`
           works on columns instead.
           """, level=2),
        mc("m4", "In pandas 3, how do you carry the previous value forward into gaps?",
           "`s.ffill()`", ["`s.fillna(method='ffill')`", "`s.fillna(forward=True)`", "`s.dropna(method='ffill')`"],
           """
           The `method=` argument of `fillna` was removed in pandas 3; the
           dedicated methods `ffill` and `bfill` replace it.
           """, level=2),
        out("o1", "What does this print?",
            """
            import pandas as pd
            s = pd.Series([2, None, 7, None])
            print(s.isna().sum(), s.fillna(0).tolist())
            """,
            """
            Two entries are missing. `fillna(0)` replaces them, and the Series
            stays float because it was float to begin with.
            """, level=1),
        out("o2", "What does this print?",
            """
            import pandas as pd
            s = pd.Series([2, None, 7], dtype="Int64")
            print(s.tolist(), s.dtype)
            """,
            """
            The nullable integer dtype keeps whole numbers and stores the gap
            as `pd.NA`.
            """, level=2),
        out("o3", "What does this print?",
            """
            import numpy as np
            import pandas as pd
            df = pd.DataFrame({"a": [1, np.nan, np.nan], "b": [2, 5, np.nan]})
            print(len(df.dropna()), len(df.dropna(how="all")), len(df.dropna(thresh=1)))
            """,
            """
            Only row 0 is complete. Row 2 is entirely missing, so `how='all'`
            drops just that one, and `thresh=1` keeps rows with at least one
            value: the same two.
            """, level=3),
        task("c1",
             """
             `prices` is a daily Series with some missing quotes. Make
             `filled`, the series with each gap replaced by the most recent
             earlier quote, and `n_gaps`, the number of missing quotes in the
             original.
             """,
             """
             filled = prices.ffill()
             n_gaps = prices.isna().sum()
             """,
             [
                 check("filled", 'check(close(filled.values, [10.0, 10.0, 10.5, 10.5, 10.5, 11.0]), "`filled` should carry the last known price forward into each gap.")'),
                 check("n_gaps", 'check(int(n_gaps) == 3, "`n_gaps` should count the missing values in `prices`.")'),
                 check("original", 'check(int(prices.isna().sum()) == 3, "Leave `prices` itself unchanged.")'),
             ],
             """
             `ffill` returns a new Series. `isna()` gives a boolean Series and
             summing it counts the Trues.
             """,
             starter="""
             filled = prices.fillna(0)
             n_gaps = ...
             """,
             setup="import numpy as np\nimport pandas as pd\nprices = pd.Series([10.0, np.nan, 10.5, np.nan, np.nan, 11.0])"),
    ]),

# ---------------------------------------------------------------- 3.05
concept(
    "py.pd.multiindex", "Hierarchical indexing with MultiIndex", ["py.pd.frame_select"],
    "3.05", "Hierarchical Indexing", "03.05-hierarchical-indexing",
    """
    A **MultiIndex** gives each row a tuple of labels, so a single Series can
    hold panel data such as (state, year) without a 3-D structure.

    ```python
    idx = pd.MultiIndex.from_product([["NV", "UT"], [2022, 2023]],
                                     names=["state", "year"])
    gdp = pd.Series([210, 225, 230, 244], index=idx)
    gdp["NV"]               # the NV rows, indexed by year
    gdp.loc["UT", 2023]     # one value: 244
    gdp.loc[:, 2023]        # every state in 2023
    gdp.xs(2023, level="year")   # the same, naming the level
    ```

    Other constructors: `from_tuples`, `from_arrays`, or passing a list of
    arrays as `index=`. Name the levels (`names=`) so later code can refer to
    them by name. Slicing a range of labels, as in `gdp.loc["NV":"UT"]`,
    needs the index to be sorted; call `sort_index()` first.
    """,
    [
        mc("m1", "How many entries does `pd.MultiIndex.from_product([['a', 'b', 'c'], [1, 2]])` have?",
           "6", ["5", "2", "3"],
           """
           `from_product` takes every combination: 3 x 2 = 6 pairs.
           """, level=1),
        mc("m2", "`gdp` has a (state, year) MultiIndex. What does `gdp.loc[:, 2023]` return?",
           "Every state's value for 2023, indexed by state",
           ["Every year's value for the first state", "Only the first row", "An error: `:` is not allowed in loc"],
           """
           The first slot `:` keeps all states; the second picks year 2023.
           The year level is dropped from the result.
           """, level=2),
        mc("m3", "Why give a MultiIndex level names such as `names=['state', 'year']`?",
           "So methods like `xs`, `groupby` and `unstack` can refer to a level by name instead of by position",
           ["Names are required or the index cannot be created", "Named levels make lookups faster", "So the levels are sorted automatically"],
           """
           Names make code readable and robust to reordering levels. They are
           optional.
           """, level=1),
        out("o1", "What does this print?",
            """
            import pandas as pd
            idx = pd.MultiIndex.from_product([["NV", "UT"], [2022, 2023]], names=["state", "year"])
            gdp = pd.Series([210, 225, 230, 244], index=idx)
            print(gdp.loc["UT", 2022], gdp["NV"].sum())
            """,
            """
            UT in 2022 is the third entry, 230. The NV rows are 210 and 225.
            """, level=1),
        out("o2", "What does this print?",
            """
            import pandas as pd
            idx = pd.MultiIndex.from_product([["NV", "UT"], [2022, 2023]], names=["state", "year"])
            gdp = pd.Series([210, 225, 230, 244], index=idx)
            print(gdp.xs(2023, level="year").tolist())
            """,
            """
            A cross-section at year 2023 takes one value per state.
            """, level=2),
        task("c1",
             """
             Build `emp`, a Series of employment counts with a MultiIndex whose
             levels are named `county` and `quarter`. The counties are "Elko"
             and "Lyon"; the quarters are "Q1" and "Q2"; the values in order
             (Elko Q1, Elko Q2, Lyon Q1, Lyon Q2) are 510, 530, 300, 290. Then
             set `q2` to the Q2 value for every county.
             """,
             """
             import pandas as pd
             idx = pd.MultiIndex.from_product([["Elko", "Lyon"], ["Q1", "Q2"]], names=["county", "quarter"])
             emp = pd.Series([510, 530, 300, 290], index=idx)
             q2 = emp.xs("Q2", level="quarter")
             """,
             [
                 check("index", """
                     import pandas as pd
                     check(isinstance(getattr(emp, "index", None), pd.MultiIndex), "`emp` should have a MultiIndex.")
                     check(list(emp.index.names) == ["county", "quarter"], "Name the levels county and quarter.")
                 """),
                 check("values", 'check(emp.loc[("Lyon", "Q1")] == 300 and emp.loc[("Elko", "Q2")] == 530, "Check that the values follow the order Elko Q1, Elko Q2, Lyon Q1, Lyon Q2.")'),
                 check("q2", 'check(list(q2.index) == ["Elko", "Lyon"] and list(q2) == [530, 290], "`q2` should hold the Q2 value for each county, indexed by county.")'),
             ],
             """
             `from_product` builds every (county, quarter) pair in order.
             `xs(..., level=...)` or `emp.loc[:, "Q2"]` takes the cross-section.
             """,
             starter="""
             import pandas as pd
             idx = ...
             emp = ...
             q2 = ...
             """),
    ]),

concept(
    "py.pd.reshape_index", "Stacking, unstacking and resetting the index", ["py.pd.multiindex"],
    "3.05", "Hierarchical Indexing", "03.05-hierarchical-indexing",
    """
    A table can be stored **long** (one row per state-year, a MultiIndex) or
    **wide** (states as rows, years as columns). Two methods move between
    them:

    - `unstack()` moves the innermost index level into the columns (long to
      wide). Pass a level name to choose which one.
    - `stack()` moves the columns into the innermost index level (wide to
      long).

    Columns and index can also trade places:

    ```python
    long = df.set_index(["state", "year"])   # columns become the index
    flat = long.reset_index()                # index levels become columns
    wide = long["gdp"].unstack("year")       # one column per year
    ```

    Data files usually arrive flat, so `set_index` then `unstack` is the
    common route to a wide table; `reset_index` is the way back.
    """,
    [
        mc("m1", "`s` has a (state, year) MultiIndex with 3 states and 4 years. What is the shape of `s.unstack()`?",
           "`(3, 4)`", ["`(4, 3)`", "`(12, 1)`", "`(12,)`"],
           """
           `unstack` moves the innermost level (year) into the columns,
           leaving one row per state.
           """, level=2),
        mc("m2", "What does `df.reset_index()` do to a DataFrame indexed by (state, year)?",
           "Turns `state` and `year` into ordinary columns and gives a fresh 0, 1, 2, ... index",
           ["Deletes the index labels entirely", "Sorts the rows by the index", "Moves the year level into the columns"],
           """
           `reset_index` is the inverse of `set_index`: the levels come back
           as columns.
           """, level=1),
        mc("m3", "Which pair of calls round-trips a Series with a two-level MultiIndex back to the same shape?",
           "`s.unstack().stack()`", ["`s.unstack().unstack()`", "`s.stack().reset_index()`", "`s.set_index().unstack()`"],
           """
           `unstack` moves a level to the columns; `stack` moves it back.
           """, level=2),
        out("o1", "What does this print?",
            """
            import pandas as pd
            df = pd.DataFrame({"state": ["NV", "NV", "UT", "UT"],
                               "year": [2022, 2023, 2022, 2023],
                               "gdp": [210, 225, 230, 244]})
            wide = df.set_index(["state", "year"])["gdp"].unstack()
            print(wide)
            """,
            """
            One row per state and one column per year. The column axis is
            named after the year level.
            """, level=2),
        out("o2", "What does this print?",
            """
            import pandas as pd
            df = pd.DataFrame({"state": ["NV", "UT"], "gdp": [210, 230]}).set_index("state")
            print(df.reset_index().columns.tolist())
            """,
            """
            The index named state comes back as the first column.
            """, level=1),
        task("c1",
             """
             `panel` is a flat DataFrame with columns `county`, `year` and
             `wage`. Make `wide`, a DataFrame with one row per county, one
             column per year, and wages as values. Then make `long`, a flat
             DataFrame with columns `county`, `year` and `wage` again, by
             reversing the steps.
             """,
             """
             wide = panel.set_index(["county", "year"])["wage"].unstack("year")
             long = wide.stack().rename("wage").reset_index()
             """,
             [
                 check("wide", 'check(getattr(wide, "shape", None) == (2, 3) and list(wide.columns) == [2021, 2022, 2023] and list(wide.index) == ["Elko", "Nye"], "`wide` should have counties as rows and years 2021-2023 as columns.")'),
                 check("wide values", 'check(close(wide.loc["Nye", 2022], 23.5), "Check the values in `wide`: Nye in 2022 should be 23.5.")'),
                 check("long", 'check(sorted(long.columns) == ["county", "wage", "year"] and len(long) == 6, "`long` should be flat, with 6 rows and columns county, year and wage.")'),
             ],
             """
             `set_index` builds the (county, year) MultiIndex, `unstack("year")`
             spreads the years across columns, and `stack` plus `reset_index`
             undo it. `rename` names the stacked Series so its column is
             called wage.
             """,
             starter="""
             wide = ...
             long = ...
             """,
             setup="import pandas as pd\npanel = pd.DataFrame({'county': ['Elko'] * 3 + ['Nye'] * 3, 'year': [2021, 2022, 2023] * 2, 'wage': [24.0, 25.1, 26.0, 22.0, 23.5, 24.2]})"),
    ]),

# ---------------------------------------------------------------- 3.06
concept(
    "py.pd.concat", "Concatenating with pd.concat", ["py.pd.index", "py.np.concat_split"],
    "3.06", "Combining Datasets: Concat and Append", "03.06-concat-and-append",
    """
    `pd.concat` stacks Series or DataFrames along an axis, like
    `np.concatenate` but label-aware.

    ```python
    q1 = pd.DataFrame({"sales": [5, 7]}, index=["a", "b"])
    q2 = pd.DataFrame({"sales": [6, 9]}, index=["a", "b"])
    pd.concat([q1, q2])                      # 4 rows; index a, b, a, b
    pd.concat([q1, q2], ignore_index=True)   # index 0..3
    pd.concat([q1, q2], keys=["Q1", "Q2"])   # MultiIndex (quarter, store)
    pd.concat([q1, q2], axis=1)              # side by side
    ```

    Pandas keeps duplicate labels by default. Pass `verify_integrity=True`
    to raise an error instead, `ignore_index=True` to renumber, or `keys=`
    to record where each piece came from. When the pieces have different
    columns, the default `join="outer"` keeps all of them (gaps become NaN);
    `join="inner"` keeps only shared columns.

    `DataFrame.append` was removed in pandas 2. Use `pd.concat`.
    """,
    [
        mc("m1", "`a` and `b` each have index `[0, 1]`. What is the index of `pd.concat([a, b])`?",
           "`[0, 1, 0, 1]`", ["`[0, 1, 2, 3]`", "`[0, 1]`", "It raises an error because labels repeat"],
           """
           concat keeps the original labels, duplicates included. Use
           `ignore_index=True` for 0 to 3.
           """, level=1),
        mc("m2", "`x` has columns `['a', 'b']` and `y` has columns `['b', 'c']`. What columns does `pd.concat([x, y], join='inner')` have?",
           "`['b']`", ["`['a', 'b', 'c']`", "`['a', 'c']`", "It raises an error"],
           """
           An inner join keeps only the column labels that every piece shares.
           """, level=2),
        mc("m3", "How do you add new rows to a DataFrame in pandas 3?",
           "`pd.concat([df, new_rows])`", ["`df.append(new_rows)`", "`df.extend(new_rows)`", "`df + new_rows`"],
           """
           `DataFrame.append` no longer exists. `df + new_rows` would align and
           add values, not stack rows.
           """, level=1),
        out("o1", "What does this print?",
            """
            import pandas as pd
            q1 = pd.Series([5, 7], index=["a", "b"])
            q2 = pd.Series([6, 9], index=["a", "b"])
            both = pd.concat([q1, q2], keys=["Q1", "Q2"])
            print(both.loc["Q2", "a"], len(both))
            """,
            """
            `keys` adds an outer level naming the source piece.
            """, level=2),
        out("o2", "What does this print?",
            """
            import pandas as pd
            x = pd.DataFrame({"a": [1], "b": [2]})
            y = pd.DataFrame({"b": [3], "c": [4]})
            print(pd.concat([x, y], ignore_index=True))
            """,
            """
            The outer join keeps every column; cells a piece did not have are
            NaN, which turns those columns float.
            """, level=3),
        task("c1",
             """
             `jan`, `feb` and `mar` are DataFrames of transactions with the
             same columns, each indexed from 0. Make `quarter`, one DataFrame
             with all the rows in month order and a fresh index 0, 1, 2, ...,
             and `tagged`, the same rows with an outer index level holding
             "jan", "feb" or "mar".
             """,
             """
             import pandas as pd
             quarter = pd.concat([jan, feb, mar], ignore_index=True)
             tagged = pd.concat([jan, feb, mar], keys=["jan", "feb", "mar"])
             """,
             [
                 check("quarter", 'check(getattr(quarter, "shape", None) == (6, 2) and list(quarter.index) == list(range(6)), "`quarter` should have 6 rows and a fresh index 0..5.")'),
                 check("order", 'check(list(quarter["amount"]) == [10, 20, 30, 40, 50, 60], "Keep the rows in month order: jan, feb, mar.")'),
                 check("tagged", 'check(len(tagged) == 6 and int(tagged.loc[("feb", 0), "amount"]) == 30, "`tagged` should have an outer level jan/feb/mar from keys=.")'),
             ],
             """
             `ignore_index=True` renumbers; `keys=` records the source of each
             block as an outer index level.
             """,
             starter="""
             import pandas as pd
             quarter = pd.concat([jan, feb, mar])
             tagged = ...
             """,
             setup="import pandas as pd\njan = pd.DataFrame({'id': [1, 2], 'amount': [10, 20]})\nfeb = pd.DataFrame({'id': [3, 4], 'amount': [30, 40]})\nmar = pd.DataFrame({'id': [5, 6], 'amount': [50, 60]})"),
    ]),

# ---------------------------------------------------------------- 3.07
concept(
    "py.pd.merge", "Merging and joining tables", ["py.pd.concat", "py.pd.frame_select"],
    "3.07", "Combining Datasets: Merge and Join", "03.07-merge-and-join",
    """
    `pd.merge` combines tables by matching **key values**, the way a
    database join does.

    ```python
    workers = pd.DataFrame({"id": [1, 2, 3], "county": ["Elko", "Nye", "Elko"]})
    rates = pd.DataFrame({"county": ["Elko", "Nye"], "min_wage": [12.0, 11.5]})
    pd.merge(workers, rates, on="county")   # many-to-one: each worker gets a rate
    ```

    The shape of the result depends on the keys: **one-to-one** keeps the
    row count, **many-to-one** repeats the "one" side's rows, and
    **many-to-many** produces every matching pair.

    Options to know:

    - `on="key"`, or `left_on=` / `right_on=` when the names differ, or
      `left_index=True` / `right_index=True` to match on the index.
    - `how="inner"` (default, keys in both), `"left"`, `"right"`, `"outer"`.
    - `suffixes=("_x", "_y")` renames clashing non-key columns.
    """,
    [
        mc("m1", "A left table has 5 workers across 2 counties; the right table has one row per county. How many rows does an inner merge on county give (all counties match)?",
           "5", ["2", "10", "7"],
           """
           Many-to-one: each worker row picks up its county's row, so the
           worker count is kept.
           """, level=2),
        mc("m2", "Left has key `k` values `[1, 1]`; right has `[1, 1, 1]`. How many rows does `pd.merge(left, right, on='k')` have?",
           "6", ["2", "3", "5"],
           """
           Many-to-many joins pair every matching row: 2 x 3.
           """, level=2),
        mc("m3", "The key is `fips` in one table and `county_code` in the other. Which call joins them?",
           "`pd.merge(a, b, left_on='fips', right_on='county_code')`",
           ["`pd.merge(a, b, on=['fips', 'county_code'])`", "`pd.merge(a, b, how='fips')`", "`pd.concat([a, b], on='fips')`"],
           """
           `left_on` and `right_on` name the key in each table. Both columns
           are kept in the result.
           """, level=1),
        mc("m4", "Which `how=` keeps every row of the left table, filling NaN where the right has no match?",
           "`'left'`", ["`'inner'`", "`'right'`", "`'cross'`"],
           """
           A left join keeps all left keys. `inner` would drop unmatched ones.
           """, level=1),
        out("o1", "What does this print?",
            """
            import pandas as pd
            workers = pd.DataFrame({"id": [1, 2, 3], "county": ["Elko", "Nye", "Lyon"]})
            rates = pd.DataFrame({"county": ["Elko", "Nye"], "min_wage": [12.0, 11.5]})
            m = pd.merge(workers, rates, on="county", how="left")
            print(m["min_wage"].tolist())
            """,
            """
            Lyon has no rate, so the left join gives it NaN.
            """, level=2),
        out("o2", "What does this print?",
            """
            import pandas as pd
            a = pd.DataFrame({"k": [1, 2], "v": [10, 20]})
            b = pd.DataFrame({"k": [1, 2], "v": [5, 6]})
            print(pd.merge(a, b, on="k", suffixes=("_old", "_new")).columns.tolist())
            """,
            """
            Both tables have a non-key column `v`, so the suffixes tell the
            two apart.
            """, level=1),
        task("c1",
             """
             `hh` lists households with columns `hh_id` and `fips`. `cty` lists
             counties with columns `county_code` and `median_rent`. Make
             `joined`, every household with its county's median rent (keep
             households whose county is missing from `cty`, with NaN rent).
             """,
             """
             import pandas as pd
             joined = pd.merge(hh, cty, left_on="fips", right_on="county_code", how="left")
             """,
             [
                 check("rows", 'check(len(joined) == 4 and list(joined["hh_id"]) == [101, 102, 103, 104], "`joined` should keep all 4 households in their original order.")'),
                 check("rent", """
                     import numpy as np
                     r = joined["median_rent"].to_numpy(dtype=float)
                     check(close(r[:3], [1200, 900, 1200]), "Each household should get its own county's median_rent.")
                     check(np.isnan(r[3]), "The household with no matching county should have NaN rent: use a left join.")
                 """),
             ],
             """
             The key names differ, so use `left_on` and `right_on`.
             `how="left"` keeps unmatched households.
             """,
             starter="""
             import pandas as pd
             joined = pd.merge(hh, cty, left_on="fips", right_on="county_code")
             """,
             setup="import pandas as pd\nhh = pd.DataFrame({'hh_id': [101, 102, 103, 104], 'fips': [7, 9, 7, 99]})\ncty = pd.DataFrame({'county_code': [7, 9], 'median_rent': [1200, 900]})"),
    ]),

# ---------------------------------------------------------------- 3.08
concept(
    "py.pd.groupby", "GroupBy: split, apply, combine", ["py.pd.frame_select", "py.np.aggregates"],
    "3.08", "Aggregation and Grouping", "03.08-aggregation-and-grouping",
    """
    Series and DataFrames have the usual aggregates (`sum`, `mean`,
    `median`, `min`, `max`, `count`, `std`), and `describe()` gives several
    at once.

    `groupby` runs an aggregate per group in three steps: **split** the rows
    by a key, **apply** a function to each piece, **combine** the results.

    ```python
    wages = pd.DataFrame({
        "sector": ["retail", "mining", "retail", "mining"],
        "wage":   [16.0, 34.0, 18.0, 30.0],
        "hours":  [30, 45, 34, 40]})
    wages.groupby("sector")["wage"].mean()   # mining 32.0, retail 17.0
    wages.groupby("sector").size()            # rows per group
    wages.groupby(["sector", "hours"]).sum()  # several keys
    ```

    The result is indexed by the group keys, in sorted order. The key can be
    a column name, a list of names, an array or Series of the same length,
    a dict that maps index labels to groups, or a function of the index.
    """,
    [
        mc("m1", "`df.groupby('state')['income'].median()` returns",
           "A Series with one median income per state, indexed by state",
           ["One overall median", "A DataFrame with the same number of rows as df", "The state with the highest median"],
           """
           Split by state, take the median of each piece, combine into a
           Series keyed by state.
           """, level=1),
        mc("m2", "What is the difference between `g.size()` and `g.count()` on a grouped DataFrame?",
           "`size` counts rows per group; `count` counts non-missing values per column per group",
           ["They are identical", "`count` counts rows; `size` counts bytes", "`size` skips missing values; `count` does not"],
           """
           `size` ignores the values entirely. `count` excludes NaN, so it can
           differ between columns.
           """, level=2),
        mc("m3", "Which expression gives the mean wage by sector **and** year?",
           "`df.groupby(['sector', 'year'])['wage'].mean()`",
           ["`df.groupby('sector')['wage'].mean('year')`", "`df.groupby('sector', 'year')['wage'].mean()`", "`df['wage'].groupby().mean(['sector', 'year'])`"],
           """
           Pass a list of keys. The result has a two-level MultiIndex.
           """, level=1),
        out("o1", "What does this print?",
            """
            import pandas as pd
            df = pd.DataFrame({"sector": ["retail", "mining", "retail", "mining"],
                               "wage": [16.0, 34.0, 18.0, 30.0]})
            print(df.groupby("sector")["wage"].max())
            """,
            """
            The groups come out in sorted order, mining before retail.
            """, level=1),
        out("o2", "What does this print?",
            """
            import pandas as pd
            s = pd.Series([3, 5, 2, 8, 1], index=["a", "b", "a", "c", "b"])
            print(s.groupby(level=0).sum().tolist())
            """,
            """
            Grouping by the index labels: a is 3 + 2, b is 5 + 1, c is 8.
            """, level=2),
        task("c1",
             """
             `tx` holds transactions with columns `county`, `month` and
             `amount`. Make `by_county`, total amount per county (a Series),
             and `n_by_month`, the number of transactions per month.
             """,
             """
             by_county = tx.groupby("county")["amount"].sum()
             n_by_month = tx.groupby("month").size()
             """,
             [
                 check("by_county", 'check(dict(by_county) == {"Elko": 70, "Nye": 60}, "`by_county` should be the sum of amount for each county.")'),
                 check("n_by_month", 'check(dict(n_by_month) == {1: 2, 2: 3}, "`n_by_month` should count the rows in each month.")'),
             ],
             """
             Select the column after grouping, then aggregate. `size()`
             counts rows per group.
             """,
             starter="""
             by_county = tx["amount"].sum()
             n_by_month = ...
             """,
             setup="import pandas as pd\ntx = pd.DataFrame({'county': ['Elko', 'Nye', 'Elko', 'Nye', 'Elko'], 'month': [1, 1, 2, 2, 2], 'amount': [10, 20, 25, 40, 35]})"),
    ]),

concept(
    "py.pd.groupby_ops", "GroupBy: aggregate, filter, transform, apply", ["py.pd.groupby"],
    "3.08", "Aggregation and Grouping", "03.08-aggregation-and-grouping",
    """
    Beyond a single aggregate, a GroupBy object offers four tools. Each
    returns a different shape:

    | method | returns |
    |---|---|
    | `agg` | one row per group (several statistics at once) |
    | `filter` | the original rows of groups that pass a test |
    | `transform` | one value per original row, same index |
    | `apply` | whatever your function returns, glued together |

    ```python
    g = df.groupby("county")["wage"]
    g.agg(["min", "mean", "max"])            # 3 columns per county
    df.groupby("county").filter(lambda d: len(d) >= 30)   # drop small counties
    df["demeaned"] = df["wage"] - g.transform("mean")     # within-county deviation
    ```

    `transform` is the workhorse for within-group demeaning, the fixed-effects
    trick in panel econometrics: subtract each group's mean from its own rows.
    """,
    [
        mc("m1", "`df` has 100 rows in 5 groups. How many rows does `df.groupby('g')['x'].transform('mean')` have?",
           "100", ["5", "1", "It depends on the group sizes"],
           """
           `transform` broadcasts each group's result back to that group's
           rows, so the output matches the input index.
           """, level=1),
        mc("m2", "Which keeps only the rows belonging to counties with average wage above 20?",
           "`df.groupby('county').filter(lambda d: d['wage'].mean() > 20)`",
           ["`df.groupby('county')['wage'].mean() > 20`", "`df.groupby('county').transform(lambda d: d.mean() > 20)`", "`df[df['wage'] > 20]`"],
           """
           `filter` evaluates the function on each group and keeps or drops
           all of that group's rows. The last option tests individual rows.
           """, level=2),
        mc("m3", "What does `df.groupby('sector')['wage'].agg(['mean', 'std'])` return?",
           "A DataFrame with one row per sector and columns `mean` and `std`",
           ["A Series of means", "A Series with a two-level index", "Two separate DataFrames"],
           """
           A list of functions gives one column per function.
           """, level=1),
        out("o1", "What does this print?",
            """
            import pandas as pd
            df = pd.DataFrame({"g": ["a", "a", "b", "b"], "x": [1.0, 3.0, 10.0, 14.0]})
            print((df["x"] - df.groupby("g")["x"].transform("mean")).tolist())
            """,
            """
            Group means are 2 and 12; each row minus its own group's mean.
            """, level=2),
        out("o2", "What does this print?",
            """
            import pandas as pd
            df = pd.DataFrame({"g": ["a", "a", "b", "c", "c", "c"], "x": range(6)})
            print(df.groupby("g").filter(lambda d: len(d) > 1)["g"].tolist())
            """,
            """
            Group b has one row, so it is dropped; the others keep all rows.
            """, level=2),
        task("c1",
             """
             `panel` has columns `county`, `year` and `wage`. Add a column
             `wage_dm` equal to each wage minus the mean wage of its county
             (within-county demeaning). Then make `summary`, a DataFrame with
             one row per county and columns `mean` and `max` of `wage`.
             """,
             """
             panel["wage_dm"] = panel["wage"] - panel.groupby("county")["wage"].transform("mean")
             summary = panel.groupby("county")["wage"].agg(["mean", "max"])
             """,
             [
                 check("wage_dm", 'check("wage_dm" in panel.columns and close(panel["wage_dm"], [-1.0, 1.0, -3.0, 0.0, 3.0]), "`wage_dm` should be each wage minus its county\'s mean wage, on every row.")'),
                 check("summary", 'check(list(getattr(summary, "columns", [])) == ["mean", "max"] and close(summary.loc["Nye"].values, [20.0, 23.0]), "`summary` should have columns mean and max, one row per county.")'),
             ],
             """
             `transform("mean")` returns the group mean aligned to every row,
             so the subtraction is row by row. `agg` with a list gives one
             column per statistic.
             """,
             starter="""
             panel["wage_dm"] = panel["wage"] - panel.groupby("county")["wage"].mean()
             summary = ...
             """,
             setup="import pandas as pd\npanel = pd.DataFrame({'county': ['Elko', 'Elko', 'Nye', 'Nye', 'Nye'], 'year': [1, 2, 1, 2, 3], 'wage': [24.0, 26.0, 17.0, 20.0, 23.0]})"),
    ]),

# ---------------------------------------------------------------- 3.09
concept(
    "py.pd.pivot", "Pivot tables", ["py.pd.groupby_ops", "py.pd.reshape_index"],
    "3.09", "Pivot Tables", "03.09-pivot-tables",
    """
    A pivot table is a two-way groupby laid out as a grid: one key down the
    rows, another across the columns, an aggregate in each cell.

    ```python
    df.pivot_table(values="wage", index="sector", columns="sex", aggfunc="mean")
    # is the same as
    df.groupby(["sector", "sex"])["wage"].mean().unstack()
    ```

    Useful options:

    - `aggfunc=` takes a name (`"mean"` is the default, `"sum"`,
      `"count"`, ...), a function, or a dict mapping columns to functions.
    - `index=` and `columns=` accept lists for multilevel grids, and you can
      bin a numeric column first with `pd.cut` to use it as a key.
    - `margins=True` adds an `All` row and column computed over the raw
      data, not by averaging the cell values.
    - `fill_value=` replaces empty cells.
    """,
    [
        mc("m1", "What `aggfunc` does `pivot_table` use if you do not pass one?",
           "`'mean'`", ["`'sum'`", "`'count'`", "`'first'`"],
           """
           The default is the mean. Use `aggfunc='sum'` for totals.
           """, level=1),
        mc("m2", "Which groupby expression gives the same grid as `df.pivot_table('pay', index='region', columns='year', aggfunc='sum')`?",
           "`df.groupby(['region', 'year'])['pay'].sum().unstack()`",
           ["`df.groupby('region')['pay'].sum()`", "`df.groupby(['year', 'region'])['pay'].sum()`", "`df.groupby('pay').sum().unstack()`"],
           """
           Group by both keys, aggregate, then move the column key (year)
           into the columns with `unstack`.
           """, level=2),
        mc("m3", "With `margins=True` and `aggfunc='mean'`, how is the `All` column computed?",
           "As the mean over all raw rows in that row group, not the mean of the cell means",
           ["As the sum of the cells", "As the mean of the cell means", "As the count of rows"],
           """
           Margins re-aggregate the underlying data, so unequal group sizes are
           weighted correctly.
           """, level=3),
        out("o1", "What does this print?",
            """
            import pandas as pd
            df = pd.DataFrame({"region": ["N", "N", "S", "S", "S"],
                               "year": [2023, 2024, 2023, 2024, 2024],
                               "pay": [10, 12, 8, 9, 11]})
            print(df.pivot_table("pay", index="region", columns="year", aggfunc="sum"))
            """,
            """
            South in 2024 has two rows, 9 and 11, which sum to 20.
            """, level=2),
        out("o2", "What does this print?",
            """
            import pandas as pd
            df = pd.DataFrame({"region": ["N", "N", "S"], "year": [1, 2, 1], "pay": [10, 12, 8]})
            t = df.pivot_table("pay", index="region", columns="year", aggfunc="sum", fill_value=0)
            print(t.loc["S", 2])
            """,
            """
            No South row has year 2, so that cell would be missing;
            `fill_value=0` puts 0 there.
            """, level=2),
        task("c1",
             """
             `survey` has columns `state`, `sex` and `income`. Make `grid`, a
             pivot table of median income with states as rows and sex as
             columns, including an `All` row and column.
             """,
             """
             grid = survey.pivot_table(values="income", index="state", columns="sex",
                                       aggfunc="median", margins=True)
             """,
             [
                 check("layout", 'check(list(getattr(grid, "index", [])) == ["NV", "UT", "All"] and list(grid.columns) == ["F", "M", "All"], "`grid` should have states (plus All) as rows and sex (plus All) as columns.")'),
                 check("median", 'check(close(grid.loc["NV", "M"], 50.0) and close(grid.loc["All", "All"], 46.0), "Use the median income in each cell, and margins=True for the All row and column.")'),
             ],
             """
             `aggfunc="median"` sets the statistic; `margins=True` adds the
             totals computed over the raw rows.
             """,
             starter="""
             grid = survey.pivot_table(values="income", index="state", columns="sex")
             """,
             setup="import pandas as pd\nsurvey = pd.DataFrame({'state': ['NV', 'NV', 'NV', 'UT', 'UT', 'UT', 'NV'], 'sex': ['F', 'M', 'M', 'F', 'M', 'F', 'F'], 'income': [40.0, 46.0, 54.0, 38.0, 60.0, 47.0, 44.0]})"),
    ]),

# ---------------------------------------------------------------- 3.10
concept(
    "py.pd.strings", "Vectorized string operations", ["py.pd.series_select"],
    "3.10", "Vectorized String Operations", "03.10-working-with-strings",
    """
    Text columns have a `.str` accessor that applies Python string methods
    to every entry, skipping missing values instead of crashing on them.
    In pandas 3 such columns have the dtype `str` by default.

    ```python
    names = pd.Series(["  elko county", "Washoe COUNTY", None])
    names.str.strip().str.title()          # 'Elko County', 'Washoe County', NaN
    names.str.len()
    names.str.contains("county", case=False)
    ```

    Regular-expression methods do pattern work: `contains`, `match`,
    `extract` (capture groups become columns), `replace`, `findall`,
    `split`. Indexing helpers include `str[0:3]`, `str.get(i)` after a
    split, and `str.cat`. For a column of delimited tags,
    `str.get_dummies("|")` makes one 0/1 indicator column per tag.

    ```python
    codes = pd.Series(["NAICS-445", "NAICS-212"])
    codes.str.extract(r"(\\d+)")     # one column of the digits
    ```
    """,
    [
        mc("m1", "What does `pd.Series(['ab', 'cde']).str.len()` return?",
           "A Series `[2, 3]`", ["`2`, the number of entries", "`5`, the total length", "An error: Series has no len"],
           """
           `.str` methods work element by element and return a Series.
           """, level=1),
        mc("m2", "What is the dtype of `pd.Series(['NV', 'UT'])` in pandas 3?",
           "`str`", ["`object`", "`category`", "`<U2`"],
           """
           Pandas 3 infers its dedicated string dtype for text by default,
           printed as `str`.
           """, level=1),
        mc("m3", "`s.str.get_dummies('|')` on `['wage|rent', 'rent']` returns",
           "A DataFrame with columns `rent` and `wage` holding 1s and 0s",
           ["A Series of lists", "The number of tags per row", "A boolean Series"],
           """
           Each distinct tag becomes an indicator column, sorted by name.
           """, level=2),
        out("o1", "What does this print?",
            """
            import pandas as pd
            s = pd.Series(["  elko county", "Washoe COUNTY", "nye"])
            print(s.str.strip().str.title().tolist())
            """,
            """
            `strip` removes the outer spaces; `title` capitalises each word.
            """, level=1),
        out("o2", "What does this print?",
            """
            import pandas as pd
            codes = pd.Series(["NAICS-445", "NAICS-212", "SIC-58"])
            print(codes.str.extract(r"-(\\d+)")[0].astype(int).sum())
            """,
            """
            The capture group pulls the digits after the dash: 445 + 212 + 58.
            """, level=2),
        out("o3", "What does this print?",
            """
            import pandas as pd
            tags = pd.Series(["wage|rent", "rent", "tax|wage"])
            print(tags.str.get_dummies("|"))
            """,
            """
            One indicator column per distinct tag, in alphabetical order.
            """, level=2),
        task("c1",
             """
             `place` holds messy county names such as `"  washoe county, NV"`.
             Make `clean`, the names with outer spaces removed, in title case,
             and with `" County"` and the state suffix dropped (so the result
             is `"Washoe"`). Then make `state`, the two-letter state code
             after the comma, in upper case.
             """,
             """
             parts = place.str.split(",")
             clean = parts.str[0].str.strip().str.title().str.replace(" County", "")
             state = parts.str[1].str.strip().str.upper()
             """,
             [
                 check("clean", 'check(list(clean) == ["Washoe", "Clark", "Elko", "Salt Lake"], "`clean` should be the bare county names in title case, without the word County.")'),
                 check("state", 'check(list(state) == ["NV", "NV", "NV", "UT"], "`state` should be the upper-case code after the comma.")'),
             ],
             """
             Split on the comma, then use `.str[0]` and `.str[1]` to take each
             part. Chain `strip`, `title` and `replace` on the name part.
             """,
             starter="""
             parts = ...
             clean = ...
             state = ...
             """,
             setup="import pandas as pd\nplace = pd.Series(['  washoe county, NV', 'CLARK COUNTY,nv', 'elko county , Nv', 'salt lake county, ut'])"),
    ]),

# ---------------------------------------------------------------- 3.11
concept(
    "py.pd.datetime", "Dates, times and date ranges", ["py.pd.series_select", "py.np.dtypes"],
    "3.11", "Working with Time Series", "03.11-working-with-time-series",
    """
    Python's `datetime` handles one date at a time; NumPy's `datetime64`
    packs many into an array. Pandas builds on both with **Timestamp** (one
    moment) and **DatetimeIndex** (many), plus **Period** for spans such as
    a month or quarter and **Timedelta** for durations.

    ```python
    pd.to_datetime(["2024-01-15", "2024-03-01"])     # parse strings
    pd.Timestamp("2024-02-28") + pd.Timedelta(days=2)  # 2024-03-01
    pd.date_range("2024-01-01", periods=4, freq="QE")  # quarter ends
    s = pd.Series([1.2, 1.5, 1.4],
                  index=pd.date_range("2024-01-31", periods=3, freq="ME"))
    s.loc["2024-02"]       # partial-string indexing: all of February
    ```

    Frequency codes in pandas 3 include `"D"` (day), `"B"` (business day),
    `"W"`, `"ME"` (month end), `"MS"` (month start), `"QE"`, `"YE"` and
    `"h"` (hour). The old `"M"` and `"H"` are no longer accepted. Codes take
    multipliers: `"2h"`, `"3ME"`.
    """,
    [
        mc("m1", "Which call gives the last day of each month for January to March 2024?",
           "`pd.date_range('2024-01-01', periods=3, freq='ME')`",
           ["`pd.date_range('2024-01-01', periods=3, freq='MS')`", "`pd.date_range('2024-01-01', periods=3, freq='D')`", "`pd.date_range('2024-01-31', periods=3)`"],
           """
           `ME` is month end; `MS` is month start; with no `freq`, the step is
           one day.
           """, level=1),
        mc("m2", "What is `pd.Timestamp('2024-02-28') + pd.Timedelta(days=1)`?",
           "`Timestamp('2024-02-29 00:00:00')`", ["`Timestamp('2024-03-01 00:00:00')`", "An error: you cannot add days to a date", "`Timestamp('2024-02-29')` as a string"],
           """
           2024 is a leap year, and Timestamp arithmetic knows the calendar.
           """, level=2),
        mc("m3", "In pandas 3, what happens with `pd.date_range('2024-01-01', periods=3, freq='M')`?",
           "A `ValueError`: the month-end alias is now `'ME'`",
           ["Three month starts", "Three minutes", "Three month ends, silently"],
           """
           The single-letter `M` for month end was removed. Use `ME` (or `MS`
           for month starts, `min` for minutes).
           """, level=2),
        out("o1", "What does this print?",
            """
            import pandas as pd
            idx = pd.date_range("2024-01-01", periods=4, freq="QE")
            print(idx.strftime("%Y-%m-%d").tolist())
            """,
            """
            Quarter ends: the last days of March, June, September and December.
            """, level=1),
        out("o2", "What does this print?",
            """
            import pandas as pd
            s = pd.Series(range(6), index=pd.date_range("2024-01-30", periods=6, freq="D"))
            print(s.loc["2024-02"].tolist())
            """,
            """
            The daily index covers Jan 30 to Feb 4. A partial string selects
            every timestamp in February.
            """, level=2),
        out("o3", "What does this print?",
            """
            import pandas as pd
            d = pd.to_datetime(["2024-03-15", "2024-07-04"])
            print(d.quarter.tolist(), (d[1] - d[0]).days)
            """,
            """
            DatetimeIndex has date-part attributes like `quarter`, and
            subtracting timestamps gives a Timedelta.
            """, level=2),
        task("c1",
             """
             `raw` is a DataFrame with a text column `date` (like
             `"2024-01-31"`) and a column `cpi`. Make `cpi`, a Series of the
             CPI values indexed by a DatetimeIndex built from `date`. Then set
             `h1` to the values from January through June 2024 using
             partial-string indexing.
             """,
             """
             import pandas as pd
             cpi = pd.Series(raw["cpi"].values, index=pd.to_datetime(raw["date"]))
             h1 = cpi.loc["2024-01":"2024-06"]
             """,
             [
                 check("index", """
                     import pandas as pd
                     check(isinstance(getattr(cpi, "index", None), pd.DatetimeIndex), "`cpi` should be indexed by a DatetimeIndex (use pd.to_datetime).")
                     check(len(cpi) == 9 and close(cpi.iloc[0], 300.0), "`cpi` should hold all 9 CPI values in order.")
                 """),
                 check("h1", 'check(len(h1) == 6 and close(h1.iloc[-1], 302.5), "`h1` should hold January to June 2024.")'),
             ],
             """
             `pd.to_datetime` parses the strings. With a DatetimeIndex,
             `loc["2024-01":"2024-06"]` takes whole months at both ends.
             `raw.set_index("date")` works too once the column is converted.
             """,
             starter="""
             import pandas as pd
             cpi = ...
             h1 = ...
             """,
             setup="import pandas as pd\nraw = pd.DataFrame({'date': [str(d.date()) for d in pd.date_range('2024-01-31', periods=9, freq='ME')], 'cpi': [300.0, 300.4, 301.0, 301.5, 302.0, 302.5, 303.1, 303.4, 304.0]})"),
    ]),

concept(
    "py.pd.resample", "Resampling, shifting and rolling windows", ["py.pd.datetime", "py.pd.groupby"],
    "3.11", "Working with Time Series", "03.11-working-with-time-series",
    """
    With a DatetimeIndex, three tools cover most time-series chores.

    **Resample** groups by calendar period, like a groupby on time:

    ```python
    daily.resample("ME").mean()    # monthly average
    daily.resample("QE").last()    # value at the end of each quarter
    ```

    **asfreq** picks the values at new timestamps without aggregating.

    **Shift** moves the data relative to the index, which is how you build
    lags and growth rates:

    ```python
    gdp.shift(1)                   # last period's value on this row
    gdp / gdp.shift(1) - 1         # period-on-period growth
    gdp.pct_change()               # the same thing
    ```

    **Rolling** windows compute a statistic over the last k observations:
    `s.rolling(3).mean()` is a 3-period moving average whose first two
    values are NaN because the window is not yet full.
    """,
    [
        mc("m1", "`s` is a daily Series covering one year. How many values does `s.resample('ME').sum()` have?",
           "12", ["365", "1", "52"],
           """
           Resampling to month end makes one bin per month.
           """, level=1),
        mc("m2", "What is the first value of `s.shift(1)`?",
           "NaN", ["The first value of `s`", "0", "The last value of `s`"],
           """
           Shifting forward by one moves every value down a row, leaving no
           value for the first row.
           """, level=1),
        mc("m3", "`s` has 10 values. How many non-missing values does `s.rolling(4).mean()` have?",
           "7", ["10", "6", "4"],
           """
           The first 3 windows are incomplete, so they are NaN; positions 4
           to 10 give 7 values.
           """, level=2),
        out("o1", "What does this print?",
            """
            import pandas as pd
            idx = pd.date_range("2024-01-01", periods=6, freq="15D")
            s = pd.Series([1, 2, 3, 4, 5, 6], index=idx)
            print(s.resample("ME").sum().tolist())
            """,
            """
            The dates are Jan 1, Jan 16, Jan 31, Feb 15, Mar 1, Mar 16. Summing
            by month gives 1+2+3, 4, and 5+6.
            """, level=3),
        out("o2", "What does this print?",
            """
            import pandas as pd
            gdp = pd.Series([100.0, 110.0, 99.0])
            print(gdp.pct_change().round(2).tolist())
            """,
            """
            Growth is 10% then -10%; the first value has no predecessor.
            """, level=2),
        out("o3", "What does this print?",
            """
            import pandas as pd
            s = pd.Series([2.0, 4.0, 6.0, 8.0])
            print(s.rolling(2).mean().tolist())
            """,
            """
            Each value averages itself and the previous one.
            """, level=1),
        task("c1",
             """
             `price` is a daily Series with a DatetimeIndex covering the first
             quarter of 2024. Make `monthly`, the average price per calendar
             month; `mom`, the month-on-month growth rate of `monthly` (first
             value NaN); and `ma7`, the 7-day moving average of the daily price.
             """,
             """
             monthly = price.resample("ME").mean()
             mom = monthly.pct_change()
             ma7 = price.rolling(7).mean()
             """,
             [
                 check("monthly", 'check(len(monthly) == 3 and close(monthly.iloc[0], 15.0), "`monthly` should have one average per month (resample with \'ME\').")'),
                 check("mom", """
                     import numpy as np
                     check(len(mom) == 3 and np.isnan(mom.iloc[0]) and close(mom.iloc[1], monthly.iloc[1] / monthly.iloc[0] - 1), "`mom` should be the growth rate from one month to the next.")
                 """),
                 check("ma7", """
                     import numpy as np
                     check(len(ma7) == len(price) and np.isnan(ma7.iloc[5]) and close(ma7.iloc[6], 3.0), "`ma7` should be a 7-day rolling mean on the daily data, NaN until 7 days are in.")
                 """),
             ],
             """
             `resample("ME")` bins by month; `pct_change` divides by the
             previous value; `rolling(7)` uses a window of 7 rows.
             """,
             starter="""
             monthly = ...
             mom = ...
             ma7 = price.rolling(7)
             """,
             setup="import numpy as np\nimport pandas as pd\nprice = pd.Series(np.arange(91, dtype=float), index=pd.date_range('2024-01-01', periods=91, freq='D'))"),
    ]),

# ---------------------------------------------------------------- 3.12
concept(
    "py.pd.eval_query", "Fast expressions: eval and query", ["py.pd.frame_select"],
    "3.12", "High-Performance Pandas: eval and query", "03.12-performance-eval-and-query",
    """
    An expression like `(a + b) / (c - d)` on large columns builds a
    temporary array for each step. `pd.eval` and the DataFrame methods
    `eval` and `query` parse the expression as a string and can evaluate it
    in one pass, saving memory and sometimes time on big frames.

    ```python
    df.eval("pay = wage * hours")          # returns a new frame with `pay`
    df.query("hours > 40 and wage < 20")   # rows that satisfy the condition
    cutoff = 25
    df.query("wage > @cutoff")             # @ refers to a Python variable
    ```

    Inside the string, column names are used bare. `and`, `or` and `not`
    work, as do comparisons chained like `10 < wage < 30`. For small frames
    the speed gain is nil; the readability is the main reason to reach for
    `query`.
    """,
    [
        mc("m1", "In `df.query('price > @limit')`, what does `@limit` refer to?",
           "A Python variable named `limit` in the surrounding code",
           ["A column named `limit`", "A row labelled `limit`", "A placeholder you must fill with `.format`"],
           """
           The `@` prefix pulls in local variables; bare names are columns.
           """, level=1),
        mc("m2", "With pandas copy-on-write, what does `df.eval('z = x + y')` do to `df`?",
           "Nothing: it returns a new DataFrame that has the extra column",
           ["Adds column `z` to `df` in place", "Raises an error because eval cannot assign", "Returns only the new column"],
           """
           `eval` returns a modified copy by default. Assign the result
           (`df = df.eval(...)`) to keep it.
           """, level=2),
        mc("m3", "When is `query` or `eval` most likely to beat ordinary pandas expressions for speed?",
           "On large frames with compound expressions, where intermediate arrays are costly",
           ["On every frame, always", "On very small frames", "Only when the frame has string columns"],
           """
           The overhead of parsing the string dominates on small data. The
           savings come from avoiding big temporaries.
           """, level=2),
        out("o1", "What does this print?",
            """
            import pandas as pd
            df = pd.DataFrame({"wage": [15.0, 22.0, 30.0], "hours": [40, 35, 45]})
            cutoff = 20
            print(df.query("wage > @cutoff and hours >= 40").index.tolist())
            """,
            """
            Only row 2 has a wage above 20 and at least 40 hours.
            """, level=1),
        out("o2", "What does this print?",
            """
            import pandas as pd
            df = pd.DataFrame({"a": [1, 2], "b": [10, 20]})
            out = df.eval("c = a * b")
            print(df.columns.tolist(), out["c"].tolist())
            """,
            """
            `df` is unchanged; the returned frame has the new column.
            """, level=2),
        task("c1",
             """
             `hh` has columns `income`, `size` and `rent`. Using `eval`, make
             `hh2`, a copy of `hh` with a new column `per_cap` equal to income
             divided by size. Then, using `query` with the Python variable
             `burden`, set `stressed` to the rows of `hh2` whose rent is more
             than `burden` times income.
             """,
             """
             hh2 = hh.eval("per_cap = income / size")
             stressed = hh2.query("rent > @burden * income")
             """,
             [
                 check("per_cap", 'check("per_cap" in getattr(hh2, "columns", []) and close(hh2["per_cap"], [20.0, 15.0, 30.0, 9.0]), "`hh2` should have per_cap = income / size.")'),
                 check("stressed", 'check(list(stressed.index) == [1, 3], "`stressed` should keep rows where rent exceeds burden times income.")'),
                 check("methods", 'check(uses("eval(") and uses("query(") and uses("@burden"), "Use eval for the column and query with @burden for the rows.")'),
             ],
             """
             `eval` with an assignment returns a new frame. Inside `query`,
             `@burden` reads the Python variable.
             """,
             starter="""
             hh2 = ...
             stressed = ...
             """,
             setup="import pandas as pd\nhh = pd.DataFrame({'income': [40.0, 45.0, 60.0, 36.0], 'size': [2, 3, 2, 4], 'rent': [10.0, 15.0, 12.0, 14.0]})\nburden = 0.3"),
    ]),

    ],
}
