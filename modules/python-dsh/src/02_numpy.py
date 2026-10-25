# Chapter 2 of the Python Data Science Handbook: Introduction to NumPy.
#
# Every question and note here is original. Each concept links to the book
# section it follows; the book's own text is CC-BY-NC-ND and is never adapted.
# Build with: python3 scripts/python_bank.py build numpy

CHAPTER = {
    "chapter": "numpy",
    "title": "Introduction to NumPy",
    "domain": "python_numpy",
    "concepts": [

# ---------------------------------------------------------------- 2.01
concept(
    "py.np.dtypes", "Fixed-type arrays and dtypes", [],
    "2.01", "Understanding Data Types in Python", "02.01-understanding-data-types",
    """
    A Python list can hold anything, so every element is a full Python object
    with its own type tag and reference count. A NumPy array stores one type
    for the whole block, packed side by side in memory. That single type is
    the array's **dtype**.

    Mixing types does not give you a mixed array. NumPy picks one dtype that
    can hold every value, a step called *upcasting*: ints with a float become
    floats, and anything with a string becomes strings.

    ```python
    import numpy as np
    wages = np.array([15, 22, 31])        # int64
    prices = np.array([1, 2.5, 3])        # float64: the ints were upcast
    codes = np.array([1, 2], dtype=np.int8)
    ```

    Think of a list as a shelf of labelled boxes and an array as an egg
    carton: every slot is the same size, which is what makes it fast.
    """,
    [
        mc("m1", "Why can NumPy add two million-element arrays much faster than a Python loop over two lists?",
           "Every element has the same fixed type, so the work runs in compiled code over a packed block of memory",
           ["NumPy uses several CPU cores for every operation by default",
            "NumPy stores arrays on disk and streams them",
            "Python lists cannot hold more than a few thousand numbers"],
           """
           A list stores pointers to separate Python objects, so a loop has to
           look up each object's type on every step. An array's single dtype
           lets one compiled loop walk through contiguous memory.
           """, level=1),
        mc("m2", "What is the dtype of `np.array([3, 4.0, 5])`?",
           "`float64`", ["`int64`", "`object`", "a mix of `int64` and `float64`"],
           """
           One float forces the whole array to a floating type. An array never
           mixes dtypes; NumPy upcasts to the type that can represent every value.
           """, level=1),
        mc("m3", "You run `a = np.array([1, 2, 3])` and then `a[0] = 9.7`. What is `a[0]` afterwards?",
           "`9`", ["`9.7`", "`10`", "It raises a `TypeError`"],
           """
           Assigning into an existing array converts the value to the array's
           dtype. For an integer array the fractional part is dropped (not
           rounded), so 9.7 is stored as 9.
           """),
        out("o1", "What does this print?",
            """
            import numpy as np
            x = np.array([1, 2, 3], dtype=float)
            print(x)
            """,
            """
            `dtype=float` means `float64`. NumPy prints floats with a trailing
            dot when they have no fractional part.
            """, level=1),
        out("o2", "What does this print?",
            """
            import numpy as np
            x = np.array([True, 2, 3.5])
            print(x.dtype)
            """,
            """
            Booleans upcast to numbers (True is 1), and the 3.5 makes the whole
            array floating point.
            """),
        task("c1",
             """
             Create an array `hours` holding the values 8, 7.5 and 9, and an array
             `ids` holding 101, 102 and 103 stored as 16-bit integers.
             """,
             """
             import numpy as np
             hours = np.array([8, 7.5, 9])
             ids = np.array([101, 102, 103], dtype=np.int16)
             """,
             [
                 check("hours", """
                     check(same_array(hours, [8, 7.5, 9]), "`hours` should hold 8, 7.5 and 9.")
                     check(hours.dtype.kind == "f", "`hours` should be a floating-point array.")
                 """),
                 check("ids", """
                     check(same_array(ids, [101, 102, 103]), "`ids` should hold 101, 102 and 103.")
                     check(str(ids.dtype) == "int16", "`ids` should have dtype int16. Pass dtype= when you create it.")
                 """),
             ],
             """
             `np.array` infers the dtype from the values, so 7.5 makes `hours`
             float64. For `ids`, the `dtype=` argument fixes the storage type.
             """,
             starter="""
             import numpy as np
             hours = ...
             ids = ...
             """, level=1),
        task("c2",
             """
             The list `raw` (already defined) mixes integers and floats. Make
             `as_int`, an integer array of the same values with the decimals
             dropped, by converting with `.astype`.
             """,
             """
             import numpy as np
             as_int = np.array(raw).astype(int)
             """,
             [
                 check("values", """
                     import numpy as np
                     check(same_array(as_int, [3, 7, 2, 9]), "`as_int` should be [3, 7, 2, 9]: astype(int) drops decimals.")
                     check(as_int.dtype.kind == "i", "`as_int` should have an integer dtype.")
                 """),
             ],
             """
             `np.array(raw)` is float64 because of the decimals. `.astype(int)`
             returns a new integer array, truncating toward zero.
             """,
             starter="""
             import numpy as np
             print(raw)
             as_int = ...
             """,
             setup="raw = [3, 7.9, 2.2, 9]"),
    ]),

concept(
    "py.np.create", "Creating arrays from scratch", ["py.np.dtypes"],
    "2.01", "Understanding Data Types in Python", "02.01-understanding-data-types",
    """
    You rarely type arrays out by hand. NumPy has constructors for the common
    shapes:

    ```python
    np.zeros(5)                 # five 0.0s
    np.ones((2, 3), dtype=int)  # 2 rows, 3 columns of 1
    np.full(4, 2.5)             # [2.5 2.5 2.5 2.5]
    np.arange(0, 10, 2)         # [0 2 4 6 8]  (stop is excluded)
    np.linspace(0, 1, 5)        # 5 evenly spaced points, both ends included
    np.eye(3)                   # 3 x 3 identity matrix
    rng = np.random.default_rng(42)
    rng.normal(0, 1, size=3)    # three draws from N(0, 1)
    ```

    The difference to remember: `arange` takes a **step** and leaves out the
    stop value; `linspace` takes a **count** and includes both ends.
    """,
    [
        mc("m1", "Which call gives `[0.   0.25 0.5  0.75 1.  ]`?",
           "`np.linspace(0, 1, 5)`", ["`np.arange(0, 1, 5)`", "`np.arange(0, 1.25, 5)`", "`np.linspace(0, 1, 4)`"],
           """
           `linspace(start, stop, num)` returns `num` evenly spaced points and
           includes the stop. Five points from 0 to 1 are a quarter apart.
           """, level=1),
        mc("m2", "What shape does `np.zeros((3, 2))` have?",
           "3 rows and 2 columns", ["2 rows and 3 columns", "6 elements in one row", "3 elements, each equal to 2"],
           """
           A shape tuple lists the length of each axis in order: rows first,
           then columns.
           """, level=1),
        mc("m3", "Why pass a seed, as in `np.random.default_rng(7)`, in an analysis script?",
           "So the same random numbers come out every time the script runs",
           ["So the numbers are more random", "So the generator runs faster", "Because the generator raises an error without one"],
           """
           A seed fixes the generator's starting state. Reproducible draws mean a
           colleague who reruns your bootstrap gets your exact numbers.
           """),
        out("o1", "What does this print?",
            """
            import numpy as np
            print(np.arange(3, 12, 3))
            """,
            """
            `arange` counts from 3 in steps of 3 and stops *before* 12.
            """, level=1),
        out("o2", "What does this print?",
            """
            import numpy as np
            a = np.full((2, 3), 7)
            print(a.sum())
            """,
            """
            `np.full((2, 3), 7)` is six sevens, which sum to 42.
            """),
        out("o3", "What does this print?",
            """
            import numpy as np
            print(np.eye(2, dtype=int))
            """,
            """
            `eye(n)` is the n x n identity: ones on the diagonal, zeros elsewhere.
            """, level=1),
        task("c1",
             """
             Create:

             - `grid`: a 4 x 5 array of zeros,
             - `quarters`: the 9 evenly spaced values from 0 to 2 inclusive,
             - `odds`: the odd numbers from 1 to 19 inclusive, using `np.arange`.
             """,
             """
             import numpy as np
             grid = np.zeros((4, 5))
             quarters = np.linspace(0, 2, 9)
             odds = np.arange(1, 20, 2)
             """,
             [
                 check("grid", """
                     check(getattr(grid, "shape", None) == (4, 5), "`grid` should have shape (4, 5).")
                     check(not grid.any(), "`grid` should be all zeros.")
                 """),
                 check("quarters", """
                     import numpy as np
                     check(close(quarters, np.arange(9) * 0.25), "`quarters` should be 0, 0.25, ..., 2.")
                 """),
                 check("odds", """
                     check(same_array(odds, list(range(1, 20, 2))), "`odds` should be 1, 3, ..., 19.")
                     check(uses("arange"), "Build `odds` with np.arange.")
                 """),
             ],
             """
             `linspace(0, 2, 9)` gives nine points a quarter apart because both
             ends are included. For `arange`, the stop must be past 19 (20 works)
             because the stop value is excluded.
             """,
             starter="""
             import numpy as np
             grid = ...
             quarters = ...
             odds = ...
             """, level=1),
        task("c2",
             """
             Simulate 1,000 daily returns. Make a generator with
             `np.random.default_rng(2024)` and draw `returns` from a normal
             distribution with mean 0.001 and standard deviation 0.02.
             """,
             """
             import numpy as np
             rng = np.random.default_rng(2024)
             returns = rng.normal(0.001, 0.02, size=1000)
             """,
             [
                 check("shape", """
                     check(getattr(returns, "shape", None) == (1000,), "`returns` should hold 1,000 draws.")
                 """),
                 check("seeded", """
                     import numpy as np
                     ref = np.random.default_rng(2024).normal(0.001, 0.02, size=1000)
                     check(close(returns, ref), "Use default_rng(2024) and draw once with rng.normal(mean, sd, size=1000).")
                 """),
             ],
             """
             `rng.normal(loc, scale, size)` draws `size` values. Seeding the
             generator makes the 1,000 draws identical on every run.
             """,
             starter="""
             import numpy as np
             rng = ...
             returns = ...
             """),
    ]),

# ---------------------------------------------------------------- 2.02
concept(
    "py.np.attributes", "Array attributes: ndim, shape, size", ["py.np.create"],
    "2.02", "The Basics of NumPy Arrays", "02.02-the-basics-of-numpy-arrays",
    """
    Every array carries a few facts about itself:

    | attribute | meaning | for `np.zeros((3, 4))` |
    |---|---|---|
    | `ndim` | number of axes | `2` |
    | `shape` | length along each axis | `(3, 4)` |
    | `size` | total element count | `12` |
    | `dtype` | element type | `float64` |
    | `itemsize` | bytes per element | `8` |
    | `nbytes` | bytes for the data | `96` |

    A panel of 50 states observed over 20 years stored as a `(50, 20)` array
    has `ndim` 2, `size` 1000 and, as float64, `nbytes` 8000.
    """,
    [
        mc("m1", "An array has shape `(4, 6, 2)`. What is its `size`?",
           "48", ["12", "3", "(4, 6, 2)"],
           "`size` multiplies the axis lengths: 4 x 6 x 2 = 48.", level=1),
        mc("m2", "What is `np.zeros((2, 5), dtype=np.int32).nbytes`?",
           "40", ["10", "80", "20"],
           "Ten elements at 4 bytes each (32 bits) is 40 bytes.", level=2),
        mc("m3", "Which attribute tells you how many axes an array has?",
           "`ndim`", ["`size`", "`shape[0]`", "`itemsize`"],
           "`ndim` is the number of axes; `shape` lists their lengths.", level=1),
        out("o1", "What does this print?",
            """
            import numpy as np
            x = np.ones((3, 2, 4))
            print(x.ndim, x.shape[1], x.size)
            """,
            "Three axes; the second has length 2; 3 x 2 x 4 = 24 elements.", level=1),
        out("o2", "What does this print?",
            """
            import numpy as np
            print(np.arange(5).shape)
            """,
            """
            A one-dimensional array's shape is a one-element tuple, written
            with a trailing comma.
            """),
        task("c1",
             """
             `panel` is already defined. Store its number of axes in `n_axes`,
             its number of rows in `n_rows`, and its total element count in `n_cells`.
             """,
             """
             n_axes = panel.ndim
             n_rows = panel.shape[0]
             n_cells = panel.size
             """,
             [
                 check("n_axes", 'check(n_axes == 2, "`n_axes` should be the number of axes (ndim).")'),
                 check("n_rows", 'check(n_rows == 50, "`n_rows` should be the length of the first axis (shape[0]).")'),
                 check("n_cells", 'check(n_cells == 1000, "`n_cells` should be the total number of elements (size).")'),
                 check("not hard-coded", 'check(uses("panel"), "Read the values from `panel` rather than typing numbers.")'),
             ],
             "Read facts from the array itself so the code still works when the data changes shape.",
             starter="""
             n_axes = ...
             n_rows = ...
             n_cells = ...
             """,
             setup="import numpy as np\npanel = np.zeros((50, 20))", level=1),
    ]),

concept(
    "py.np.indexing", "Indexing single elements", ["py.np.attributes"],
    "2.02", "The Basics of NumPy Arrays", "02.02-the-basics-of-numpy-arrays",
    """
    Indexing works like a list, counting from 0, and negative numbers count
    from the end. For more than one axis, put the indices in one pair of
    brackets separated by commas: `a[row, col]`.

    ```python
    m = np.array([[10, 11, 12],
                  [20, 21, 22]])
    m[0, 2]    # 12
    m[-1, 0]   # 20  (last row, first column)
    m[1]       # the whole second row: [20 21 22]
    m[1, 1] = 99   # change one element in place
    ```

    `m[1][1]` also works but takes two steps (first the row, then the
    element); `m[1, 1]` is the idiomatic form.
    """,
    [
        mc("m1", "For `x = np.array([4, 8, 15, 16, 23, 42])`, what is `x[-2]`?",
           "23", ["16", "42", "8"],
           "-1 is the last element (42), so -2 is the one before it.", level=1),
        mc("m2", "`m` has shape `(3, 4)`. Which expression gets the element in the last row and second column?",
           "`m[2, 1]`", ["`m[1, 2]`", "`m[3, 2]`", "`m[-1, 2]`"],
           "Rows come first. The last of three rows is index 2 (or -1); the second column is index 1.", level=1),
        mc("m3", "What happens when you run `np.arange(5)[5]`?",
           "An `IndexError`, because valid indices are 0 to 4",
           ["It returns the last element", "It returns 0", "It returns an empty array"],
           "Five elements have indices 0 through 4. Index 5 is out of bounds.", level=1),
        out("o1", "What does this print?",
            """
            import numpy as np
            m = np.arange(12).reshape(3, 4)
            print(m[1, -1])
            """,
            "Row 1 is `[4 5 6 7]`; its last element is 7.", level=2),
        out("o2", "What does this print?",
            """
            import numpy as np
            m = np.zeros((2, 2), dtype=int)
            m[0, 1] = 5
            m[1, 0] = -3.8
            print(m)
            """,
            "Assignment converts to the array's dtype, so -3.8 becomes -3 (truncated toward zero).", level=2),
        task("c1",
             """
             `scores` is a 2-D array with one row per student and one column per exam.
             Set `first_last` to the last exam score of the first student, and
             change the second student's first exam score to 100.
             """,
             """
             first_last = scores[0, -1]
             scores[1, 0] = 100
             """,
             [
                 check("first_last", 'check(first_last == 88, "`first_last` should be row 0, last column.")'),
                 check("updated", 'check(scores[1, 0] == 100, "Set row 1, column 0 to 100.")'),
                 check("only one change", """
                     import numpy as np
                     ref = np.array([[71, 64, 88], [100, 77, 90], [58, 80, 69]])
                     check(same_array(scores, ref), "Change only the second student's first exam.")
                 """),
             ],
             "Index with `[row, column]`; -1 picks the last column without knowing how many exams there are.",
             starter="""
             print(scores)
             first_last = ...
             """,
             setup="import numpy as np\nscores = np.array([[71, 64, 88], [93, 77, 90], [58, 80, 69]])", level=1),
    ]),

concept(
    "py.np.slicing", "Slicing subarrays", ["py.np.indexing"],
    "2.02", "The Basics of NumPy Arrays", "02.02-the-basics-of-numpy-arrays",
    """
    A slice `start:stop:step` picks a range; leave a part out to take its
    default (0, the end, 1). The stop is excluded, as with `range`.

    ```python
    x = np.arange(10)
    x[:3]      # [0 1 2]
    x[7:]      # [7 8 9]
    x[::3]     # [0 3 6 9]
    x[::-1]    # reversed
    ```

    Each axis gets its own slice, separated by commas:

    ```python
    m = np.arange(12).reshape(3, 4)
    m[:2, 1:3]   # first two rows, columns 1 and 2
    m[:, 0]      # first column (as a 1-D array)
    m[::-1, :]   # rows in reverse order
    ```
    """,
    [
        mc("m1", "For `x = np.arange(10)`, what is `x[2:8:2]`?",
           "`[2 4 6]`", ["`[2 4 6 8]`", "`[2 3 4 5 6 7]`", "`[4 6 8]`"],
           "Start at 2, step by 2, stop before 8.", level=1),
        mc("m2", "`m` has shape `(5, 6)`. What is the shape of `m[1:4, ::2]`?",
           "`(3, 3)`", ["`(3, 6)`", "`(4, 3)`", "`(3, 2)`"],
           "Rows 1, 2, 3 give 3 rows; every second of 6 columns gives 3 columns.", level=2),
        mc("m3", "Which expression selects the last column of a 2-D array `m`?",
           "`m[:, -1]`", ["`m[-1]`", "`m[-1, :]`", "`m[:-1]`"],
           "`:` keeps every row, `-1` picks the last column. `m[-1]` is the last *row*.", level=1),
        out("o1", "What does this print?",
            """
            import numpy as np
            x = np.arange(1, 8)
            print(x[::-2])
            """,
            "A negative step walks backwards from the end: 7, 5, 3, 1.", level=2),
        out("o2", "What does this print?",
            """
            import numpy as np
            m = np.arange(12).reshape(3, 4)
            print(m[1:, :2])
            """,
            "Rows 1 and 2, columns 0 and 1.", level=2),
        task("c1",
             """
             `prices` has one row per week and one column per weekday (Mon to Fri).
             Set `last_two_weeks` to the final two rows, `wednesdays` to the
             Wednesday column for every week, and `reversed_weeks` to the rows in
             reverse order.
             """,
             """
             last_two_weeks = prices[-2:]
             wednesdays = prices[:, 2]
             reversed_weeks = prices[::-1]
             """,
             [
                 check("last_two_weeks", 'check(same_array(last_two_weeks, prices_ref[-2:]), "`last_two_weeks` should be the last two rows.")'),
                 check("wednesdays", 'check(same_array(wednesdays, prices_ref[:, 2]), "`wednesdays` should be column index 2 for every row, as a 1-D array.")'),
                 check("reversed_weeks", 'check(same_array(reversed_weeks, prices_ref[::-1]), "`reversed_weeks` should list the rows from last to first.")'),
             ],
             "A negative start counts from the end, `:` keeps a whole axis, and a step of -1 reverses.",
             starter="""
             print(prices)
             last_two_weeks = ...
             wednesdays = ...
             reversed_weeks = ...
             """,
             setup="import numpy as np\nprices = np.arange(100, 130).reshape(6, 5)\nprices_ref = prices.copy()"),
    ]),

concept(
    "py.np.views", "Views versus copies", ["py.np.slicing"],
    "2.02", "The Basics of NumPy Arrays", "02.02-the-basics-of-numpy-arrays",
    """
    Slicing a list makes a new list. Slicing an array does **not** copy the
    data: it makes a *view*, a new window onto the same memory. Change the
    view and the original changes too.

    ```python
    x = np.arange(6)
    head = x[:3]
    head[0] = 100
    print(x)          # [100   1   2   3   4   5]
    ```

    This is deliberate: you can work on a slice of a large dataset without
    duplicating it. When you want independence, ask for it with `.copy()`:

    ```python
    head = x[:3].copy()   # changing head no longer touches x
    ```

    `np.shares_memory(a, b)` tells you whether two arrays overlap.
    """,
    [
        mc("m1", "After `a = np.zeros(4); b = a[1:3]; b[:] = 7`, what is `a`?",
           "`[0. 7. 7. 0.]`", ["`[0. 0. 0. 0.]`", "`[7. 7. 7. 7.]`", "`[7. 7. 0. 0.]`"],
           "`b` is a view of `a[1:3]`, so filling `b` fills those two positions of `a`.", level=1),
        mc("m2", "Which line makes `sub` independent of the array `big`?",
           "`sub = big[:10].copy()`", ["`sub = big[:10]`", "`sub = big[:10:1]`", "`sub = big[0:10]`"],
           "Every plain slice is a view. Only `.copy()` allocates new memory.", level=1),
        mc("m3", "For a Python **list** `L`, does `L[:2][0] = 9` change `L`?",
           "No: list slicing makes a new list", ["Yes, exactly like an array view", "Only if `L` holds numbers", "It raises an error"],
           "Lists copy on slicing; NumPy arrays return views. Mixing up the two is a common bug when moving from lists to arrays.", level=2),
        out("o1", "What does this print?",
            """
            import numpy as np
            x = np.arange(5)
            y = x[::2]
            y *= 10
            print(x)
            """,
            "`y` views elements 0, 2 and 4 of `x`, so multiplying `y` in place changes those entries of `x`.", level=2),
        out("o2", "What does this print?",
            """
            import numpy as np
            x = np.arange(5)
            y = x[1:4].copy()
            y[0] = -1
            print(x[1], y[0])
            """,
            "`y` is a copy, so `x` keeps its 1.", level=1),
        task("c1",
             """
             `data` is a large array. Make `sample`, a copy of its first 5 values,
             then set every element of `sample` to 0, without changing `data`.
             """,
             """
             sample = data[:5].copy()
             sample[:] = 0
             """,
             [
                 check("sample zeroed", 'check(getattr(sample, "shape", None) == (5,) and not sample.any(), "`sample` should be five zeros.")'),
                 check("data untouched", 'check(same_array(data, data_ref), "`data` changed. Copy the slice before modifying it.")'),
                 check("independent", """
                     import numpy as np
                     check(not np.shares_memory(sample, data), "`sample` still shares memory with `data`.")
                 """),
             ],
             "Without `.copy()`, `sample[:] = 0` would write zeros into `data` as well.",
             starter="""
             sample = data[:5]
             sample[:] = 0
             """,
             setup="import numpy as np\ndata = np.arange(1, 21)\ndata_ref = data.copy()"),
    ]),

concept(
    "py.np.reshape", "Reshaping and new axes", ["py.np.attributes"],
    "2.02", "The Basics of NumPy Arrays", "02.02-the-basics-of-numpy-arrays",
    """
    `reshape` reinterprets the same elements under a new shape. The total
    size must stay the same, and one dimension may be `-1` to mean "whatever
    fits":

    ```python
    x = np.arange(12)
    x.reshape(3, 4)      # 3 rows of 4
    x.reshape(2, -1)     # 2 rows; NumPy works out 6 columns
    ```

    To turn a 1-D array into a column or a row, add an axis of length 1 with
    `np.newaxis` (or `None`):

    ```python
    v = np.array([1, 2, 3])
    v[:, np.newaxis]     # shape (3, 1): a column
    v[np.newaxis, :]     # shape (1, 3): a row
    ```

    Scikit-learn wants a 2-D feature matrix even for one feature, which is
    where `x.reshape(-1, 1)` comes up constantly.
    """,
    [
        mc("m1", "What does `np.arange(10).reshape(3, -1)` do?",
           "Raises a `ValueError`: 10 elements do not split into 3 equal rows",
           ["Returns shape `(3, 4)` padded with zeros", "Returns shape `(3, 3)` and drops one element", "Returns shape `(3, 10)`"],
           "Reshape never adds or drops elements. 10 is not divisible by 3.", level=2),
        mc("m2", "`v` has shape `(5,)`. What is the shape of `v[:, np.newaxis]`?",
           "`(5, 1)`", ["`(1, 5)`", "`(5,)`", "`(5, 5)`"],
           "The new axis goes where `np.newaxis` appears: after the existing one, giving a column.", level=1),
        mc("m3", "Why do you often see `X = x.reshape(-1, 1)` before fitting a scikit-learn model?",
           "Models expect a 2-D features matrix, one row per sample, even with a single feature",
           ["It sorts the data", "It removes missing values", "It converts integers to floats"],
           "`reshape(-1, 1)` turns n values into n rows of one column each.", level=2),
        out("o1", "What does this print?",
            """
            import numpy as np
            print(np.arange(6).reshape(2, 3))
            """,
            "Elements fill the first row, then the second (row-major order).", level=1),
        out("o2", "What does this print?",
            """
            import numpy as np
            x = np.arange(24).reshape(2, 3, -1)
            print(x.shape)
            """,
            "24 / (2 x 3) = 4, so the -1 becomes 4.", level=2),
        task("c1",
             """
             `monthly` holds 36 months of sales as a 1-D array. Make
             `by_year`, with one row per year and one column per month. Then make
             `as_column`, `monthly` as a single column (shape `(36, 1)`).
             """,
             """
             by_year = monthly.reshape(3, 12)
             as_column = monthly.reshape(-1, 1)
             """,
             [
                 check("by_year", 'check(getattr(by_year, "shape", None) == (3, 12), "`by_year` should have shape (3, 12).")'),
                 check("by_year order", 'check(same_array(by_year[1], monthly[12:24]), "Row 1 of `by_year` should be months 13 to 24.")'),
                 check("as_column", 'check(getattr(as_column, "shape", None) == (36, 1), "`as_column` should have shape (36, 1).")'),
             ],
             "`reshape(3, 12)` fills row by row, so each row is one year. `reshape(-1, 1)` or `monthly[:, np.newaxis]` gives the column.",
             starter="""
             by_year = ...
             as_column = ...
             """,
             setup="import numpy as np\nmonthly = np.arange(200, 236)"),
    ]),

concept(
    "py.np.concat_split", "Joining and splitting arrays", ["py.np.reshape"],
    "2.02", "The Basics of NumPy Arrays", "02.02-the-basics-of-numpy-arrays",
    """
    Joining:

    ```python
    np.concatenate([a, b])          # end to end along axis 0
    np.concatenate([m1, m2], axis=1)  # side by side
    np.vstack([row1, row2])         # stack vertically (new rows)
    np.hstack([col1, col2])         # stack horizontally
    ```

    Splitting does the reverse. Pass the *positions* where the pieces start:

    ```python
    x = np.arange(10)
    left, mid, right = np.split(x, [3, 7])   # [0..2], [3..6], [7..9]
    top, bottom = np.vsplit(m, 2)            # two equal halves by rows
    ```

    Splitting a sample into training and test rows is the everyday use.
    """,
    [
        mc("m1", "`a` and `b` both have shape `(2, 3)`. What is the shape of `np.vstack([a, b])`?",
           "`(4, 3)`", ["`(2, 6)`", "`(2, 2, 3)`", "`(12,)`"],
           "Vertical stacking adds rows: 2 + 2 rows of 3 columns.", level=1),
        mc("m2", "`a` and `b` both have shape `(2, 3)`. What is the shape of `np.concatenate([a, b], axis=1)`?",
           "`(2, 6)`", ["`(4, 3)`", "`(2, 3, 2)`", "`(4, 6)`"],
           "Axis 1 is the column axis, so the columns are joined: 3 + 3.", level=2),
        mc("m3", "How many pieces does `np.split(x, [2, 5, 8])` return?",
           "4", ["3", "2", "It depends on the length of `x`"],
           "Three split points give four pieces: before 2, 2 to 4, 5 to 7, and 8 onwards.", level=1),
        out("o1", "What does this print?",
            """
            import numpy as np
            a = np.array([1, 2])
            b = np.array([3, 4, 5])
            print(np.concatenate([b, a]))
            """,
            "Concatenation keeps the order of the list you pass.", level=1),
        out("o2", "What does this print?",
            """
            import numpy as np
            x = np.arange(8)
            first, second = np.split(x, [5])
            print(second)
            """,
            "The single split point 5 cuts before index 5.", level=1),
        task("c1",
             """
             `X` has 100 rows. Split it into `train` (the first 80 rows) and `test`
             (the last 20) with `np.split`. Then build `both`, the two stacked back
             together vertically.
             """,
             """
             import numpy as np
             train, test = np.split(X, [80])
             both = np.vstack([train, test])
             """,
             [
                 check("split", 'check(getattr(train, "shape", None) == (80, 3) and getattr(test, "shape", None) == (20, 3), "`train` should be 80 x 3 and `test` 20 x 3.")'),
                 check("used split", 'check(uses("split"), "Use np.split for this task.")'),
                 check("both", 'check(same_array(both, X), "`both` should equal the original X.")'),
             ],
             "`np.split(X, [80])` cuts before row 80. `vstack` puts the pieces back in order.",
             starter="""
             import numpy as np
             train, test = ...
             both = ...
             """,
             setup="import numpy as np\nX = np.arange(300).reshape(100, 3)"),
    ]),

# ---------------------------------------------------------------- 2.03
concept(
    "py.np.ufuncs", "Vectorized arithmetic with ufuncs", ["py.np.create"],
    "2.03", "Computation on NumPy Arrays: Universal Functions", "02.03-computation-on-arrays-ufuncs",
    """
    A *universal function* (ufunc) applies an operation to every element in
    compiled code. The arithmetic operators on arrays are ufuncs in disguise:

    ```python
    price = np.array([10.0, 20.0, 40.0])
    qty = np.array([3, 1, 2])
    revenue = price * qty          # np.multiply(price, qty) -> [30. 20. 80.]
    growth = np.log(price)         # natural log of each element
    np.abs(np.array([-2, 3]))      # [2 3]
    ```

    Writing `for i in range(len(x)): out[i] = ...` in Python is the slow way;
    one ufunc call replaces the loop and is typically tens to hundreds of
    times faster. "Vectorize" means exactly this.
    """,
    [
        mc("m1", "Which is the vectorized way to square every element of `x`?",
           "`x ** 2`", ["`[v ** 2 for v in x]`", "`for i in range(len(x)): x[i] = x[i] ** 2`", "`map(lambda v: v ** 2, x)`"],
           "`x ** 2` calls `np.power` once over the whole array. The others loop in Python.", level=1),
        mc("m2", "What is `np.array([1, 2, 3]) / np.array([2, 2, 2])`?",
           "`[0.5 1.  1.5]`", ["`[0 1 1]`", "`[0.5 1 1.5]` as integers", "An error, because the arrays are integers"],
           "True division `/` always produces floats; `//` is floor division.", level=1),
        mc("m3", "`np.exp(np.log(x))` for positive `x` returns",
           "`x` again, up to floating-point rounding", ["`log(x)`", "`1`", "`e`"],
           "`exp` and `log` are inverse ufuncs, applied element by element.", level=2),
        out("o1", "What does this print?",
            """
            import numpy as np
            x = np.arange(1, 5)
            print(10 - x * 2)
            """,
            "Multiplication first, element by element: 10 - [2 4 6 8].", level=1),
        out("o2", "What does this print?",
            """
            import numpy as np
            x = np.array([-7, 7])
            print(x // 2, x % 3)
            """,
            "Floor division rounds toward minus infinity, so -7 // 2 is -4; Python's modulo has the sign of the divisor, so -7 % 3 is 2.", level=3),
        task("c1",
             """
             `price` and `qty` are arrays of equal length. Compute `revenue`
             (price times quantity for each item) and `log_price` (the natural log
             of each price) without a Python loop.
             """,
             """
             import numpy as np
             revenue = price * qty
             log_price = np.log(price)
             """,
             [
                 check("revenue", 'check(close(revenue, price * qty), "`revenue` should be price times qty for each item.")'),
                 check("log_price", """
                     import numpy as np
                     check(close(log_price, np.log(price)), "`log_price` should be np.log of each price.")
                 """),
                 check("no loop", 'check(" for " not in SOURCE and "while" not in SOURCE, "Do it without a Python loop: use array arithmetic and np.log.")'),
             ],
             "Arithmetic operators and `np.log` work on whole arrays at once.",
             starter="""
             import numpy as np
             revenue = []
             for i in range(len(price)):
                 revenue.append(price[i] * qty[i])
             log_price = ...
             """,
             setup="import numpy as np\nprice = np.array([4.5, 12.0, 7.25, 30.0])\nqty = np.array([10, 3, 8, 1])"),
        task("c2",
             """
             Write a function `compound(principal, rate, years)` that returns the
             value of `principal` after `years` of annual compounding at `rate`.
             It must work when `years` is a NumPy array, returning an array of values.
             """,
             """
             def compound(principal, rate, years):
                 return principal * (1 + rate) ** years
             """,
             [
                 check("scalar", 'check(close(compound(100, 0.05, 2), 110.25), "compound(100, 0.05, 2) should be 110.25.")'),
                 check("array", """
                     import numpy as np
                     got = compound(1000, 0.1, np.arange(4))
                     check(close(got, [1000, 1100, 1210, 1331]), "With years = np.arange(4) the result should be [1000, 1100, 1210, 1331].")
                 """),
             ],
             "Because `**` is a ufunc, the same expression works for a number and for an array of years.",
             starter="""
             def compound(principal, rate, years):
                 ...
             """),
    ]),

concept(
    "py.np.ufunc_methods", "Ufunc methods: out, reduce, accumulate, outer", ["py.np.ufuncs", "py.np.reshape"],
    "2.03", "Computation on NumPy Arrays: Universal Functions", "02.03-computation-on-arrays-ufuncs",
    """
    Binary ufuncs such as `np.add` and `np.multiply` have methods that turn
    them into other tools:

    ```python
    x = np.array([2, 3, 4])
    np.add.reduce(x)          # 9       (combine everything: a sum)
    np.multiply.reduce(x)     # 24      (a product)
    np.add.accumulate(x)      # [2 5 9] (running totals)
    np.multiply.outer(x, x)   # 3 x 3 table of every pair's product
    ```

    The `out=` argument writes the result into an existing array instead of
    allocating a new one, which matters for very large arrays:

    ```python
    buf = np.empty(3)
    np.multiply(x, 10, out=buf)
    ```

    `np.cumsum` and `np.cumprod` are shortcuts for the two accumulations.
    """,
    [
        mc("m1", "What does `np.multiply.accumulate([1, 2, 3, 4])` return?",
           "`[ 1  2  6 24]`", ["`24`", "`[1 2 3 4]`", "`[ 1  3  6 10]`"],
           "accumulate keeps every intermediate result of the running product.", level=1),
        mc("m2", "Gross returns for three years are `g = np.array([1.1, 0.95, 1.2])`. Which gives total growth over the whole period?",
           "`np.multiply.reduce(g)`", ["`np.add.reduce(g)`", "`np.multiply.outer(g, g)`", "`np.add.accumulate(g)`"],
           "Compounded growth is the product of gross returns, which is what multiply.reduce computes.", level=2),
        mc("m3", "What is the shape of `np.add.outer(np.arange(3), np.arange(5))`?",
           "`(3, 5)`", ["`(5, 3)`", "`(8,)`", "`(15,)`"],
           "outer pairs every element of the first input with every element of the second: 3 rows, 5 columns.", level=2),
        out("o1", "What does this print?",
            """
            import numpy as np
            print(np.add.accumulate(np.array([5, -2, 4])))
            """,
            "Running totals: 5, 5-2, 5-2+4.", level=1),
        out("o2", "What does this print?",
            """
            import numpy as np
            print(np.multiply.outer([1, 2], [10, 20, 30]))
            """,
            "Row i, column j holds a[i] * b[j].", level=2),
        task("c1",
             """
             `flows` holds monthly net cash flows. Compute `balance`, the running
             balance after each month starting from zero, and `table`, the 1-12
             multiplication table (a 12 x 12 array whose entry [i, j] is (i+1)*(j+1)).
             Use ufunc methods for both.
             """,
             """
             import numpy as np
             balance = np.add.accumulate(flows)
             n = np.arange(1, 13)
             table = np.multiply.outer(n, n)
             """,
             [
                 check("balance", 'check(close(balance, [200, 150, 450, 430, 630]), "`balance` should be the running total of flows.")'),
                 check("table shape", 'check(getattr(table, "shape", None) == (12, 12), "`table` should be 12 x 12.")'),
                 check("table values", 'check(table[2, 4] == 15 and table[11, 11] == 144 and table[0, 0] == 1, "Entry [i, j] should be (i+1)*(j+1).")'),
             ],
             "`np.add.accumulate` (or `np.cumsum`) gives running totals; `np.multiply.outer` builds the table in one call.",
             starter="""
             import numpy as np
             balance = ...
             table = ...
             """,
             setup="import numpy as np\nflows = np.array([200, -50, 300, -20, 200])"),
    ]),

# ---------------------------------------------------------------- 2.04
concept(
    "py.np.aggregates", "Aggregations and the axis argument", ["py.np.ufuncs", "py.np.attributes"],
    "2.04", "Aggregations: Min, Max, and Everything in Between", "02.04-computation-on-arrays-aggregates",
    """
    Aggregations summarise an array: `sum`, `mean`, `std`, `min`, `max`,
    `argmin`, `argmax`, `median`, `percentile`. Call them as functions
    (`np.mean(x)`) or methods (`x.mean()`).

    The `axis` argument says which axis to **collapse**:

    ```python
    sales = np.array([[3, 5, 2],     # store 0, three products
                      [4, 1, 6]])    # store 1
    sales.sum()          # 21, everything
    sales.sum(axis=0)    # [7 6 8]  collapse rows -> one total per product
    sales.sum(axis=1)    # [10 11]  collapse columns -> one total per store
    ```

    Missing data stored as `np.nan` poisons ordinary aggregates (`np.mean`
    returns nan); the `nan`-versions (`np.nanmean`, `np.nansum`, ...) skip it.
    Prefer `np.sum` over Python's built-in `sum` on arrays: it is faster and
    understands `axis`.
    """,
    [
        mc("m1", "`m` has shape `(4, 7)`. What is the shape of `m.mean(axis=0)`?",
           "`(7,)`", ["`(4,)`", "`(4, 7)`", "a single number"],
           "axis=0 is collapsed, leaving one value per column.", level=1),
        mc("m2", "`x = np.array([2.0, np.nan, 4.0])`. What is `np.nanmean(x)`?",
           "3.0", ["nan", "2.0", "2.0, because nan counts as zero"],
           "nanmean ignores the missing value and averages 2 and 4.", level=1),
        mc("m3", "What does `np.argmax(x)` return?",
           "The index of the largest element", ["The largest element", "The number of maxima", "A boolean mask of the maxima"],
           "The `arg` prefix means *position*. Use it to find which month had the peak, not what the peak was.", level=1),
        out("o1", "What does this print?",
            """
            import numpy as np
            m = np.array([[1, 8, 3],
                          [6, 2, 9]])
            print(m.max(axis=1))
            """,
            "axis=1 collapses the columns: the largest value in each row.", level=1),
        out("o2", "What does this print?",
            """
            import numpy as np
            x = np.array([4, 9, 1, 9])
            print(x.argmin(), x.argmax())
            """,
            "argmin is the position of 1; argmax returns the first position of the maximum.", level=2),
        out("o3", "What does this print?",
            """
            import numpy as np
            x = np.array([1.0, 2.0, np.nan])
            print(np.sum(x), np.nansum(x))
            """,
            "Any arithmetic with nan gives nan; nansum treats it as absent.", level=2),
        task("c1",
             """
             `sales` has one row per store and one column per month. Compute
             `store_totals` (one value per store), `month_means` (average across
             stores for each month) and `best_month` (the column index with the
             largest total across stores).
             """,
             """
             store_totals = sales.sum(axis=1)
             month_means = sales.mean(axis=0)
             best_month = sales.sum(axis=0).argmax()
             """,
             [
                 check("store_totals", 'check(close(store_totals, sales.sum(axis=1)) and len(store_totals) == 4, "`store_totals` should have one value per store (sum across columns).")'),
                 check("month_means", 'check(close(month_means, sales.mean(axis=0)) and len(month_means) == 6, "`month_means` should have one value per month.")'),
                 check("best_month", 'check(int(best_month) == 3, "`best_month` should be the index of the month with the largest total.")'),
             ],
             "axis=1 collapses months (giving per-store values); axis=0 collapses stores (per-month values). argmax gives the position.",
             starter="""
             print(sales.shape)
             store_totals = ...
             month_means = ...
             best_month = ...
             """,
             setup="import numpy as np\nsales = np.array([[5, 3, 8, 9, 2, 4], [6, 2, 7, 12, 3, 5], [4, 4, 6, 10, 1, 6], [7, 5, 9, 11, 2, 3]])"),
        task("c2",
             """
             `temps` contains some missing readings stored as `np.nan`. Compute
             `avg`, the mean of the readings that are present, and `n_missing`, how
             many are missing.
             """,
             """
             import numpy as np
             avg = np.nanmean(temps)
             n_missing = np.isnan(temps).sum()
             """,
             [
                 check("avg", 'check(close(avg, 21.5), "`avg` should ignore the missing readings (try np.nanmean).")'),
                 check("n_missing", 'check(int(n_missing) == 2, "`n_missing` should count the nan values (np.isnan helps).")'),
             ],
             "`np.nanmean` skips nan. `np.isnan` returns True where a value is missing, and summing booleans counts the Trues.",
             starter="""
             import numpy as np
             avg = np.mean(temps)
             n_missing = ...
             """,
             setup="import numpy as np\ntemps = np.array([20.0, np.nan, 22.0, 23.0, np.nan, 21.0])"),
    ]),

# ---------------------------------------------------------------- 2.05
concept(
    "py.np.broadcast_rules", "Broadcasting rules", ["py.np.ufuncs", "py.np.reshape"],
    "2.05", "Computation on Arrays: Broadcasting", "02.05-computation-on-arrays-broadcasting",
    """
    Broadcasting lets arrays of different shapes combine without copying
    data. NumPy compares shapes from the **right**:

    1. If the arrays have different numbers of dimensions, pad the shorter
       shape with 1s on the **left**.
    2. Two dimensions are compatible if they are equal or one of them is 1.
       A 1 is stretched to match.
    3. If any pair is neither equal nor 1, it is an error.

    Worked example: `(3, 4)` with `(4,)`. Pad: `(1, 4)`. Compare: 3 vs 1
    stretches, 4 vs 4 matches. Result `(3, 4)`: the row is added to every row.

    `(3, 4)` with `(3,)` fails: padding gives `(1, 3)` and 4 vs 3 clash. To
    add one value per *row*, make it a column first: `v[:, np.newaxis]` has
    shape `(3, 1)`, which broadcasts.
    """,
    [
        mc("m1", "What is the shape of `np.ones((5, 1)) + np.ones((1, 3))`?",
           "`(5, 3)`", ["`(5, 1)`", "`(1, 3)`", "It raises an error"],
           "Each 1 stretches to match the other array's length on that axis.", level=1),
        mc("m2", "Which pair of shapes can NOT be broadcast together?",
           "`(4, 3)` and `(4,)`", ["`(4, 3)` and `(3,)`", "`(4, 3)` and `(4, 1)`", "`(2, 1, 3)` and `(5, 1)`"],
           "`(4,)` pads to `(1, 4)`, and its 4 clashes with the 3 columns. The others are compatible.", level=2),
        mc("m3", "What is the shape of `np.zeros((2, 1, 3)) * np.zeros((5, 1))`?",
           "`(2, 5, 3)`", ["`(2, 5, 1)`", "It raises an error", "`(5, 3)`"],
           "Pad `(5, 1)` to `(1, 5, 1)`. Compare axis by axis: 2 vs 1, 1 vs 5, 3 vs 1. Every pair has a 1, giving `(2, 5, 3)`.", level=3),
        out("o1", "What does this print?",
            """
            import numpy as np
            a = np.arange(3)
            b = np.arange(3)[:, np.newaxis]
            print(a + b)
            """,
            "A row `(3,)` plus a column `(3, 1)` broadcasts to a 3 x 3 table of sums.", level=2),
        out("o2", "What does this print?",
            """
            import numpy as np
            m = np.ones((2, 3))
            print((m * [1, 2, 3]).sum(axis=0))
            """,
            "The list `[1, 2, 3]` multiplies each row; summing two identical rows doubles it.", level=2),
        task("c1",
             """
             `m` has shape `(4, 3)` and `row_w` has shape `(3,)`, `col_w` has shape `(4,)`.
             Make `scaled_cols`, each column of `m` multiplied by the matching entry of
             `row_w`, and `scaled_rows`, each row of `m` multiplied by the matching
             entry of `col_w`.
             """,
             """
             import numpy as np
             scaled_cols = m * row_w
             scaled_rows = m * col_w[:, np.newaxis]
             """,
             [
                 check("scaled_cols", 'check(close(scaled_cols, m_ref * np.array([1, 10, 100])), "`scaled_cols` should multiply column j by row_w[j].")'),
                 check("scaled_rows", 'check(close(scaled_rows, m_ref * np.array([[1], [2], [3], [4]])), "`scaled_rows` should multiply row i by col_w[i]. A (4,) array needs to become a (4, 1) column.")'),
             ],
             "A `(3,)` array lines up with the last axis (columns) directly. A `(4,)` array must become a `(4, 1)` column to line up with the rows.",
             starter="""
             import numpy as np
             scaled_cols = ...
             scaled_rows = m * col_w
             """,
             setup="import numpy as np\nm = np.arange(12, dtype=float).reshape(4, 3)\nm_ref = m.copy()\nrow_w = np.array([1, 10, 100])\ncol_w = np.array([1, 2, 3, 4])"),
    ]),

concept(
    "py.np.broadcast_practice", "Broadcasting in practice", ["py.np.broadcast_rules", "py.np.aggregates"],
    "2.05", "Computation on Arrays: Broadcasting", "02.05-computation-on-arrays-broadcasting",
    """
    Two everyday uses:

    **Standardising columns.** With observations in rows and variables in
    columns, `X.mean(axis=0)` has shape `(k,)`, which broadcasts against
    `X` of shape `(n, k)`:

    ```python
    Z = (X - X.mean(axis=0)) / X.std(axis=0)
    ```

    Each column now has mean 0 and standard deviation 1, the scaling many
    estimators expect.

    **Evaluating a function on a grid.** A row of x values and a column of y
    values broadcast to every (x, y) pair:

    ```python
    x = np.linspace(0, 1, 50)
    y = np.linspace(0, 1, 40)[:, np.newaxis]
    profit = 3 * x - x**2 + 2 * y - y**2   # shape (40, 50)
    ```

    Pairwise distances, demeaning within groups and outer products all follow
    the same pattern: line the axes up, then let broadcasting do the loop.
    """,
    [
        mc("m1", "`X` has shape `(200, 5)`. Which line demeans every column?",
           "`X - X.mean(axis=0)`", ["`X - X.mean(axis=1)`", "`X - X.mean()`", "`X.mean(axis=0) - X.T`"],
           "`X.mean(axis=0)` has shape `(5,)`, one mean per column, and broadcasts across all 200 rows.", level=1),
        mc("m2", "`X` has shape `(200, 5)`. Why does `X - X.mean(axis=1)` fail?",
           "The row means have shape `(200,)`, which pads to `(1, 200)` and clashes with 5 columns",
           ["Means cannot be subtracted from arrays", "axis=1 is not allowed for mean", "It does not fail; it demeans the columns"],
           "To demean rows, keep the axis: `X.mean(axis=1, keepdims=True)` has shape `(200, 1)`.", level=2),
        mc("m3", "`a` has shape `(n, 2)` (points) and `b` has shape `(m, 2)`. What shape is `a[:, None, :] - b[None, :, :]`?",
           "`(n, m, 2)`", ["`(n, 2)`", "`(m, n, 2)`", "It raises an error"],
           "The inserted axes make `(n, 1, 2)` and `(1, m, 2)`, which broadcast to every pair of points.", level=3),
        out("o1", "What does this print?",
            """
            import numpy as np
            X = np.array([[1.0, 10.0],
                          [3.0, 30.0]])
            print(X - X.mean(axis=0))
            """,
            "Column means are 2 and 20; subtracting them centres each column.", level=2),
        out("o2", "What does this print?",
            """
            import numpy as np
            x = np.arange(3)
            y = np.arange(2)[:, None]
            print((x * y).shape)
            """,
            "A `(3,)` row and a `(2, 1)` column broadcast to `(2, 3)`.", level=1),
        task("c1",
             """
             `X` has one row per household and one column per variable. Make `Z`,
             the standardised version in which every column has mean 0 and
             (population) standard deviation 1. Do not loop over columns.
             """,
             """
             Z = (X - X.mean(axis=0)) / X.std(axis=0)
             """,
             [
                 check("shape", 'check(getattr(Z, "shape", None) == X.shape, "`Z` should have the same shape as X.")'),
                 check("means", 'check(close(Z.mean(axis=0), 0, atol=1e-9), "Every column of Z should have mean 0.")'),
                 check("sds", 'check(close(Z.std(axis=0), 1), "Every column of Z should have standard deviation 1.")'),
                 check("no loop", 'check(" for " not in SOURCE, "Use broadcasting instead of a loop.")'),
             ],
             "`X.mean(axis=0)` and `X.std(axis=0)` each have one value per column and broadcast over the rows.",
             starter="""
             Z = ...
             """,
             setup="import numpy as np\nX = np.random.default_rng(3).normal([50, 2, 1000], [10, 0.5, 300], size=(40, 3))"),
        task("c2",
             """
             Evaluate `f(p, q) = p * q - p**2` on a grid: `p` takes 30 values from 0
             to 10 (use `np.linspace`) and `q` takes 20 values from 0 to 5. Store the
             results in `F` with shape `(20, 30)`: rows follow `q`, columns follow `p`.
             """,
             """
             import numpy as np
             p = np.linspace(0, 10, 30)
             q = np.linspace(0, 5, 20)[:, np.newaxis]
             F = p * q - p**2
             """,
             [
                 check("shape", 'check(getattr(F, "shape", None) == (20, 30), "`F` should have shape (20, 30).")'),
                 check("values", """
                     import numpy as np
                     p = np.linspace(0, 10, 30); q = np.linspace(0, 5, 20)
                     check(close(F[7, 11], p[11] * q[7] - p[11]**2), "F[i, j] should be f(p[j], q[i]).")
                 """),
             ],
             "Make `q` a column so it lines up with rows, and leave `p` as a row. Broadcasting fills in every pair.",
             starter="""
             import numpy as np
             p = ...
             q = ...
             F = ...
             """),
    ]),

# ---------------------------------------------------------------- 2.06
concept(
    "py.np.comparisons", "Comparisons and counting with booleans", ["py.np.aggregates"],
    "2.06", "Comparisons, Masks, and Boolean Logic", "02.06-boolean-arrays-and-masks",
    """
    Comparison operators are ufuncs too. `x > 3` returns a **boolean array**
    of the same shape:

    ```python
    rain = np.array([0.0, 2.1, 0.0, 5.4, 0.3])
    rain > 1            # [False  True False  True False]
    ```

    Booleans count as 0 and 1, so aggregates answer questions directly:

    ```python
    np.sum(rain > 1)          # 2 days above 1 mm
    np.mean(rain > 0)         # 0.6: share of days with any rain
    np.any(rain > 5)          # True
    np.all(rain >= 0)         # True
    np.count_nonzero(rain)    # 3
    ```

    Combine conditions with `&` (and), `|` (or) and `~` (not), and wrap each
    comparison in parentheses: `(rain > 0) & (rain < 1)`.
    """,
    [
        mc("m1", "What does `np.mean(x > 0)` measure?",
           "The share of elements that are positive", ["The mean of the positive elements", "The number of positive elements", "Whether all elements are positive"],
           "True counts as 1, so the mean of a boolean array is the fraction of Trues.", level=1),
        mc("m2", "Why does `(x > 2) & (x < 5)` need the parentheses?",
           "`&` binds more tightly than `>` and `<`, so without them NumPy evaluates `2 & x` first",
           ["Parentheses make it faster", "They are optional style", "`&` only works on lists"],
           "Operator precedence puts `&` above comparisons. Always parenthesise each comparison.", level=2),
        mc("m3", "`np.all(m > 0, axis=1)` on a 2-D array returns",
           "One boolean per row: whether every value in that row is positive",
           ["One boolean per column", "A single boolean", "The positive values in each row"],
           "Like `sum`, `all` and `any` take an axis to collapse.", level=2),
        out("o1", "What does this print?",
            """
            import numpy as np
            x = np.array([3, 8, 1, 9, 4])
            print(np.sum((x > 2) & (x < 9)))
            """,
            "3, 8 and 4 satisfy both conditions.", level=1),
        out("o2", "What does this print?",
            """
            import numpy as np
            x = np.array([0, 2, 0, 7])
            print(np.count_nonzero(x), np.any(x > 6), np.all(x > 0))
            """,
            "Two non-zero values; 7 exceeds 6; the zeros are not positive.", level=1),
        task("c1",
             """
             `income` holds household incomes. Compute `n_low` (how many are below
             20,000), `share_middle` (the fraction between 20,000 and 80,000
             inclusive) and `any_negative` (whether any value is below zero).
             """,
             """
             import numpy as np
             n_low = np.sum(income < 20000)
             share_middle = np.mean((income >= 20000) & (income <= 80000))
             any_negative = np.any(income < 0)
             """,
             [
                 check("n_low", 'check(int(n_low) == 3, "`n_low` should count incomes below 20,000.")'),
                 check("share_middle", 'check(close(share_middle, 0.5), "`share_middle` should be the fraction from 20,000 to 80,000 inclusive.")'),
                 check("any_negative", 'check(bool(any_negative) is True, "`any_negative` should be True when any income is below zero.")'),
             ],
             "Sum a boolean array to count, take its mean for a share, and use `np.any` for 'is there at least one'.",
             starter="""
             import numpy as np
             n_low = ...
             share_middle = ...
             any_negative = ...
             """,
             setup="import numpy as np\nincome = np.array([12000, 45000, 80000, 95000, 20000, -500, 150000, 60000, 18000, 33000])"),
    ]),

concept(
    "py.np.masks", "Boolean masks for selection", ["py.np.comparisons", "py.np.indexing"],
    "2.06", "Comparisons, Masks, and Boolean Logic", "02.06-boolean-arrays-and-masks",
    """
    A boolean array used as an index keeps the elements where it is True. This
    is **masking**:

    ```python
    wage = np.array([14.0, 32.5, 9.0, 21.0])
    wage[wage > 15]            # [32.5 21. ]
    wage[wage < 10] = 10.0     # raise everything below 10 to the floor
    ```

    The result of a mask selection is a 1-D array, whatever the input shape,
    and it is a copy, not a view.

    Use `&` `|` `~` on arrays, never `and` `or` `not`. The keywords ask whether
    a whole object is true, which is ambiguous for an array with several
    elements and raises an error. `np.where(cond, a, b)` picks from `a` where
    the condition holds and from `b` elsewhere.
    """,
    [
        mc("m1", "What happens with `x[(x > 0) and (x < 5)]` for an array `x` of several elements?",
           "A `ValueError`: the truth value of an array is ambiguous",
           ["It works like `&`", "It returns the first matching element", "It returns an empty array"],
           "`and` needs a single True or False. Use `&` for element-wise logic.", level=2),
        mc("m2", "`m` has shape `(3, 4)`. What is the shape of `m[m > 0]` when 5 entries are positive?",
           "`(5,)`", ["`(3, 4)`", "`(3, 4)` with zeros elsewhere", "`(5, 1)`"],
           "Masking flattens: you get the selected values in a 1-D array.", level=2),
        mc("m3", "What does `np.where(x < 0, 0, x)` return?",
           "`x` with negative values replaced by 0", ["The positions of the negative values", "Only the negative values", "A boolean array"],
           "With three arguments, `np.where` chooses element by element between the second and third.", level=1),
        out("o1", "What does this print?",
            """
            import numpy as np
            x = np.array([5, -2, 7, -9, 3])
            print(x[x < 0])
            """,
            "The mask keeps the negative entries, in order.", level=1),
        out("o2", "What does this print?",
            """
            import numpy as np
            x = np.arange(6)
            x[x % 2 == 1] = 0
            print(x)
            """,
            "Assigning through a mask changes the original array: odd entries become 0.", level=1),
        out("o3", "What does this print?",
            """
            import numpy as np
            x = np.array([4, 15, 8, 23])
            print(np.where(x > 10, 'high', 'low'))
            """,
            "Arrays of strings print with quotes inside the brackets.", level=2),
        task("c1",
             """
             `age` and `earnings` describe the same people. Set `prime_age` to the
             earnings of people aged 25 to 54 inclusive, and replace every negative
             earning in `earnings` with 0 (in place).
             """,
             """
             prime_age = earnings[(age >= 25) & (age <= 54)]
             earnings[earnings < 0] = 0
             """,
             [
                 check("prime_age", 'check(close(sorted(prime_age), sorted([52000, 31000, -1000, 40000])) or close(sorted(prime_age), sorted([52000, 31000, 0, 40000])), "`prime_age` should hold earnings for ages 25 to 54 inclusive.")'),
                 check("clipped", 'check(earnings.min() >= 0 and close(earnings.sum(), 52000 + 31000 + 18000 + 40000 + 70000), "Every negative earning should now be 0, and nothing else should change.")'),
             ],
             "Combine both age conditions with `&`; assign through a mask to change only the matching entries.",
             starter="""
             prime_age = ...
             """,
             setup="import numpy as np\nage = np.array([22, 30, 45, 54, 60, 25, 19])\nearnings = np.array([-500, 52000, 31000, -1000, 70000, 40000, 18000])"),
        task("c2",
             """
             Create `capped`, a copy of `bids` where every bid above 100 is replaced
             by 100 and the rest are unchanged. Use `np.where`.
             """,
             """
             import numpy as np
             capped = np.where(bids > 100, 100, bids)
             """,
             [
                 check("values", 'check(same_array(capped, [50, 100, 99, 100, 100]), "`capped` should cap values at 100.")'),
                 check("original", 'check(same_array(bids, [50, 120, 99, 100, 300]), "Leave `bids` unchanged.")'),
                 check("where", 'check(uses("where"), "Use np.where for this task.")'),
             ],
             "`np.where(cond, if_true, if_false)` builds a new array without touching the original.",
             starter="""
             import numpy as np
             capped = ...
             """,
             setup="import numpy as np\nbids = np.array([50, 120, 99, 100, 300])"),
    ]),

# ---------------------------------------------------------------- 2.07
concept(
    "py.np.fancy", "Fancy indexing with index arrays", ["py.np.slicing", "py.np.reshape"],
    "2.07", "Fancy Indexing", "02.07-fancy-indexing",
    """
    Pass an array (or list) of positions to pull out several elements at once:

    ```python
    x = np.array([10, 20, 30, 40, 50])
    x[[4, 0, 0]]          # [50 10 10]: any order, repeats allowed
    ```

    The result takes the **shape of the index**, not of `x`:

    ```python
    idx = np.array([[0, 1], [3, 4]])
    x[idx]                # [[10 20] [40 50]]
    ```

    In two dimensions, index arrays for rows and columns pair up element by
    element (and broadcast): `m[[0, 2], [1, 3]]` is `[m[0, 1], m[2, 3]]`.
    Mix with slices to take whole rows: `m[[2, 0], :]` reorders rows. A random
    subsample is `X[rng.choice(len(X), size=10, replace=False)]`.

    Unlike slicing, fancy indexing returns a copy.
    """,
    [
        mc("m1", "For `x = np.arange(10, 20)`, what is `x[[1, 1, 9]]`?",
           "`[11 11 19]`", ["`[11 19]`", "`[1 1 9]`", "an `IndexError` because of the repeat"],
           "Each index is looked up independently, so repeats return repeated values.", level=1),
        mc("m2", "`m` has shape `(4, 5)`. What is `m[[0, 3], [1, 4]]`?",
           "A 1-D array of two values: `m[0, 1]` and `m[3, 4]`",
           ["The 2 x 2 block of rows 0, 3 and columns 1, 4", "Rows 0 to 3 and columns 1 to 4", "An error"],
           "Two index arrays pair up position by position. For the block, use `m[np.ix_([0, 3], [1, 4])]` or `m[[0, 3]][:, [1, 4]]`.", level=3),
        mc("m3", "`x` has 100 elements and `idx` has shape `(5, 2)`. What is the shape of `x[idx]`?",
           "`(5, 2)`", ["`(100,)`", "`(10,)`", "`(5,)`"],
           "Fancy indexing returns an array shaped like the index.", level=2),
        out("o1", "What does this print?",
            """
            import numpy as np
            m = np.arange(12).reshape(3, 4)
            print(m[[2, 0], 1:3])
            """,
            "Rows 2 and 0 (in that order), columns 1 and 2.", level=2),
        out("o2", "What does this print?",
            """
            import numpy as np
            x = np.array([7, 3, 9])
            order = np.array([1, 2, 0])
            print(x[order])
            """,
            "Position 1 first, then 2, then 0.", level=1),
        task("c1",
             """
             `X` has one row per respondent. Draw a sample of 10 distinct rows with
             `rng.choice` (no replacement) using the generator `rng` already
             created, and store them in `sample`. Also store the drawn row numbers in `rows`.
             """,
             """
             rows = rng.choice(len(X), size=10, replace=False)
             sample = X[rows]
             """,
             [
                 check("rows", """
                     import numpy as np
                     check(len(rows) == 10 and len(np.unique(rows)) == 10, "`rows` should be 10 distinct row numbers.")
                 """),
                 check("sample", 'check(same_array(sample, X[rows]), "`sample` should be the rows of X at positions `rows`.")'),
                 check("shape", 'check(getattr(sample, "shape", None) == (10, 4), "`sample` should be 10 x 4.")'),
             ],
             "Draw positions with `rng.choice(n, size, replace=False)` and use them as a fancy index on the rows.",
             starter="""
             rows = ...
             sample = ...
             """,
             setup="import numpy as np\nrng = np.random.default_rng(5)\nX = np.arange(400).reshape(100, 4)"),
    ]),

concept(
    "py.np.fancy_modify", "Modifying values with fancy indexing", ["py.np.fancy", "py.np.ufunc_methods"],
    "2.07", "Fancy Indexing", "02.07-fancy-indexing",
    """
    Fancy indexes work on the left of `=` too:

    ```python
    x = np.zeros(5)
    x[[1, 3]] = 7          # [0. 7. 0. 7. 0.]
    ```

    The trap: with **repeated** indices, `x[idx] += 1` does not add once per
    occurrence. NumPy computes `x[idx] + 1` and then assigns, so a repeated
    position is just written twice with the same value.

    ```python
    counts = np.zeros(3)
    counts[[0, 0, 2]] += 1     # [1. 0. 1.]  not [2. 0. 1.]
    np.add.at(counts, [0, 0, 2], 1)   # unbuffered: really adds per occurrence
    ```

    `np.add.at` is how you accumulate into bins. Paired with
    `np.searchsorted` (which finds the bin each value falls into) it makes a
    small histogram by hand; `np.bincount` does the integer case directly.
    """,
    [
        mc("m1", "After `x = np.zeros(4); x[[2, 2, 2]] += 1`, what is `x[2]`?",
           "1.0", ["3.0", "0.0", "2.0"],
           "The buffered `+=` writes `0 + 1` to position 2 three times. Use `np.add.at` to count every occurrence.", level=2),
        mc("m2", "Which line increments `counts` once for every entry in `idx`, including repeats?",
           "`np.add.at(counts, idx, 1)`", ["`counts[idx] += 1`", "`counts[idx] = counts[idx] + 1`", "`counts += idx`"],
           "`ufunc.at` applies the operation unbuffered, once per index.", level=1),
        mc("m3", "`np.searchsorted(edges, values)` returns",
           "For each value, the position in the sorted `edges` where it would be inserted",
           ["The sorted values", "The edges that equal a value", "A boolean mask"],
           "Insert positions are bin numbers, which is why it pairs with `np.add.at` for binning.", level=3),
        out("o1", "What does this print?",
            """
            import numpy as np
            x = np.zeros(4, dtype=int)
            np.add.at(x, [0, 3, 3, 3], 1)
            print(x)
            """,
            "add.at counts every occurrence, so position 3 gets three increments.", level=1),
        out("o2", "What does this print?",
            """
            import numpy as np
            print(np.bincount(np.array([1, 1, 4, 2, 1])))
            """,
            "bincount gives the count of 0, 1, 2, ... up to the largest value.", level=2),
        task("c1",
             """
             `scores` holds exam scores from 0 to 100. Build `hist`, an integer
             array of 5 counts for the bins [0, 20), [20, 40), [40, 60), [60, 80)
             and [80, 100]. Use `np.searchsorted` to find each score's bin and
             `np.add.at` to count.
             """,
             """
             import numpy as np
             edges = np.array([20, 40, 60, 80])
             bins = np.searchsorted(edges, scores, side="right")
             hist = np.zeros(5, dtype=int)
             np.add.at(hist, bins, 1)
             """,
             [
                 check("counts", 'check(same_array(hist, [2, 3, 2, 4, 3]), "`hist` should count scores per bin. A score of exactly 20 belongs to the second bin.")'),
                 check("tools", 'check(uses("searchsorted") and uses("add.at"), "Use np.searchsorted and np.add.at.")'),
             ],
             "With `side='right'`, a score equal to an edge goes into the bin that starts at that edge, which matches half-open bins.",
             starter="""
             import numpy as np
             edges = np.array([20, 40, 60, 80])
             hist = ...
             """,
             setup="import numpy as np\nscores = np.array([5, 19, 20, 35, 39.5, 40, 59, 60, 61, 75, 79, 80, 99, 100])", level=3),
    ]),

# ---------------------------------------------------------------- 2.08
concept(
    "py.np.sorting", "Sorting and argsort", ["py.np.fancy", "py.np.aggregates"],
    "2.08", "Sorting Arrays", "02.08-sorting",
    """
    `np.sort(x)` returns a sorted copy; `x.sort()` sorts in place. Both
    default to ascending order; for descending, reverse: `np.sort(x)[::-1]`.

    `np.argsort(x)` returns the **positions** that would sort `x`. Its real
    use is sorting one array by another:

    ```python
    names = np.array(["Lee", "Ahn", "Diaz"])
    gdp = np.array([3.1, 1.2, 2.4])
    order = np.argsort(gdp)
    names[order]          # ['Ahn' 'Diaz' 'Lee'], by GDP
    ```

    With `axis`, each row or column is sorted independently, which breaks the
    link between columns of the same row. To sort whole rows by one column,
    use `X[np.argsort(X[:, k])]`.
    """,
    [
        mc("m1", "What does `np.argsort(np.array([30, 10, 20]))` return?",
           "`[1 2 0]`", ["`[2 0 1]`", "`[10 20 30]`", "`[0 1 2]`"],
           "The smallest value 10 is at position 1, then 20 at 2, then 30 at 0.", level=1),
        mc("m2", "How do you sort the rows of `X` by its first column, keeping each row intact?",
           "`X[np.argsort(X[:, 0])]`", ["`np.sort(X, axis=0)`", "`np.sort(X[:, 0])`", "`X.sort(axis=1)`"],
           "`np.sort(X, axis=0)` sorts every column separately and scrambles rows. argsort on one column gives a row order to apply to the whole array.", level=2),
        mc("m3", "What is the difference between `np.sort(x)` and `x.sort()`?",
           "`np.sort` returns a sorted copy; `x.sort()` sorts `x` itself and returns None",
           ["None; they are aliases", "`x.sort()` is descending", "`np.sort` only works on 1-D arrays"],
           "A common bug is `x = x.sort()`, which sets `x` to None.", level=1),
        out("o1", "What does this print?",
            """
            import numpy as np
            x = np.array([4, 1, 3])
            y = np.array(['d', 'a', 'c'])
            print(y[np.argsort(x)[::-1]])
            """,
            "argsort gives ascending positions; reversing gives descending order of `x`.", level=2),
        out("o2", "What does this print?",
            """
            import numpy as np
            m = np.array([[3, 1],
                          [2, 5]])
            print(np.sort(m, axis=0))
            """,
            "Each column is sorted on its own: [3, 2] becomes [2, 3] and [1, 5] stays.", level=2),
        task("c1",
             """
             `countries` and `gdp` are matching arrays. Set `top3` to the names of
             the three countries with the largest GDP, largest first.
             """,
             """
             import numpy as np
             top3 = countries[np.argsort(gdp)[::-1][:3]]
             """,
             [
                 check("top3", 'check(list(top3) == ["E", "B", "F"], "`top3` should list the three largest by GDP, largest first.")'),
             ],
             "argsort ascending, reverse, take the first three positions, and use them to index the names.",
             starter="""
             import numpy as np
             top3 = ...
             """,
             setup="import numpy as np\ncountries = np.array(['A', 'B', 'C', 'D', 'E', 'F'])\ngdp = np.array([1.5, 7.2, 0.9, 3.3, 9.8, 5.1])"),
    ]),

concept(
    "py.np.partition", "Partial sorts and nearest neighbours", ["py.np.sorting", "py.np.broadcast_practice"],
    "2.08", "Sorting Arrays", "02.08-sorting",
    """
    When you only need the k smallest values, a full sort is wasted work.
    `np.partition(x, k)` moves the k smallest values to the front (in no
    particular order) and the rest after them; `np.argpartition` returns the
    positions instead.

    ```python
    x = np.array([9, 2, 7, 1, 5])
    np.partition(x, 2)[:2]      # the two smallest: 1 and 2, in some order
    ```

    Combined with broadcasting this gives nearest neighbours in a few lines.
    For points in rows of `P` (shape `(n, d)`):

    ```python
    diff = P[:, None, :] - P[None, :, :]       # (n, n, d)
    d2 = (diff ** 2).sum(axis=-1)              # squared distances (n, n)
    nearest = np.argpartition(d2, 1, axis=1)[:, :2]
    ```

    Each row's closest point is itself (distance 0), so the neighbour is the
    other entry. This is the core of k-nearest-neighbour methods.
    """,
    [
        mc("m1", "What does `np.partition(x, 3)` guarantee?",
           "The 3 smallest values come first, in no particular order, and position 3 holds the value a full sort would put there",
           ["`x` is fully sorted", "The first 3 elements are sorted", "The 3 largest values come first"],
           "Partition only separates the k smallest from the rest, which is faster than sorting.", level=2),
        mc("m2", "Why is the nearest point to each row of a distance matrix usually itself?",
           "Its distance to itself is 0, the smallest possible", ["argpartition always returns 0 first", "The matrix is sorted", "Broadcasting duplicates rows"],
           "Skip the diagonal (or take the second-smallest) to get the nearest *other* point.", level=2),
        mc("m3", "`P` has shape `(50, 2)`. What shape is `((P[:, None, :] - P[None, :, :]) ** 2).sum(-1)`?",
           "`(50, 50)`", ["`(50, 2)`", "`(50, 50, 2)`", "`(2500,)`"],
           "The difference is `(50, 50, 2)`; summing the last axis leaves one squared distance per pair.", level=3),
        out("o1", "What does this print?",
            """
            import numpy as np
            x = np.array([8, 3, 6, 1, 9])
            print(np.sort(np.partition(x, 2)[:2]))
            """,
            "The two smallest values are 1 and 3; sorting them makes the output definite.", level=2),
        task("c1",
             """
             `P` holds points in rows (shape `(n, 2)`). Compute `D2`, the `(n, n)`
             matrix of squared Euclidean distances using broadcasting, and
             `nearest`, an array whose entry i is the row number of the point
             closest to point i (not counting point i itself).
             """,
             """
             import numpy as np
             D2 = ((P[:, None, :] - P[None, :, :]) ** 2).sum(axis=-1)
             masked = D2 + np.diag(np.full(len(P), np.inf))
             nearest = masked.argmin(axis=1)
             """,
             [
                 check("D2", """
                     import numpy as np
                     ref = ((P[:, None, :] - P[None, :, :]) ** 2).sum(axis=-1)
                     check(close(D2, ref), "`D2[i, j]` should be the squared distance between points i and j.")
                 """),
                 check("nearest", 'check(same_array(nearest, [1, 0, 3, 2, 3]), "`nearest[i]` should be the closest *other* point to point i.")'),
             ],
             "Exclude self-matches by putting infinity on the diagonal before `argmin`, or take the second entry of an `argpartition`.",
             starter="""
             import numpy as np
             D2 = ...
             nearest = ...
             """,
             setup="import numpy as np\nP = np.array([[0.0, 0.0], [1.0, 0.0], [5.0, 5.0], [5.5, 5.0], [7.0, 6.0]])", level=3),
        mc("m4", "What does `np.argpartition(x, 2)` return?",
           "Positions into `x` arranged so the first two index its two smallest values, in no particular order",
           ["The two smallest values of `x`, sorted", "The positions that would fully sort `x`",
            "The two smallest values of `x`, in no particular order"],
           """
           `argpartition` is to `partition` what `argsort` is to `sort`: it
           returns indices, not values. Use them with fancy indexing, as in
           `x[np.argpartition(x, 2)[:2]]`.
           """, level=2),
        mc("m5", "Each row of the squared-distance matrix `D2` includes a point's distance to itself. To be sure of each point's K nearest *other* points, what should the partition position be in `np.argpartition(D2, ?, axis=1)`?",
           "`K + 1`", ["`K`", "`K - 1`", "`len(D2) - K`"],
           """
           The zero on the diagonal takes one of the smallest slots in every
           row, so you need the K + 1 smallest entries: the point itself plus
           its K neighbours.
           """, level=3),
        out("o2", "What does this print?",
            """
            import numpy as np
            X = np.array([[5, 1, 4],
                          [3, 9, 2]])
            p = np.partition(X, 1, axis=1)
            print(p[:, 1])
            """,
            """
            With `axis=1` each row is partitioned on its own. Position 1 always
            holds the value a full sort would put there, so the column is each
            row's second-smallest value: 4 in the first row and 3 in the second.
            """, level=2),
        task("c2",
             """
             `prices` holds the prices of ten products. Set `idx` to the
             positions of the three cheapest products, ordered from cheapest
             to dearest, and `cheapest3` to their prices in that order. Use
             `np.argpartition` to find the three before ordering them.
             """,
             """
             import numpy as np
             part = np.argpartition(prices, 3)[:3]
             idx = part[np.argsort(prices[part])]
             cheapest3 = prices[idx]
             """,
             [
                 check("idx", 'check(same_array(idx, [6, 2, 9]), "`idx` should be the positions of the three cheapest products, cheapest first.")'),
                 check("cheapest3", 'check(close(cheapest3, [1.25, 2.5, 3.0]), "`cheapest3` should be the three lowest prices in increasing order.")'),
             ],
             "Partition to get the three positions, then sort just those three by price with `argsort`.",
             starter="""
             import numpy as np
             idx = ...
             cheapest3 = ...
             """,
             setup="import numpy as np\nprices = np.array([9.5, 4.0, 2.5, 7.75, 5.0, 12.0, 1.25, 8.0, 6.5, 3.0])", level=2),
    ]),

# ---------------------------------------------------------------- 2.09
concept(
    "py.np.structured", "Structured arrays", ["py.np.dtypes", "py.np.masks"],
    "2.09", "Structured Data: NumPy's Structured Arrays", "02.09-structured-data-numpy",
    """
    A structured array gives each element named fields of different types,
    like rows of a small table:

    ```python
    dt = np.dtype([("name", "U10"), ("age", "i4"), ("wage", "f8")])
    people = np.array([("Ana", 34, 21.5), ("Ben", 51, 30.0)], dtype=dt)
    people["wage"]               # [21.5 30. ]
    people[people["age"] > 40]["name"]   # ['Ben']
    people[0]                    # one record: ('Ana', 34, 21.5)
    ```

    Type codes: `U10` is a Unicode string of up to 10 characters, `i4` a
    32-bit integer, `f8` a 64-bit float.

    Structured arrays are useful for reading binary formats and for fast
    record handling, but for everyday tabular work Pandas DataFrames (the next
    chapter) do this job with far more convenience.
    """,
    [
        mc("m1", "With `dt = np.dtype([('id', 'i4'), ('score', 'f8')])`, how do you get every score from a structured array `a`?",
           "`a['score']`", ["`a.score()`", "`a[:, 1]`", "`a[1]`"],
           "Fields are accessed by name in brackets. `a[1]` is the second record.", level=1),
        mc("m2", "What does the type code `'U8'` mean?",
           "A Unicode string of at most 8 characters", ["An unsigned 8-bit integer", "An 8-byte float", "A string of exactly 8 bytes"],
           "`U` is Unicode text; `u1` would be an unsigned 8-bit integer.", level=2),
        mc("m3", "For general tabular analysis with mixed column types, what does the book point you towards instead of structured arrays?",
           "Pandas DataFrames", ["Python dictionaries of lists", "Record arrays only", "Nested lists"],
           "Structured arrays are a low-level tool; DataFrames add labels, alignment and missing-data handling.", level=1),
        out("o1", "What does this print?",
            """
            import numpy as np
            dt = np.dtype([("city", "U10"), ("pop", "i8")])
            a = np.array([("Reno", 264), ("Elko", 20), ("Ely", 4)], dtype=dt)
            print(a[a["pop"] > 10]["city"])
            """,
            "The mask keeps records with population over 10; then the city field is taken.", level=2),
        task("c1",
             """
             Create a structured array `firms` with fields `name` (string, up to
             12 characters), `employees` (32-bit integer) and `revenue` (float)
             holding these three firms: ("Acme", 120, 3.5), ("Birch", 45, 1.25),
             ("Cobalt", 300, 9.0). Then set `big` to the names of firms with more
             than 100 employees.
             """,
             """
             import numpy as np
             dt = np.dtype([("name", "U12"), ("employees", "i4"), ("revenue", "f8")])
             firms = np.array([("Acme", 120, 3.5), ("Birch", 45, 1.25), ("Cobalt", 300, 9.0)], dtype=dt)
             big = firms[firms["employees"] > 100]["name"]
             """,
             [
                 check("fields", 'check(getattr(firms, "dtype", None) is not None and firms.dtype.names == ("name", "employees", "revenue"), "`firms` should have fields name, employees and revenue, in that order.")'),
                 check("types", 'check(firms.dtype["employees"].kind == "i" and firms.dtype["revenue"].kind == "f" and firms.dtype["name"].kind == "U", "Use a string, an integer and a float type for the three fields.")'),
                 check("values", 'check(close(firms["revenue"], [3.5, 1.25, 9.0]) and list(firms["name"]) == ["Acme", "Birch", "Cobalt"], "The three records should match the task.")'),
                 check("big", 'check(list(big) == ["Acme", "Cobalt"], "`big` should hold the names of firms with more than 100 employees.")'),
             ],
             "Define the dtype as a list of (name, type) pairs, then filter with a mask on one field and select another.",
             starter="""
             import numpy as np
             dt = ...
             firms = ...
             big = ...
             """, level=2),
        mc("m4", "What field names does `np.dtype('U10,i4,f8')` give the three fields?",
           "`f0`, `f1` and `f2`", ["`U10`, `i4` and `f8`", "`0`, `1` and `2`", "None; the fields can only be reached by position"],
           """
           A comma-separated string of type codes is a shortcut for when the
           names do not matter; NumPy numbers the fields `f0`, `f1`, `f2`.
           """, level=2),
        mc("m5", "After `rec = data.view(np.recarray)`, how can you read the `age` field?",
           "As an attribute, `rec.age`, as well as with `rec['age']`",
           ["Only with `rec['age']`", "Only with `rec.get('age')`", "As `rec[:, 'age']`"],
           """
           A record array is a structured array whose fields are also
           attributes. The convenience costs a little: attribute access is
           slower than indexing a plain structured array by field name.
           """, level=2),
        out("o2", "What does this print?",
            """
            import numpy as np
            tp = np.dtype([("id", "i8"), ("mat", "f8", (2, 2))])
            X = np.zeros(3, dtype=tp)
            print(X["mat"].shape)
            """,
            """
            A field can itself hold an array. Each of the three records carries
            a 2 by 2 block, so the `mat` field of the whole array has shape
            (3, 2, 2).
            """, level=3),
        task("c2",
             """
             Make `workers`, a structured array of length 4 created with
             `np.zeros` and a dtype given as a dictionary of `names`
             (`"name"`, `"age"`, `"wage"`) and `formats` (a Unicode string of
             up to 10 characters, a 32-bit integer and a 64-bit float). Fill
             its fields from the lists `names`, `ages` and `wages`. Then set
             `young_wage` to the mean wage of workers under 30.
             """,
             """
             import numpy as np
             workers = np.zeros(4, dtype={"names": ("name", "age", "wage"),
                                          "formats": ("U10", "i4", "f8")})
             workers["name"] = names
             workers["age"] = ages
             workers["wage"] = wages
             young_wage = workers[workers["age"] < 30]["wage"].mean()
             """,
             [
                 check("dtype", 'check(getattr(workers, "dtype", None) is not None and workers.dtype.names == ("name", "age", "wage"), "`workers` should have fields name, age and wage, in that order.")'),
                 check("formats", 'check(workers.dtype["name"].kind == "U" and workers.dtype["age"] == np.dtype("i4") and workers.dtype["wage"] == np.dtype("f8"), "Use U10, i4 and f8 for the three fields.")'),
                 check("filled", 'check(list(workers["name"]) == names and list(workers["age"]) == ages and close(workers["wage"], wages), "Copy the three lists into the fields.")'),
                 check("young_wage", 'check(close(young_wage, 16.5), "`young_wage` should be the mean wage of workers younger than 30.")'),
             ],
             "Assigning a list to a field fills that column; a mask on `age` then selects records whose `wage` you average.",
             starter="""
             import numpy as np
             workers = ...
             young_wage = ...
             """,
             setup="import numpy as np\nnames = ['Ana', 'Ben', 'Cy', 'Dee']\nages = [25, 45, 37, 19]\nwages = [18.0, 31.5, 27.0, 15.0]", level=2),
    ]),

    ],
}
