# Chapter 1 of the Python Data Science Handbook: IPython: Beyond Normal Python.
#
# Every question and note here is original. Each concept links to the book
# section it follows; the book's own text is CC-BY-NC-ND and is never adapted.
# Build with: python3 scripts/python_bank.py build ipython
#
# Learner code runs in plain Python, not IPython, with subprocesses and the
# network blocked. Magics, `?` and `!` commands are therefore taught through
# multiple-choice questions, and the runnable questions use the standard-library
# machinery underneath each feature (inspect, fnmatch, runpy, sys.displayhook,
# os, traceback, timeit, cProfile, tracemalloc).

CHAPTER = {
    "chapter": "ipython",
    "title": "IPython: Beyond Normal Python",
    "domain": "python_ipython",
    "concepts": [

# ---------------------------------------------------------------- 1.01
concept(
    "py.ipy.help", "Help, docstrings and source with ? and ??", [],
    "1.01", "Help and Documentation in IPython", "01.01-help-and-documentation",
    """
    In IPython you rarely leave the prompt to look something up. Put a `?`
    after any name to see its signature and **docstring**, the string literal
    written as the first statement of a function, class or module. Put `??`
    after it to see the **source code** as well, when that source is Python.

    ```python
    In [1]: def cpi_adjust(price, cpi, base=100):
       ...:     "Express a nominal price in base-period terms."
       ...:     return price * base / cpi

    In [2]: cpi_adjust?     # signature and docstring
    In [3]: cpi_adjust??    # the three lines above
    ```

    Underneath, these are ordinary Python: the docstring lives in
    `cpi_adjust.__doc__`, `help(cpi_adjust)` formats it, and the `inspect`
    module gives you `inspect.signature` and `inspect.getsource`. Writing a
    one-line docstring for your own functions means `?` will explain them to
    you a month later.
    """,
    [
        mc("m1", "In IPython, what does typing `round?` and pressing Enter do?",
           "Shows the signature and docstring of `round` without calling it",
           ["Calls `round()` with no arguments and shows the error",
            "Searches the web for documentation on `round`",
            "Prints the C source code that implements `round`"],
           """
           A trailing `?` asks IPython to describe the object. Nothing is
           called; you get its type, signature and docstring.
           """, level=1),
        mc("m2", "Why does `len??` show no more than `len?` does?",
           "`len` is implemented in C, so there is no Python source for IPython to display",
           ["`??` only works on functions you defined yourself in this session",
            "`??` is an alias for `?` in IPython 8",
            "Built-in names are hidden from introspection for security"],
           """
           `??` shows source when the object was written in Python. Built-ins
           such as `len` are compiled C, so IPython falls back to the same
           information `?` gives.
           """),
        mc("m3", "Where does the text that `?` shows for your own function come from?",
           "The string literal written as the first statement in the function body",
           ["A comment on the line above the `def`",
            "The function's name and argument names only",
            "A separate .txt file IPython looks up by function name"],
           """
           That first string becomes the function's `__doc__` attribute.
           Comments are discarded by the compiler and never reach `?`.
           """, level=1),
        out("o1", "What does this print?",
            '''
            import inspect

            def growth(old, new, periods=1):
                """Average growth rate per period."""
                return (new / old) ** (1 / periods) - 1

            print(inspect.signature(growth))
            print(growth.__doc__)
            ''',
            """
            `inspect.signature` gives the parameter list exactly as IPython
            shows it on the first line of `growth?`, and `__doc__` is the
            docstring.
            """, level=1),
        out("o2", "What does this print?",
            """
            def tariff(price, rate):
                # applies an import tariff
                return price * (1 + rate)

            print(tariff.__doc__)
            """,
            """
            A comment is not a docstring. With no string literal as the first
            statement, `__doc__` is `None`, so `tariff?` would show no
            documentation at all.
            """),
        out("o3", "What does this print?",
            """
            import inspect

            def spread(bid, ask):
                return ask - bid

            print(inspect.getsource(spread).splitlines()[1].strip())
            """,
            """
            `inspect.getsource` returns the function's source text, which is
            what `spread??` displays. Line 1 of that text (counting from 0) is
            the body.
            """, level=2),
        task("c1",
             """
             Write a function `real_wage(nominal, cpi, base=100)` that returns
             `nominal * base / cpi`, and give it a docstring that mentions
             "CPI", so that `real_wage?` in IPython would explain it.
             """,
             '''
             def real_wage(nominal, cpi, base=100):
                 """Convert a nominal wage to base-period terms using the CPI."""
                 return nominal * base / cpi
             ''',
             [
                 check("signature", """
                     import inspect
                     check(callable(get("real_wage")), "`real_wage` should be a function.")
                     check(str(inspect.signature(real_wage)) == "(nominal, cpi, base=100)",
                           "The parameters should be (nominal, cpi, base=100), in that order.")
                 """),
                 check("docstring", """
                     check(real_wage.__doc__ is not None and "CPI" in real_wage.__doc__,
                           "Add a docstring (a string literal as the first line of the body) that mentions CPI.")
                 """),
                 check("value", """
                     check(close(real_wage(25.0, 125.0), 20.0) and close(real_wage(30, 150, base=50), 10.0),
                           "real_wage should return nominal * base / cpi.")
                 """),
             ],
             """
             The docstring is the first statement of the body. `inspect` and
             IPython's `?` both read it from `__doc__`.
             """,
             starter="""
             def real_wage(nominal, cpi, base=100):
                 ...
             """, level=1),
        task("c2",
             """
             A function `deflate` is already defined (you cannot see its code).
             Using `inspect` and `__doc__`, set `params` to a list of its
             parameter names, in order, and `summary` to the first line of its
             docstring.
             """,
             """
             import inspect
             params = list(inspect.signature(deflate).parameters)
             summary = deflate.__doc__.splitlines()[0]
             """,
             [
                 check("params", """
                     check(list(get("params")) == ["series", "index", "base_year"],
                           "`params` should list the parameter names in order. Look at inspect.signature(...).parameters.")
                 """),
                 check("summary", """
                     check(get("summary") == "Divide a nominal series by a price index.",
                           "`summary` should be only the first line of the docstring.")
                 """),
             ],
             """
             `inspect.signature(f).parameters` is an ordered mapping keyed by
             parameter name, so `list(...)` gives the names. The docstring is a
             plain string; split it into lines and keep the first.
             """,
             starter="""
             import inspect
             params = ...
             summary = ...
             """,
             setup='''
             def deflate(series, index, base_year=2015):
                 """Divide a nominal series by a price index.

                 The index is rescaled so that base_year equals 100.
                 """
                 return [s / i * 100 for s, i in zip(series, index)]
             ''', level=2),
    ]),

concept(
    "py.ipy.completion", "Tab completion and wildcard search", ["py.ipy.help"],
    "1.01", "Help and Documentation in IPython", "01.01-help-and-documentation",
    """
    Typing `prices.` and pressing **Tab** lists the attributes and methods of
    `prices`. Type a few letters first (`prices.so` then Tab) and IPython
    narrows the list or finishes the name for you. The same works for module
    names after `import` and for file paths inside strings. Names that start
    with an underscore stay hidden until you type the underscore yourself.

    When you know part of a name but not where it sits, use a `*` wildcard
    with `?`:

    ```python
    In [1]: str.*strip*?     # lstrip, rstrip, strip
    In [2]: *Error?          # every name in scope ending in Error
    ```

    In plain Python, `dir(obj)` returns the names Tab draws on, and the
    standard `fnmatch` module matches the same shell-style patterns:
    `fnmatch.filter(dir(str), "*strip*")`.
    """,
    [
        mc("m1", "You have a list `rates` and type `rates.` then press Tab in IPython. What appears?",
           "The list's public methods, such as `append`, `count` and `sort`",
           ["Every name defined in the session",
            "The values stored in `rates`",
            "Only the dunder methods such as `__len__`"],
           """
           Tab after a dot completes attribute names of that object. Names
           beginning with `_` are left out until you type an underscore.
           """, level=1),
        mc("m2", "Which IPython command lists every `str` method whose name contains `find`?",
           "`str.*find*?`", ["`str.find??`", "`find*(str)?`", "`dir(str, 'find')`"],
           """
           `*` matches any run of characters, including none, so `*find*`
           catches both `find` and `rfind`. The trailing `?` asks IPython to
           list the matches.
           """),
        mc("m3", "In the pattern `*Warning`, what does `*` match?",
           "Any sequence of characters, including an empty one",
           ["Exactly one character", "Only letters, not digits", "Only names defined by the user"],
           """
           These are shell-style wildcards: `*` is any run of characters and
           `?` inside a pattern (not at the end) is exactly one character.
           """, level=1),
        out("o1", "What does this print?",
            """
            import fnmatch
            print(fnmatch.filter(dir(str), "*find*"))
            """,
            """
            `dir(str)` is alphabetical, and two methods contain `find`. This is
            the list `str.*find*?` shows in IPython.
            """, level=1),
        out("o2", "What does this print?",
            """
            public = [n for n in dir(list) if not n.startswith("_")]
            print(public)
            """,
            """
            These are the names IPython offers after `some_list.` and Tab. The
            many underscore names in `dir(list)` are hidden by default.
            """, level=2),
        task("c1",
             """
             Using `dir` and `fnmatch`, set `strip_methods` to the `str` method
             names that end in `strip`, and `dict_public` to the names in
             `dir(dict)` that do not start with an underscore.
             """,
             """
             import fnmatch
             strip_methods = fnmatch.filter(dir(str), "*strip")
             dict_public = [n for n in dir(dict) if not n.startswith("_")]
             """,
             [
                 check("strip", """
                     check(sorted(get("strip_methods")) == ["lstrip", "rstrip", "strip"],
                           "`strip_methods` should be the three str methods ending in 'strip'.")
                 """),
                 check("public", """
                     ref = [n for n in dir(dict) if not n.startswith("_")]
                     check(sorted(get("dict_public")) == sorted(ref),
                           "`dict_public` should hold every dict attribute name that does not begin with an underscore.")
                 """),
             ],
             """
             The pattern `*strip` anchors at the end of the name. Filtering out
             leading underscores mimics what Tab shows after `d.`.
             """,
             starter="""
             import fnmatch
             strip_methods = ...
             dict_public = ...
             """, level=1),
    ]),

# ---------------------------------------------------------------- 1.02
concept(
    "py.ipy.shortcuts", "Keyboard shortcuts and history search", ["py.ipy.help"],
    "1.02", "Keyboard Shortcuts in the IPython Shell", "01.02-shell-keyboard-shortcuts",
    """
    The IPython terminal borrows Emacs-style key bindings, so you can edit a
    long line without reaching for the mouse. A few earn their keep fast:

    | keys | effect |
    |---|---|
    | Ctrl-a / Ctrl-e | jump to start / end of the line |
    | Ctrl-k | cut from the cursor to the end of the line |
    | Up arrow | previous command (repeat to go further back) |
    | Ctrl-r | search backwards through past commands |
    | Ctrl-l | clear the screen |
    | Ctrl-c | interrupt the running command |

    Ctrl-r is the most useful: type a fragment such as `read_csv` and IPython
    shows the most recent command containing it. Press Ctrl-r again for older
    matches. In plain Python terms it is a backwards scan of a list of strings:

    ```python
    match = next(c for c in reversed(history) if "read_csv" in c)
    ```
    """,
    [
        mc("m1", "A regression has been running for ten minutes and you want to stop it without closing IPython. Which keys?",
           "Ctrl-c", ["Ctrl-d", "Ctrl-l", "Ctrl-k"],
           """
           Ctrl-c sends an interrupt, which raises `KeyboardInterrupt` in the
           running code and returns you to the prompt. Ctrl-d on an empty line
           asks to exit IPython altogether.
           """, level=1),
        mc("m2", "Your cursor is in the middle of a long line and you want to delete everything after it. Which keys?",
           "Ctrl-k", ["Ctrl-a", "Ctrl-e", "Ctrl-r"],
           """
           Ctrl-k cuts from the cursor to the end of the line. Ctrl-a and
           Ctrl-e only move the cursor; Ctrl-r searches history.
           """, level=1),
        mc("m3", "You typed a long `pd.read_csv(...)` call about forty commands ago. What is the quickest way to get it back?",
           "Press Ctrl-r and type `read_csv`",
           ["Press the Up arrow forty times",
            "Press Ctrl-a, then type `read_csv`",
            "Press Ctrl-l and scroll up"],
           """
           Ctrl-r searches backwards for the most recent command containing
           what you type. The Up arrow works too, but one step at a time.
           """),
        mc("m4", "In the IPython terminal, what does typing `df.gr` and then pressing the Up arrow do?",
           "Steps back to the previous command that starts with `df.gr`",
           ["Moves the cursor to the line above",
            "Deletes the characters `df.gr`",
            "Completes `df.gr` to `df.groupby`"],
           """
           With text already on the line, Up only recalls history entries
           that begin with that text. Completing names is Tab's job.
           """, level=2),
        out("o1",
            """
            This loop mimics Ctrl-r: it scans the history from the newest command
            backwards. What does it print?
            """,
            """
            history = [
                "import pandas as pd",
                "wages = pd.read_csv('wages.csv')",
                "wages.mean()",
                "wages.median()",
                "wages['hourly'].mean()",
            ]
            query = "mean"
            print(next(cmd for cmd in reversed(history) if query in cmd))
            """,
            """
            `reversed` starts from the last entry, and the first one that
            contains `mean` wins, just as Ctrl-r shows the most recent match
            first.
            """, level=1),
    ]),

# ---------------------------------------------------------------- 1.03
concept(
    "py.ipy.magics", "Magic commands: %run, %timeit, %lsmagic", ["py.ipy.help"],
    "1.03", "IPython Magic Commands", "01.03-magic-commands",
    """
    **Magic commands** are IPython's own additions to Python syntax. A *line
    magic* starts with one `%` and applies to the rest of that line; a *cell
    magic* starts with `%%` and applies to the whole cell below it.

    ```python
    In [1]: %run simulate.py          # run a script, keep its variables
    In [2]: %timeit sorted(prices)    # time one line
    In [3]: %lsmagic                  # list every available magic
    In [4]: %timeit?                  # magics have docstrings too
    ```

    `%run` executes a file as if it were the main program, then leaves its
    top-level names (functions, data, results) in your session so you can
    keep exploring them. The standard library has the same idea in
    `runpy.run_path`, which runs a file and returns its namespace as a
    dictionary:

    ```python
    import runpy
    ns = runpy.run_path("simulate.py", run_name="__main__")
    ns["results"]
    ```
    """,
    [
        mc("m1", "What is the difference between `%timeit` and `%%timeit`?",
           "`%timeit` times one line; `%%timeit` times the whole cell it starts",
           ["`%%timeit` runs twice as many loops",
            "`%%timeit` works only in Jupyter, `%timeit` only in the terminal",
            "There is no difference; the extra `%` is optional"],
           """
           One `%` makes a line magic, two make a cell magic. The cell form
           is handy when the code to time needs several lines.
           """, level=1),
        mc("m2", "After `%run model.py` finishes, where are the variables that `model.py` defined at top level?",
           "In your interactive namespace, so you can use them at the next prompt",
           ["Nowhere; they disappear when the script ends",
            "Only in a dictionary called `Out`",
            "In a new module object you must import first"],
           """
           `%run` executes the file and then copies its globals into your
           session, which is what makes it handy for inspecting a script's
           results.
           """),
        mc("m3", "Which command lists all the magic commands available in your IPython session?",
           "`%lsmagic`", ["`%magic?`", "`dir(%)`", "`help(magic)`"],
           """
           `%lsmagic` prints every line and cell magic. `%magic` prints a long
           general description of the magic system.
           """, level=1),
        mc("m4", "When IPython runs a script with `%run report.py`, what is `__name__` inside the script?",
           "`'__main__'`, so an `if __name__ == '__main__':` block runs",
           ["`'report'`", "`'__ipython__'`", "`None`"],
           """
           By default `%run` executes the file as the main program, exactly as
           `python report.py` would.
           """, level=2),
        out("o1", "What does this print?",
            """
            import runpy

            with open("demo.py", "w") as f:
                f.write("rate = 0.05\\n")
                f.write("years = 3\\n")
                f.write("growth = round((1 + rate) ** years, 4)\\n")

            ns = runpy.run_path("demo.py")
            print(ns["growth"], "years" in ns)
            """,
            """
            `run_path` executes the file and returns its top-level names, the
            same names `%run demo.py` would leave in your IPython session.
            1.05 cubed is 1.157625, rounded to four places.
            """, level=2),
        task("c1",
             """
             A script `elasticity.py` is already in the working folder. Run it
             with `runpy.run_path` as the main program (`run_name="__main__"`),
             keep the returned namespace in `ns`, and set `elasticity` to the
             value the script computed under that same name.
             """,
             """
             import runpy
             ns = runpy.run_path("elasticity.py", run_name="__main__")
             elasticity = ns["elasticity"]
             """,
             [
                 check("ns", """
                     check(isinstance(get("ns"), dict) and "elasticity" in ns,
                           "`ns` should be the dictionary returned by runpy.run_path.")
                 """),
                 check("value", """
                     check(close(get("elasticity"), -0.5),
                           "`elasticity` should be the value the script computed.")
                 """),
                 check("main", """
                     check(ns.get("ran_as_main") is True,
                           "Run the script as the main program by passing run_name='__main__'.")
                 """),
             ],
             """
             `runpy.run_path` is the plain-Python version of `%run`: it
             executes the file and hands back its globals. With
             `run_name="__main__"` the script's main block runs too.
             """,
             starter="""
             import runpy
             ns = ...
             elasticity = ...
             """,
             setup='''
             with open("elasticity.py", "w") as f:
                 f.write(
                     "q0, q1 = 100.0, 95.0\\n"
                     "p0, p1 = 10.0, 11.0\\n"
                     "elasticity = ((q1 - q0) / q0) / ((p1 - p0) / p0)\\n"
                     "ran_as_main = False\\n"
                     "if __name__ == '__main__':\\n"
                     "    ran_as_main = True\\n"
                 )
             ''', level=2),
    ]),

# ---------------------------------------------------------------- 1.04
concept(
    "py.ipy.history", "Input and output history: In, Out and _", ["py.ipy.magics"],
    "1.04", "Input and Output History", "01.04-input-output-history",
    """
    IPython numbers every prompt and records two objects as you work:

    - `In` is a list of the code you typed, so `In[3]` is the third input.
    - `Out` is a dict from prompt number to the value that prompt displayed.

    Shortcuts save typing: `_` is the last output, `__` the one before, `___`
    the one before that, and `_7` is `Out[7]`. A cell whose value is `None`,
    such as a `print` call, puts nothing in `Out`. Ending a line with `;`
    hides the value and keeps it out of `Out` too. `%history -n 1-5` lists
    inputs 1 to 5 with their numbers.

    ```python
    In [1]: 120 * 1.08
    Out[1]: 129.6
    In [2]: _ / 12        # uses Out[1]
    Out[2]: 10.8
    ```

    A script has none of this: an expression on its own line is computed and
    thrown away. The display step is `sys.displayhook`, which the interactive
    shells call with each value.
    """,
    [
        mc("m1", "Three cells in a row displayed `5`, `12` and `40`. What is `__` (two underscores) now?",
           "`12`", ["`5`", "`40`", "`52`"],
           """
           `_` is the latest output (40), `__` the one before it (12), and
           `___` the one before that (5).
           """, level=1),
        mc("m2", "Prompt 4 was `print(total)`. What is `Out[4]`?",
           "There is no `Out[4]`: `print` returns `None`, so nothing was stored",
           ["The printed text, as a string", "`None`", "The value of `total`"],
           """
           `print` writes text but its value is `None`, and IPython stores
           only values it displays. Looking up `Out[4]` raises `KeyError`.
           """),
        mc("m3", "Why end a line such as `fig = make_plot(data);` with a semicolon in IPython?",
           "To stop the value being displayed and stored in `Out`",
           ["Python requires it at the end of an assignment",
            "To run the line in the background",
            "To make the line a cell magic"],
           """
           The trailing semicolon suppresses the display. The code still runs
           and any assignment still happens.
           """, level=1),
        mc("m4", "What is `_3` in IPython?",
           "The output of prompt 3, the same as `Out[3]`",
           ["The input of prompt 3, the same as `In[3]`",
            "The third most recent output",
            "A syntax error"],
           """
           `_N` is shorthand for `Out[N]`. Inputs are reached through `In[N]`
           (or `_iN`).
           """, level=2),
        out("o1", "This runs as a script, not in IPython. What does it print?",
            """
            price = 80
            price * 1.25
            print("done")
            """,
            """
            In a script, a bare expression is computed and discarded; only an
            interactive shell shows its value. The only output is from `print`.
            """, level=1),
        out("o2", "What does this print?",
            """
            import sys, builtins
            sys.displayhook(6 * 7)
            sys.displayhook(None)
            print(builtins._ + 1)
            """,
            """
            `sys.displayhook` is what interactive Python calls with each
            expression's value. It prints the repr and saves the value as `_`,
            but it skips `None` entirely, which is why `print` calls never
            become `_`.
            """, level=3),
        task("c1",
             """
             Build a tiny version of IPython's history. Write
             `run_cells(cells)` that takes a list of expression strings and
             returns `(In, Out)`: `In` starts as `[""]` and gets every cell
             appended; `Out` is a dict from cell number (starting at 1) to the
             cell's value, evaluated with `eval`. Leave a cell out of `Out` if
             its value is `None` or if its text ends with `;` (strip the `;`
             before evaluating).
             """,
             """
             def run_cells(cells):
                 In, Out = [""], {}
                 for n, src in enumerate(cells, start=1):
                     In.append(src)
                     text = src.rstrip()
                     quiet = text.endswith(";")
                     value = eval(text.rstrip(";"))
                     if value is not None and not quiet:
                         Out[n] = value
                 return In, Out
             """,
             [
                 check("in", """
                     In_, Out_ = run_cells(["2 + 3", "None", "7 * 6;", "len('abc')"])
                     check(In_ == ["", "2 + 3", "None", "7 * 6;", "len('abc')"],
                           "`In` should start with an empty string and hold every cell's text.")
                 """),
                 check("out", """
                     In_, Out_ = run_cells(["2 + 3", "None", "7 * 6;", "len('abc')"])
                     check(Out_ == {1: 5, 4: 3},
                           "`Out` should map cell numbers to values, skipping None and cells ending in ';'.")
                 """),
             ],
             """
             Numbering from 1 matches IPython's prompts, and `In[0]` is an
             empty placeholder for the same reason. The semicolon and `None`
             rules are the two ways a value stays out of `Out`.
             """,
             starter="""
             def run_cells(cells):
                 In, Out = [""], {}
                 ...
                 return In, Out
             """, level=3),
    ]),

# ---------------------------------------------------------------- 1.05
concept(
    "py.ipy.shell", "Shell commands with ! and %cd", ["py.ipy.magics"],
    "1.05", "IPython and Shell Commands", "01.05-ipython-and-shell-commands",
    """
    Any line that starts with `!` goes to the operating system's shell
    instead of Python:

    ```python
    In [1]: !ls data
    In [2]: files = !ls data          # capture the output as a list of lines
    In [3]: year = 2023
    In [4]: !echo {year}              # {name} inserts a Python value
    ```

    The captured result is a list-like object with one string per line of
    output, so ordinary Python can filter it.

    Each `!` command runs in a fresh subshell, so `!cd reports` changes
    directory only for that throwaway shell. To move IPython itself, use the
    `%cd` magic. Common ones such as `cd`, `pwd` and `ls` also work without
    the `%` (this is called *automagic*). In plain Python the same jobs belong
    to `os.getcwd`, `os.chdir`, `os.listdir` and `os.makedirs`.
    """,
    [
        mc("m1", "You run `!cd output` and then `!pwd`. Why is the directory unchanged?",
           "Each `!` command runs in its own subshell, which exits when the command ends",
           ["`!cd` needs an absolute path",
            "`pwd` reports the directory IPython was installed in",
            "Shell commands are queued and run only when IPython exits"],
           """
           The `cd` happened inside a short-lived child shell. Use `%cd output`
           (or plain `cd output`, through automagic) to change IPython's own
           working directory.
           """, level=2),
        mc("m2", "After `lines = !wc -l survey.csv`, what kind of object is `lines`?",
           "A list-like object holding the command's output, one string per line",
           ["The command's exit code as an int",
            "A single string with the whole output",
            "`None`, because shell output is only printed"],
           """
           Assigning a `!` command captures its standard output as a list of
           lines (IPython's `SList`), which you can index, loop over and filter.
           """, level=1),
        mc("m3", "With `region = 'north'` defined, which line passes the value to a shell command?",
           "`!echo {region}`", ["`!echo region`", "`!echo %region`", "`%echo region`"],
           """
           Braces around a Python name expand it before the line goes to the
           shell. Without them the shell sees the literal word.
           """),
        out("o1", "What does this print?",
            """
            import os
            start = os.getcwd()
            os.makedirs("reports/2024", exist_ok=True)
            os.chdir("reports/2024")
            print(os.path.relpath(os.getcwd(), start))
            """,
            """
            `os.chdir` changes the current process's directory and it stays
            changed, which is what `%cd` does for IPython. `relpath` shows the
            new location relative to where we started.
            """, level=1),
        out("o2", "What does this print?",
            """
            listing = "wages.csv\\nprices.csv\\nnotes.txt\\n"   # what `!ls` might return
            lines = listing.splitlines()
            print(len(lines), [n for n in lines if n.endswith(".csv")])
            """,
            """
            Captured shell output is split into one string per line, so
            filtering it is ordinary list work.
            """, level=1),
        task("c1",
             """
             A folder `data` in the working directory holds several files. Set
             `csvs` to the sorted list of the file names in `data` that end in
             `.csv`, using the `os` module (the plain-Python version of
             `files = !ls data` followed by a filter).
             """,
             """
             import os
             csvs = sorted(n for n in os.listdir("data") if n.endswith(".csv"))
             """,
             [
                 check("csvs", """
                     check(list(get("csvs")) == ["prices.csv", "rents.csv", "wages.csv"],
                           "`csvs` should be the sorted names in data/ that end with .csv.")
                 """),
             ],
             """
             `os.listdir` returns the names in a folder in arbitrary order, so
             sort after filtering.
             """,
             starter="""
             import os
             csvs = ...
             """,
             setup="""
             import os
             os.makedirs("data", exist_ok=True)
             for name in ["wages.csv", "notes.txt", "prices.csv", "rents.csv", "readme.md"]:
                 with open(os.path.join("data", name), "w") as f:
                     f.write("x\\n")
             """, level=1),
        task("c2",
             """
             Make a folder `results/q1` (creating `results` too), change into it
             with `os.chdir`, and set `here` to the name of the folder you are
             now in (just the last part of the path).
             """,
             """
             import os
             os.makedirs("results/q1", exist_ok=True)
             os.chdir("results/q1")
             here = os.path.basename(os.getcwd())
             """,
             [
                 check("cwd", """
                     import os
                     check(os.getcwd().endswith(os.path.join("results", "q1")),
                           "The working directory should now be results/q1. Did you call os.chdir?")
                 """),
                 check("here", """
                     check(get("here") == "q1", "`here` should be the last part of the current path.")
                 """),
             ],
             """
             `os.makedirs` creates intermediate folders. After `os.chdir`, every
             later relative path starts from the new folder, as after `%cd`.
             """,
             starter="""
             import os
             here = ...
             """, level=2),
    ]),

# ---------------------------------------------------------------- 1.06
concept(
    "py.ipy.errors", "Tracebacks, %xmode and %debug", ["py.ipy.magics"],
    "1.06", "Errors and Debugging", "01.06-errors-and-debugging",
    """
    When code fails, Python shows a **traceback**: the chain of calls that led
    to the error, outermost first, ending with the exception type and message.
    IPython lets you choose how much detail you see with `%xmode`: `Minimal`
    (just the error line), `Plain` (classic Python), `Context` (the default,
    with surrounding lines of code) and `Verbose` (also the values of
    variables).

    ```python
    In [5]: %xmode Verbose
    In [6]: %debug        # open the debugger at the last exception
    ipdb> up              # move to the calling frame
    ipdb> p wage          # print a variable there
    ipdb> q               # quit
    ```

    `%pdb on` makes the debugger open by itself after every exception. The
    same information is available to programs: an exception object carries
    its traceback in `e.__traceback__`, and the `traceback` module turns it
    into frames and text.
    """,
    [
        mc("m1", "Which `%xmode` setting also shows the values of local variables in each frame?",
           "`Verbose`", ["`Context`", "`Plain`", "`Minimal`"],
           """
           `Verbose` adds variable values, which helps when the error depends
           on data. `Context` (the default) shows code around each line;
           `Minimal` shows only the exception.
           """, level=1),
        mc("m2", "A function call just raised an exception. What does typing `%debug` do?",
           "Opens the debugger in the frame where the exception happened, so you can inspect its variables",
           ["Reruns the last cell line by line",
            "Turns on verbose tracebacks for future errors",
            "Deletes the variables involved in the error"],
           """
           `%debug` is post-mortem debugging: you land in the failing frame
           with all its local state intact.
           """),
        mc("m3", "Inside the debugger, the error is in a helper but the bad argument came from its caller. Which command moves to the caller's frame?",
           "`up` (or `u`)", ["`next` (or `n`)", "`continue` (or `c`)", "`step` (or `s`)"],
           """
           `up` and `down` move through the call stack without running any
           code. `next` and `step` execute lines, and `continue` resumes.
           """, level=2),
        mc("m4", "What does `%pdb on` do?",
           "Starts the debugger automatically whenever an exception is raised",
           ["Prevents exceptions from being raised",
            "Prints a traceback for every successful cell",
            "Installs the pdb module"],
           """
           With automatic debugging on, you skip typing `%debug` after each
           failure; `%pdb off` turns it back off.
           """, level=1),
        out("o1", "What does this print?",
            """
            import traceback

            def share(part, total):
                return part / total

            try:
                share(3, 0)
            except ZeroDivisionError as e:
                print(type(e).__name__)
                print(traceback.format_exception_only(e)[-1].strip())
            """,
            """
            `format_exception_only` gives the final line of a traceback, which
            is all `%xmode Minimal` shows.
            """, level=1),
        out("o2", "What does this print?",
            """
            import traceback

            def unit_cost(total, units):
                return total / units

            def report(rows):
                out = []
                for t, u in rows:
                    out.append(unit_cost(t, u))
                return out

            try:
                report([(100, 4), (50, 0)])
            except ZeroDivisionError as e:
                frames = traceback.extract_tb(e.__traceback__)
                print([f.name for f in frames])
            """,
            """
            The traceback lists frames from the outermost call inward: the
            module itself, then `report`, then `unit_cost`, where the division
            failed. `up` in the debugger walks this list backwards.
            """, level=3),
        task("c1",
             """
             Write `parse_wages(rows)` that converts each string in `rows` to a
             float. Strings that cannot be converted raise `ValueError`; catch
             that, skip the entry, and record its position. Return a tuple
             `(values, bad)` where `bad` is the list of failing indexes.
             """,
             """
             def parse_wages(rows):
                 values, bad = [], []
                 for i, r in enumerate(rows):
                     try:
                         values.append(float(r))
                     except ValueError:
                         bad.append(i)
                 return values, bad
             """,
             [
                 check("good", """
                     vals, bad = parse_wages(["15.5", "n/a", "22", "", "18.25"])
                     check(close(vals, [15.5, 22.0, 18.25]), "`values` should hold only the strings that converted.")
                 """),
                 check("bad", """
                     vals, bad = parse_wages(["15.5", "n/a", "22", "", "18.25"])
                     check(list(bad) == [1, 3], "`bad` should list the positions (from 0) of the strings that failed.")
                 """),
             ],
             """
             Wrap just the conversion in `try`, and catch only `ValueError` so
             that a real bug of another kind still surfaces with its traceback.
             """,
             starter="""
             def parse_wages(rows):
                 values, bad = [], []
                 ...
                 return values, bad
             """, level=2),
        task("c2",
             """
             Calling `run_model()` (already defined) raises an exception deep
             inside some helper. Call it inside `try/except Exception as e`,
             then set `err_type` to the exception's class name and `failed_in`
             to the name of the function where it was raised (the last frame
             of its traceback).
             """,
             """
             import traceback
             try:
                 run_model()
             except Exception as e:
                 err_type = type(e).__name__
                 failed_in = traceback.extract_tb(e.__traceback__)[-1].name
             """,
             [
                 check("type", """
                     check(get("err_type") == "KeyError", "`err_type` should be the exception's class name, as a string.")
                 """),
                 check("frame", """
                     check(get("failed_in") == "lookup_rate",
                           "`failed_in` should be the function name in the last frame of traceback.extract_tb(e.__traceback__).")
                 """),
             ],
             """
             The innermost frame is where the exception was raised; that is
             where `%debug` would drop you.
             """,
             starter="""
             import traceback
             err_type = ...
             failed_in = ...
             """,
             setup="""
             RATES = {"north": 0.03, "south": 0.025}

             def lookup_rate(region):
                 return RATES[region]

             def project(region, base):
                 return base * (1 + lookup_rate(region))

             def run_model():
                 out = []
                 for r in ["north", "west"]:
                     out.append(project(r, 100))
                 return out
             """, level=3),
    ]),

# ---------------------------------------------------------------- 1.07
concept(
    "py.ipy.timing", "Timing code with %timeit and %time", ["py.ipy.magics"],
    "1.07", "Profiling and Timing Code", "01.07-timing-and-profiling",
    """
    `%timeit stmt` runs a statement many times, picks the loop count for you,
    and reports a mean and standard deviation per run. That is the right tool
    for fast code, where one run is too short to measure. `%time stmt` runs
    it once and reports wall and CPU time, which suits slow code or code that
    should not be repeated. `%%timeit` times a whole cell.

    Both are wrappers around tools you can call from plain Python:

    ```python
    import timeit, time
    timeit.timeit("sum(range(1000))", number=500)   # total seconds for 500 runs
    timeit.repeat("sum(range(1000))", number=500, repeat=5)  # 5 such totals

    t0 = time.perf_counter()
    result = sum(range(10**6))
    elapsed = time.perf_counter() - t0
    ```

    Compare timings only on the same machine, and look for large gaps (ten
    times, a hundred times), not small ones that noise can explain.
    """,
    [
        mc("m1", "You want to know how long one call to a function that takes 20 seconds needs. Which is the sensible choice?",
           "`%time`, which runs it once",
           ["`%timeit`, which will repeat it many times",
            "`%lsmagic`", "`%run`"],
           """
           `%timeit` repeats the statement to average out noise, which costs
           minutes for a 20-second call. A single `%time` run is plenty when
           the code is that slow.
           """, level=1),
        mc("m2", "What does `timeit.timeit(stmt, number=1000)` return?",
           "The total time in seconds, as a float, for 1000 executions",
           ["The average time per execution in microseconds",
            "A list of 1000 timings",
            "The value of the last execution of `stmt`"],
           """
           Divide by `number` for the time per run. `timeit.repeat` returns a
           list of such totals, one per repeat.
           """),
        mc("m3", "Why does `%timeit` report the spread of several runs rather than a single number?",
           "Run times vary with whatever else the machine is doing, so one measurement can mislead",
           ["Python code runs at a different speed every second by design",
            "The first run is always skipped",
            "To estimate how much memory the statement uses"],
           """
           Other processes, caches and the CPU's clock speed all add noise.
           Several runs show how stable the estimate is.
           """, level=1),
        out("o1", "What does this print?",
            """
            import timeit
            times = timeit.repeat("sorted(range(200))", number=50, repeat=4)
            print(len(times), type(times[0]).__name__, all(t > 0 for t in times))
            """,
            """
            `repeat` gives one total per repeat, each a float number of
            seconds. The values themselves change from run to run, which is why
            the question asks only about their shape.
            """, level=1),
        task("c1",
             """
             `data_list` (a list) and `data_set` (a set) both hold the integers
             0 to 199,999. Use `timeit.timeit` with `number=200` and
             `globals=globals()` to time the statement `199_999 in data_list`,
             storing the total in `t_list`, and the same test on `data_set` in
             `t_set`.
             """,
             """
             import timeit
             t_list = timeit.timeit("199_999 in data_list", number=200, globals=globals())
             t_set = timeit.timeit("199_999 in data_set", number=200, globals=globals())
             """,
             [
                 check("floats", """
                     check(isinstance(get("t_list"), float) and isinstance(get("t_set"), float),
                           "`t_list` and `t_set` should be the floats that timeit.timeit returns.")
                     check(t_list > 0 and t_set > 0, "Both timings should be positive.")
                 """),
                 check("order", """
                     check(t_list > 20 * t_set,
                           "Searching the list should be far slower than the set. Did you time `199_999 in data_list` for t_list?")
                 """),
                 check("timeit", 'check(uses("timeit"), "Use timeit.timeit for the measurements.")'),
             ],
             """
             A list membership test scans every element; a set looks the value
             up by hash. The gap is in the hundreds or thousands, so it shows
             up clearly however noisy the machine is.
             """,
             starter="""
             import timeit
             t_list = ...
             t_set = ...
             """,
             setup="""
             data_list = list(range(200_000))
             data_set = set(data_list)
             """, level=2),
        task("c2",
             """
             Write `elapsed(func)` that calls `func()` once and returns how many
             seconds it took, measured with `time.perf_counter` (a tiny `%time`).
             """,
             """
             import time

             def elapsed(func):
                 t0 = time.perf_counter()
                 func()
                 return time.perf_counter() - t0
             """,
             [
                 check("float", """
                     t = elapsed(lambda: None)
                     check(isinstance(t, float) and t >= 0, "elapsed should return a non-negative float number of seconds.")
                 """),
                 check("calls", """
                     calls = []
                     elapsed(lambda: calls.append(1))
                     check(calls == [1], "elapsed should call func exactly once.")
                 """),
                 check("order", """
                     slow = elapsed(lambda: sum(range(2_000_000)))
                     fast = elapsed(lambda: None)
                     check(slow > 20 * fast and slow > 0, "A slow function should take much longer than an empty one.")
                 """),
             ],
             """
             Read `perf_counter` before and after the call and subtract. It is
             a high-resolution clock meant for exactly this; its absolute value
             means nothing on its own.
             """,
             starter="""
             import time

             def elapsed(func):
                 ...
             """, level=1),
    ]),

concept(
    "py.ipy.profiling", "Profiling with %prun, %lprun and %memit", ["py.ipy.timing", "py.ipy.errors"],
    "1.07", "Profiling and Timing Code", "01.07-timing-and-profiling",
    """
    Timing says *how long*; profiling says *where*. `%prun stmt` runs the
    statement under Python's built-in `cProfile` and prints, for every
    function called, how many times it ran and how long it took, sorted by
    the time spent inside each function itself.

    Two popular extensions go further. After `%load_ext line_profiler`,
    `%lprun -f func stmt` shows the time spent on each *line* of `func`.
    After `%load_ext memory_profiler`, `%memit stmt` reports peak memory and
    `%mprun` gives line-by-line memory use.

    ```python
    import cProfile, pstats
    prof = cProfile.Profile()
    prof.enable()
    simulate_economy(1000)
    prof.disable()
    pstats.Stats(prof).sort_stats("tottime").print_stats(5)
    ```

    The standard library's `tracemalloc` covers the `%memit` idea:
    `tracemalloc.get_traced_memory()` returns the current and peak bytes.
    """,
    [
        mc("m1", "Which tool tells you which *function* in a long pipeline uses the most time?",
           "`%prun`", ["`%timeit`", "`%time`", "`%memit`"],
           """
           `%prun` breaks the run down by function. The timing magics give one
           number for the whole statement.
           """, level=1),
        mc("m2", "What must you do before `%lprun` works in a fresh IPython session?",
           "Install the line_profiler package and run `%load_ext line_profiler`",
           ["Nothing; it is built into IPython",
            "Run `%prun` first to collect data",
            "Restart IPython in debug mode"],
           """
           `%lprun` and `%memit` come from third-party extensions. Once
           installed, `%load_ext` registers their magics.
           """),
        mc("m3", "In `%prun` output, `tottime` is the time inside a function excluding calls to other functions. What does `cumtime` include?",
           "The function's own time plus the time of every function it called",
           ["Only the time of the first call",
            "The time since IPython started",
            "The time spent in the profiler itself"],
           """
           A thin wrapper that calls something slow has a small `tottime` but
           a large `cumtime`. Look at `tottime` to find where work really
           happens.
           """, level=2),
        mc("m4", "You run `%lprun -f clean_data pipeline(df)`. What do you get?",
           "Time spent on each line of `clean_data` while `pipeline(df)` runs",
           ["Time spent on each line of `pipeline`",
            "Memory used by each line of `clean_data`",
            "A count of how often `pipeline` is called"],
           """
           `-f` names the function to inspect line by line, and the rest of the
           line is the statement that exercises it.
           """, level=2),
        out("o1", "What does this print?",
            """
            import cProfile, pstats

            def tax(income):
                return 0.2 * income

            def payroll(n):
                return sum(tax(w) for w in range(n))

            prof = cProfile.Profile()
            prof.enable()
            payroll(25)
            prof.disable()

            stats = pstats.Stats(prof).stats
            for (filename, line, func), (cc, ncalls, tt, ct, callers) in stats.items():
                if func == "tax":
                    print(ncalls)
            """,
            """
            The profiler counts every call. `tax` runs once per worker, the
            same number `%prun payroll(25)` shows in its `ncalls` column.
            """, level=2),
        task("c1",
             """
             `simulate(n)` is already defined and calls a helper `draw_shock`.
             Profile one call of `simulate(300)` with `cProfile.Profile`,
             build `stats = pstats.Stats(prof)`, and set `shock_calls` to the
             number of times `draw_shock` was called, read from `stats.stats`.
             """,
             """
             import cProfile, pstats
             prof = cProfile.Profile()
             prof.enable()
             simulate(300)
             prof.disable()
             stats = pstats.Stats(prof)
             shock_calls = next(v[1] for k, v in stats.stats.items() if k[2] == "draw_shock")
             """,
             [
                 check("stats", """
                     import pstats
                     check(isinstance(get("stats"), pstats.Stats), "`stats` should be a pstats.Stats built from your profiler.")
                 """),
                 check("calls", """
                     check(get("shock_calls") == 900,
                           "`shock_calls` should be the call count recorded for draw_shock. Keys of stats.stats are (file, line, name) and each value starts with (primitive calls, total calls, ...).")
                 """),
             ],
             """
             Each key in `stats.stats` is `(filename, line, function name)`
             and each value starts with the primitive and total call counts.
             Reading the profile directly is what `%prun` formats into a table.
             """,
             starter="""
             import cProfile, pstats
             prof = ...
             stats = ...
             shock_calls = ...
             """,
             setup="""
             def draw_shock(t):
                 return ((t * 7919) % 101) / 100 - 0.5

             def simulate(n):
                 path = [0.0]
                 for t in range(n):
                     for k in range(3):
                         path.append(path[-1] * 0.9 + draw_shock(t + k))
                 return path
             """, level=3),
        task("c2",
             """
             Measure peak memory the way `%memit` would, using `tracemalloc`.
             Set `peak_small` to the peak traced bytes while building
             `[float(i) for i in range(1_000)]`, and `peak_big` to the peak for
             `range(200_000)`. Start tracing before each build and stop it
             after reading `tracemalloc.get_traced_memory()`.
             """,
             """
             import tracemalloc

             tracemalloc.start()
             small = [float(i) for i in range(1_000)]
             peak_small = tracemalloc.get_traced_memory()[1]
             tracemalloc.stop()
             del small

             tracemalloc.start()
             big = [float(i) for i in range(200_000)]
             peak_big = tracemalloc.get_traced_memory()[1]
             tracemalloc.stop()
             """,
             [
                 check("ints", """
                     check(isinstance(get("peak_small"), int) and isinstance(get("peak_big"), int),
                           "Both peaks should be the int byte counts from get_traced_memory().")
                     check(peak_small > 0, "`peak_small` should be positive: start tracing before building the list.")
                 """),
                 check("order", """
                     check(peak_big > 20 * peak_small,
                           "`peak_big` should be far larger than `peak_small`. Trace each build separately.")
                 """),
             ],
             """
             `get_traced_memory()` returns `(current, peak)` in bytes since
             tracing started. Restarting between the two builds keeps the small
             measurement from leaking into the big one.
             """,
             starter="""
             import tracemalloc
             peak_small = ...
             peak_big = ...
             """, level=2),
    ]),

    ],
}
