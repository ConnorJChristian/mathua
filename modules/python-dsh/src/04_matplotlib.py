# Chapter 4 of the Python Data Science Handbook: Visualization with Matplotlib.
#
# Every question and note here is original. Each concept links to the book
# section it follows; the book's own text is CC-BY-NC-ND and is never adapted.
# Build with: python3 scripts/python_bank.py build matplotlib
#
# Code tasks run on the Agg backend (no window). Hidden tests inspect the
# figures the learner's code leaves open, through `figures()`.

CHAPTER = {
    "chapter": "matplotlib",
    "title": "Visualization with Matplotlib",
    "domain": "python_matplotlib",
    "concepts": [

# ---------------------------------------------------------------- 4.00
concept(
    "py.mpl.interfaces", "Figures, axes and the two interfaces", ["py.np.create"],
    "4.00", "Introduction to Matplotlib", "04.00-introduction-to-matplotlib",
    """
    A Matplotlib picture has two layers. The **Figure** is the whole canvas;
    each **Axes** is one plotting panel inside it, with its own x and y axis.

    There are two ways to drive them. The *pyplot* style calls functions on
    `plt` and lets Matplotlib track a "current" figure and axes for you. The
    *object-oriented* style asks for the objects and calls their methods:

    ```python
    import matplotlib.pyplot as plt
    import numpy as np
    years = np.arange(2015, 2025)
    debt = np.array([62, 64, 63, 66, 70, 79, 77, 74, 75, 76])

    fig, ax = plt.subplots()      # one figure, one axes
    ax.plot(years, debt)
    fig.savefig("debt.png")       # the format comes from the extension
    ```

    In a script, `plt.show()` opens a window and blocks until it is closed;
    in a notebook the figure appears below the cell. The object style is
    clearer once a figure has more than one panel, because you always know
    which panel a call affects.
    """,
    [
        mc("m1", "In `fig, ax = plt.subplots()`, what is `ax`?",
           "One plotting panel (an Axes) inside the figure",
           ["The x axis only", "The whole canvas, including every panel", "A NumPy array of the plotted data"],
           """
           The Figure is the container; an Axes is a single panel with its own
           coordinate system, ticks and labels. With no arguments, `subplots`
           makes exactly one.
           """, level=1),
        mc("m2", "How does `fig.savefig(\"wage_growth.pdf\")` decide the file format?",
           "From the file extension, so this writes a PDF",
           ["It always writes PNG and names it .pdf", "From an rcParam that must be set first", "It asks which format to use"],
           """
           `savefig` infers the format from the extension (png, pdf, svg and
           others). You can also pass `format=` explicitly.
           """, level=1),
        mc("m3", "Which statement about the pyplot interface (`plt.plot`, `plt.title`, ...) is true?",
           "Each call acts on whichever figure and axes are current",
           ["It cannot draw more than one line per figure",
            "It is a separate library from the object-oriented interface",
            "It only works inside notebooks"],
           """
           pyplot keeps hidden state: the current figure and the current axes.
           That is convenient for quick plots but makes it less obvious which
           panel a call changes when there are several.
           """, level=2),
        out("o1", "What does this print?",
            """
            import matplotlib.pyplot as plt
            fig, ax = plt.subplots()
            ax.plot([1, 2, 3], [4, 1, 5])
            ax.plot([1, 2, 3], [2, 2, 2])
            print(len(fig.axes), len(ax.lines))
            """,
            """
            Both `plot` calls draw into the same axes, so there is one axes
            holding two lines.
            """, level=1),
        out("o2", "What does this print?",
            """
            import matplotlib.pyplot as plt
            plt.plot([0, 1], [0, 1])
            plt.figure()
            plt.plot([0, 1], [1, 0])
            plt.plot([0, 1], [0.5, 0.5])
            print(len(plt.get_fignums()), len(plt.gca().lines))
            """,
            """
            The first `plt.plot` creates figure 1. `plt.figure()` makes figure 2
            current, so the next two lines go there. `plt.gca()` is figure 2's
            axes.
            """, level=2),
        task("c1",
             """
             Using the object-oriented interface, make a figure with one axes
             and plot `rate` (y) against `month` (x). Both arrays are already
             defined. Keep the figure in `fig` and the axes in `ax`.
             """,
             """
             import matplotlib.pyplot as plt
             fig, ax = plt.subplots()
             ax.plot(month, rate)
             """,
             [
                 check("figure", """
                     figs = figures()
                     check(len(figs) == 1, "Make exactly one figure.")
                     check(len(figs[0].axes) == 1, "The figure should have one axes.")
                     check(get("ax") is figs[0].axes[0], "`ax` should be the axes inside your figure.")
                 """),
                 check("line", """
                     ax = figures()[0].axes[0]
                     check(len(ax.lines) == 1, "Draw exactly one line.")
                     x, y = ax.lines[0].get_data()
                     check(close(x, month) and close(y, rate), "Plot `rate` on the y axis against `month` on the x axis.")
                 """),
             ],
             """
             `plt.subplots()` returns the figure and its axes; `ax.plot(x, y)`
             puts x first.
             """,
             starter="""
             import matplotlib.pyplot as plt
             fig, ax = ...
             """,
             setup="""
             import numpy as np
             month = np.arange(1, 13)
             rate = np.array([3.9, 3.8, 3.8, 3.9, 4.0, 4.1, 4.3, 4.2, 4.1, 4.1, 4.2, 4.2])
             """, level=1),
    ]),

# ---------------------------------------------------------------- 4.01
concept(
    "py.mpl.line_styles", "Line plots: colors and styles", ["py.mpl.interfaces", "py.np.ufuncs"],
    "4.01", "Simple Line Plots", "04.01-simple-line-plots",
    """
    `ax.plot(x, y)` joins points with straight segments. Call it again and the
    next line gets the next color in the cycle (`C0`, `C1`, ...).

    Appearance is set with keywords:

    ```python
    t = np.linspace(0, 10, 50)
    ax.plot(t, np.exp(0.05 * t), color="C2", linestyle="--", linewidth=2)
    ax.plot(t, 1 + 0.05 * t, color="gray", linestyle=":")
    ax.plot(t, np.ones_like(t), "k-.")       # shorthand: black, dash-dot
    ```

    Colors can be names (`"gray"`), cycle codes (`"C1"`), hex strings
    (`"#1f77b4"`) or grayscale strings (`"0.6"`). Line styles are `"-"`,
    `"--"`, `":"` and `"-."`. The short format string packs a color letter,
    a marker and a line style into one argument, which is handy for quick
    looks but harder to read later.
    """,
    [
        mc("m1", "What does the format string `\"r--\"` in `ax.plot(x, y, \"r--\")` give?",
           "A red dashed line", ["A red dotted line", "Red circles with no line", "A thick red solid line"],
           "`r` is red and `--` is dashed. A dotted line is `:`.", level=1),
        mc("m2", "You call `ax.plot` three times with no color argument. What colors do the lines get?",
           "The first three colors of the property cycle, `C0`, `C1` and `C2`",
           ["All three are the same blue", "Random colors, different every run", "Black, then gray, then light gray"],
           """
           Each new line on an axes takes the next entry from the color cycle,
           so unstyled lines are distinguishable and reproducible.
           """, level=1),
        mc("m3", "The default line width is 1.5 points. Which call draws a line twice as thick?",
           "`ax.plot(x, y, linewidth=3)`", ["`ax.plot(x, y, linewidth=2)`", "`ax.plot(x, y, linestyle=2)`", "`ax.plot(x, y, size=3)`"],
           """
           The width is in points, so doubling 1.5 gives 3; `lw` is a short
           alias. `linestyle` controls dashes, not thickness.
           """, level=2),
        out("o1", "What does this print?",
            """
            import matplotlib.pyplot as plt
            fig, ax = plt.subplots()
            line, = ax.plot([1, 2, 3], [2, 4, 3], "g:")
            print(line.get_color(), line.get_linestyle())
            """,
            """
            The format string `g:` sets the color to `g` (green) and the line
            style to dotted, `:`.
            """, level=1),
        out("o2", "What does this print?",
            """
            import matplotlib.pyplot as plt
            fig, ax = plt.subplots()
            from matplotlib.colors import to_hex
            for k in range(4):
                ax.plot([0, 1], [k, k + 1])
            ax.plot([0, 1], [5, 5], color="C1")
            print(len({to_hex(ln.get_color()) for ln in ax.lines}))
            """,
            """
            The four unstyled lines take `C0` to `C3`, four different colors.
            The fifth line asks for `C1`, which repeats the second color, so
            there are four distinct colors in total.
            """, level=2),
        task("c1",
             """
             On one axes, plot `nominal` against `year` as a solid black line,
             and `real` against `year` as a dashed line with width 3. Both
             series and `year` are already defined.
             """,
             """
             import matplotlib.pyplot as plt
             fig, ax = plt.subplots()
             ax.plot(year, nominal, color="black")
             ax.plot(year, real, linestyle="--", linewidth=3)
             """,
             [
                 check("two lines", """
                     ax = figures()[0].axes[0]
                     check(len(ax.lines) == 2, "Draw two lines on the same axes.")
                 """),
                 check("nominal", """
                     from matplotlib.colors import to_rgb
                     ax = figures()[0].axes[0]
                     ln = [l for l in ax.lines if close(l.get_ydata(), nominal)]
                     check(ln, "One line should plot `nominal` against `year`.")
                     check(to_rgb(ln[0].get_color()) == (0, 0, 0), "The nominal line should be black.")
                     check(ln[0].get_linestyle() == "-", "The nominal line should be solid.")
                 """),
                 check("real", """
                     ax = figures()[0].axes[0]
                     ln = [l for l in ax.lines if close(l.get_ydata(), real)]
                     check(ln, "One line should plot `real` against `year`.")
                     check(ln[0].get_linestyle() == "--", "The real line should be dashed.")
                     check(ln[0].get_linewidth() == 3, "The real line should have width 3.")
                 """),
             ],
             """
             `color="black"` (or `"k"`) and the default style give a solid
             black line; `linestyle="--"` and `linewidth=3` style the second.
             """,
             starter="""
             import matplotlib.pyplot as plt
             fig, ax = plt.subplots()
             ax.plot(year, nominal)
             # add the real series
             """,
             setup="""
             import numpy as np
             year = np.arange(2018, 2025)
             nominal = np.array([100, 104, 107, 113, 121, 127, 131])
             real = np.array([100, 102, 104, 105, 104, 106, 108])
             """, level=2),
    ]),

concept(
    "py.mpl.limits_labels", "Axis limits, labels and titles", ["py.mpl.line_styles"],
    "4.01", "Simple Line Plots", "04.01-simple-line-plots",
    """
    Matplotlib picks limits that fit the data with a small margin. Override
    them with `set_xlim` and `set_ylim`; passing them in descending order
    flips the axis.

    ```python
    ax.plot(q, price)
    ax.set_xlim(0, 100)
    ax.set_ylim(0, 25)
    ax.set_xlabel("Quantity")
    ax.set_ylabel("Price")
    ax.set_title("Inverse demand")
    # or all at once:
    ax.set(xlim=(0, 100), ylim=(0, 25), xlabel="Quantity", title="Inverse demand")
    ```

    A common stumble when moving between interfaces: the pyplot functions
    `plt.xlabel`, `plt.xlim` and `plt.title` become methods with a `set_`
    prefix on an axes: `ax.set_xlabel`, `ax.set_xlim`, `ax.set_title`.
    `ax.axis("equal")` makes one unit the same length on both axes, which
    matters when the picture is geometric.
    """,
    [
        mc("m1", "You have `ax` from `plt.subplots()`. Which call sets the x-axis label?",
           "`ax.set_xlabel(\"Hours\")`", ["`ax.xlabel(\"Hours\")`", "`ax.label_x(\"Hours\")`", "`ax.xaxis = \"Hours\"`"],
           """
           Axes methods that change a property use the `set_` prefix. The
           unprefixed `xlabel` exists only as `plt.xlabel`.
           """, level=1),
        mc("m2", "What does `ax.set_ylim(10, 0)` do?",
           "Shows y from 0 to 10 with the axis flipped, so 10 is at the bottom",
           ["Raises an error because the lower limit is larger", "Sets limits 0 to 10 in the usual direction", "Hides the y axis"],
           "Limits are applied in the order given, so a larger first value inverts the axis.", level=2),
        mc("m3", "Which single call sets the title and both axis labels of `ax`?",
           "`ax.set(title=\"Wages\", xlabel=\"Year\", ylabel=\"Index\")`",
           ["`ax.labels(\"Wages\", \"Year\", \"Index\")`", "`plt.set(ax, \"Wages\", \"Year\", \"Index\")`", "`ax.title(\"Wages\", \"Year\", \"Index\")`"],
           "`ax.set(**kwargs)` calls the matching `set_*` method for every keyword.", level=2),
        out("o1", "What does this print?",
            """
            import matplotlib.pyplot as plt
            fig, ax = plt.subplots()
            ax.plot([0, 4], [1, 3])
            ax.set(xlim=(-2, 6), xlabel="Tax rate")
            lo, hi = ax.get_xlim()
            print(lo, hi, ax.get_xlabel())
            """,
            "The explicit limits replace the automatic ones; `get_xlabel` returns the label text.", level=1),
        out("o2", "What does this print?",
            """
            import matplotlib.pyplot as plt
            fig, ax = plt.subplots()
            ax.plot([0, 10], [0, 5])
            ax.set_xlim(10, 0)
            print(ax.xaxis_inverted())
            """,
            "A larger left limit than right limit inverts the x axis.", level=2),
        task("c1",
             """
             Plot `share` against `age` on a new axes. Give it the title
             "Labour force participation", the x label "Age", the y label
             "Share", and y limits from 0 to 1.
             """,
             """
             import matplotlib.pyplot as plt
             fig, ax = plt.subplots()
             ax.plot(age, share)
             ax.set(title="Labour force participation", xlabel="Age", ylabel="Share", ylim=(0, 1))
             """,
             [
                 check("labels", """
                     ax = figures()[0].axes[0]
                     check(ax.get_title() == "Labour force participation", "Set the title exactly as given.")
                     check(ax.get_xlabel() == "Age" and ax.get_ylabel() == "Share", "Label the x axis Age and the y axis Share.")
                 """),
                 check("limits", """
                     ax = figures()[0].axes[0]
                     check(len(ax.lines) == 1, "Plot one line.")
                     check(tuple(float(v) for v in ax.get_ylim()) == (0.0, 1.0), "The y axis should run from 0 to 1.")
                 """),
             ],
             "`ax.set` takes every property as a keyword; the separate `set_title`, `set_xlabel`, ... calls work too.",
             starter="""
             import matplotlib.pyplot as plt
             fig, ax = plt.subplots()
             ax.plot(age, share)
             """,
             setup="""
             import numpy as np
             age = np.arange(20, 70, 5)
             share = np.array([0.71, 0.82, 0.84, 0.84, 0.83, 0.81, 0.77, 0.71, 0.60, 0.42])
             """, level=1),
    ]),

# ---------------------------------------------------------------- 4.02
concept(
    "py.mpl.scatter", "Scatter plots", ["py.mpl.line_styles"],
    "4.02", "Simple Scatter Plots", "04.02-simple-scatter-plots",
    """
    Two ways to draw points. `ax.plot` with a marker and no line style draws
    every point the same way:

    ```python
    ax.plot(educ, wage, "o")                 # circles, no connecting line
    ax.scatter(educ, wage, s=hours, c=age, cmap="viridis", alpha=0.5)
    ```

    `ax.scatter` lets **each point** have its own size (`s`, in points
    squared) and color (`c`, mapped through a colormap), so a third and
    fourth variable can ride along with x and y. `alpha` makes overlapping
    points visible.

    The price of that flexibility is speed: scatter keeps per-point
    properties, so for many thousands of identical points `plot` with a
    marker is the leaner choice. `plot` creates a `Line2D` in `ax.lines`;
    `scatter` creates a `PathCollection` in `ax.collections`.
    """,
    [
        mc("m1", "You want marker size to show each firm's employment and color to show its sector code. Which function fits?",
           "`ax.scatter`, with `s=` for size and `c=` for color",
           ["`ax.plot` with `\"o\"`", "`ax.hist`", "`ax.fill_between`"],
           "Only `scatter` maps per-point values to size and color.", level=1),
        mc("m2", "Which call draws markers only, with no connecting line?",
           "`ax.plot(x, y, \"s\")`", ["`ax.plot(x, y, \"-\")`", "`ax.plot(x, y)`", "`ax.plot(x, y, \"--\")`"],
           """
           A format string with a marker (`s` is a square) and no line style
           draws markers alone. The default with no format is a line.
           """, level=1),
        mc("m3", "Why can `ax.plot(x, y, \"o\")` be faster than `ax.scatter(x, y)` for 200,000 points?",
           "All markers share one size and color, so they do not need per-point properties",
           ["`plot` samples a subset of the points", "`scatter` sorts the data first", "`plot` draws on the GPU"],
           """
           `scatter` is built to style every point individually; when every
           point looks the same that bookkeeping is wasted work.
           """, level=2),
        out("o1", "What does this print?",
            """
            import matplotlib.pyplot as plt
            fig, ax = plt.subplots()
            ax.plot([1, 2, 3], [3, 1, 2], "o")
            ax.scatter([1, 2, 3], [1, 2, 3])
            print(len(ax.lines), len(ax.collections))
            """,
            "The marker plot is a line object (with no line drawn); the scatter is a collection.", level=2),
        out("o2", "What does this print?",
            """
            import matplotlib.pyplot as plt
            fig, ax = plt.subplots()
            pts = ax.scatter([1, 2, 3, 4], [4, 3, 2, 1], s=[10, 40, 90, 160])
            print(pts.get_sizes(), len(pts.get_offsets()))
            """,
            "Sizes are stored per point; the offsets are the four (x, y) positions.", level=1),
        task("c1",
             """
             `df` holds one row per household with columns `income`,
             `spending` and `size`. Make a scatter plot of `spending` (y)
             against `income` (x), with marker size equal to `size` times 20
             and transparency `alpha=0.6`.
             """,
             """
             import matplotlib.pyplot as plt
             fig, ax = plt.subplots()
             ax.scatter(df["income"], df["spending"], s=df["size"] * 20, alpha=0.6)
             """,
             [
                 check("scatter", """
                     ax = figures()[0].axes[0]
                     check(len(ax.collections) == 1, "Use ax.scatter to draw one set of points.")
                     pts = ax.collections[0]
                     xy = pts.get_offsets()
                     check(close(xy[:, 0], df["income"]) and close(xy[:, 1], df["spending"]), "Put income on x and spending on y.")
                 """),
                 check("style", """
                     pts = figures()[0].axes[0].collections[0]
                     check(close(pts.get_sizes(), df["size"] * 20), "Marker sizes should be household size times 20.")
                     check(pts.get_alpha() == 0.6, "Set alpha to 0.6.")
                 """),
             ],
             "`s` accepts an array, one size per point; `alpha` is a single number for the whole collection.",
             starter="""
             import matplotlib.pyplot as plt
             fig, ax = plt.subplots()
             ax.plot(df["income"], df["spending"], "o")
             """,
             setup="""
             import pandas as pd
             df = pd.DataFrame({"income": [28, 35, 41, 52, 60, 75],
                                "spending": [25, 30, 33, 40, 44, 51],
                                "size": [1, 3, 2, 4, 2, 5]})
             """, level=2),
    ]),

# ---------------------------------------------------------------- 4.03
concept(
    "py.mpl.errorbars", "Error bars and uncertainty bands", ["py.mpl.scatter"],
    "4.03", "Visualizing Errors", "04.03-errorbars",
    """
    Estimates come with uncertainty, and a plot should show it. For a handful
    of point estimates, `ax.errorbar` draws a marker with a bar of the given
    half-width:

    ```python
    ax.errorbar(groups, coef, yerr=1.96 * se, fmt="o", capsize=4)
    ```

    `fmt` uses the same format codes as `plot`; `capsize` adds small caps at
    the bar ends. `yerr` can be a single number, one value per point, or a
    2 x n array for asymmetric bars.

    For a continuous curve, such as a forecast with a confidence interval,
    a shaded band reads better than hundreds of bars:

    ```python
    ax.plot(t, fit)
    ax.fill_between(t, fit - 2 * sd, fit + 2 * sd, alpha=0.3)
    ```

    `fill_between(x, lower, upper)` fills the region between the two curves.
    """,
    [
        mc("m1", "You have 6 regression coefficients and their standard errors. Which call shows 95% intervals?",
           "`ax.errorbar(x, coef, yerr=1.96 * se, fmt=\"o\")`",
           ["`ax.fill_between(x, coef, se)`", "`ax.plot(x, coef, yerr=se)`", "`ax.scatter(x, coef, s=se)`"],
           "`yerr` is the half-width of each bar; 1.96 standard errors gives a normal 95% interval.", level=1),
        mc("m2", "What are the three positional arguments of `ax.fill_between(x, a, b)`?",
           "The x values, the lower curve and the upper curve",
           ["x, y and the error half-width", "The left edge, right edge and color", "x, y and the transparency"],
           "It shades the region between `a` and `b` at each x.", level=2),
        mc("m3", "How do you give `errorbar` an interval that is longer above the point than below?",
           "Pass `yerr` as a 2 x n array: first row below, second row above",
           ["Pass two `errorbar` calls", "Use `capsize` with two values", "It is not possible"],
           "A 2 x n `yerr` holds separate lower and upper half-widths for each point.", level=3),
        out("o1", "What does this print?",
            """
            import matplotlib.pyplot as plt
            fig, ax = plt.subplots()
            out = ax.errorbar([1, 2, 3], [0.4, 0.1, 0.6], yerr=0.2, fmt="o")
            print(type(out).__name__, len(ax.containers))
            """,
            """
            `errorbar` returns an `ErrorbarContainer` that groups the marker
            line and the bar collection, and registers it in `ax.containers`.
            """, level=2),
        task("c1",
             """
             `t`, `forecast` and `sd` are arrays. Plot `forecast` against `t`
             as a line, then shade the band from `forecast - 2*sd` to
             `forecast + 2*sd` with `alpha=0.25`.
             """,
             """
             import matplotlib.pyplot as plt
             fig, ax = plt.subplots()
             ax.plot(t, forecast)
             ax.fill_between(t, forecast - 2 * sd, forecast + 2 * sd, alpha=0.25)
             """,
             [
                 check("line", """
                     ax = figures()[0].axes[0]
                     check(len(ax.lines) == 1 and close(ax.lines[0].get_ydata(), forecast), "Plot the forecast as one line.")
                 """),
                 check("band", """
                     ax = figures()[0].axes[0]
                     check(len(ax.collections) == 1, "Add one shaded band with fill_between.")
                     band = ax.collections[0]
                     ys = band.get_paths()[0].vertices[:, 1]
                     check(abs(ys.min() - (forecast - 2 * sd).min()) < 1e-9 and abs(ys.max() - (forecast + 2 * sd).max()) < 1e-9,
                           "The band should run from forecast - 2*sd to forecast + 2*sd.")
                     check(band.get_alpha() == 0.25, "Set alpha to 0.25.")
                 """),
             ],
             "`fill_between` takes x, the lower edge and the upper edge; draw the line first or after, either works.",
             starter="""
             import matplotlib.pyplot as plt
             fig, ax = plt.subplots()
             ax.plot(t, forecast)
             """,
             setup="""
             import numpy as np
             t = np.arange(12)
             forecast = 2.0 + 0.1 * t
             sd = 0.05 + 0.02 * t
             """, level=2),
        task("c2",
             """
             Plot the four estimates in `est` at x positions 0 to 3 with
             error bars of half-width `se`, using circle markers and no
             connecting line (`fmt="o"`), and caps of size 5.
             """,
             """
             import matplotlib.pyplot as plt
             import numpy as np
             fig, ax = plt.subplots()
             ax.errorbar(np.arange(4), est, yerr=se, fmt="o", capsize=5)
             """,
             [
                 check("container", """
                     ax = figures()[0].axes[0]
                     check(len(ax.containers) == 1, "Use ax.errorbar once.")
                     data_line, caps, bars = ax.containers[0].lines
                     check(close(data_line.get_ydata(), est), "Plot the estimates in `est`.")
                     check(data_line.get_linestyle() == "None", "Use fmt=\\"o\\" so the points are not joined.")
                 """),
                 check("caps", """
                     data_line, caps, bars = figures()[0].axes[0].containers[0].lines
                     check(len(caps) > 0 and caps[0].get_markersize() == 10, "Add caps of size 5 to the bars with capsize.")
                 """),
             ],
             "`fmt=\"o\"` gives markers without a line, and `capsize=5` draws a cap 5 points long at each end of every bar.",
             starter="""
             import matplotlib.pyplot as plt
             import numpy as np
             fig, ax = plt.subplots()
             ax.errorbar(np.arange(4), est)
             """,
             setup="""
             import numpy as np
             est = np.array([0.12, 0.30, 0.05, 0.22])
             se = np.array([0.04, 0.06, 0.05, 0.03])
             """, level=2),
    ]),

# ---------------------------------------------------------------- 4.04
concept(
    "py.mpl.contour", "Density and contour plots", ["py.mpl.interfaces", "py.np.broadcast_practice"],
    "4.04", "Density and Contour Plots", "04.04-density-and-contour-plots",
    """
    To picture a function of two variables, such as profit over price and
    quantity, evaluate it on a grid. `np.meshgrid` turns two 1D axes into
    two 2D coordinate arrays:

    ```python
    k = np.linspace(1, 10, 50)
    l = np.linspace(1, 10, 60)
    K, L = np.meshgrid(k, l)          # both shape (60, 50)
    Y = K**0.3 * L**0.7               # output on the grid
    ax.contour(K, L, Y, levels=8, colors="k")   # isoquant lines
    ax.contourf(K, L, Y, levels=20, cmap="viridis")  # filled bands
    ax.imshow(Y, origin="lower", extent=[1, 10, 1, 10], aspect="auto")
    ```

    `contour` draws lines of equal value; `contourf` fills between them.
    `imshow` paints the array as an image. It puts row 0 at the **top** by
    default, so pass `origin="lower"` and an `extent` to line it up with
    the x and y values.
    """,
    [
        mc("m1", "With `x` of length 40 and `y` of length 30, what shape is `X` in `X, Y = np.meshgrid(x, y)`?",
           "`(30, 40)`", ["`(40, 30)`", "`(40,)`", "`(1200,)`"],
           "Rows follow `y` and columns follow `x`, so the grid has 30 rows and 40 columns.", level=2),
        mc("m2", "What is the difference between `contour` and `contourf`?",
           "`contour` draws lines of equal value; `contourf` fills the regions between them",
           ["`contourf` is a faster version of `contour`", "`contourf` works on functions, `contour` on data", "There is none"],
           "The `f` is for filled.", level=1),
        mc("m3", "Your `imshow` picture appears upside down compared with a contour plot of the same array. Why?",
           "`imshow` places row 0 at the top by default; use `origin=\"lower\"`",
           ["`imshow` transposes the array", "The colormap is reversed", "Contour plots flip the y axis"],
           "Images follow the matrix convention (row 0 on top); contour plots follow the y axis (small values at the bottom).", level=2),
        out("o1", "What does this print?",
            """
            import numpy as np
            X, Y = np.meshgrid(np.arange(3), np.arange(2))
            print(X)
            print(Y)
            """,
            "`X` repeats the x values down the rows; `Y` repeats the y values across the columns.", level=1),
        out("o2", "What does this print?",
            """
            import numpy as np
            import matplotlib.pyplot as plt
            x = np.linspace(0, 1, 20)
            X, Y = np.meshgrid(x, x)
            fig, ax = plt.subplots()
            cs = ax.contour(X, Y, X + Y, levels=[0.5, 1.0, 1.5])
            print(cs.levels)
            """,
            "When you pass a list of levels, the contour set uses exactly those.", level=2),
        task("c1",
             """
             Build a 2D grid from `price` (x) and `qty` (y) with `np.meshgrid`,
             compute `revenue = P * Q` on it, and draw a filled contour plot
             of revenue with 10 levels. Add a colorbar for it.
             """,
             """
             import numpy as np
             import matplotlib.pyplot as plt
             P, Q = np.meshgrid(price, qty)
             revenue = P * Q
             fig, ax = plt.subplots()
             cf = ax.contourf(P, Q, revenue, levels=10)
             fig.colorbar(cf)
             """,
             [
                 check("grid", """
                     check(getattr(get("revenue"), "shape", None) == (len(qty), len(price)), "`revenue` should be a 2D grid with one row per qty and one column per price.")
                     check(close(revenue[3, 5], qty[3] * price[5]), "`revenue` should be price times quantity at each grid point.")
                 """),
                 check("contourf", """
                     from matplotlib.contour import ContourSet
                     fig = figures()[0]
                     ax = fig.axes[0]
                     sets = [c for c in ax.collections if isinstance(c, ContourSet)]
                     check(sets and sets[0].filled, "Draw a filled contour plot with contourf.")
                 """),
                 check("colorbar", """
                     check(len(figures()[0].axes) == 2, "Add a colorbar (it appears as a second axes).")
                 """),
             ],
             "`meshgrid` gives coordinate arrays; elementwise `P * Q` is the revenue grid. `fig.colorbar` takes the contour set.",
             starter="""
             import numpy as np
             import matplotlib.pyplot as plt
             P, Q = ...
             revenue = ...
             """,
             setup="""
             import numpy as np
             price = np.linspace(1, 20, 40)
             qty = np.linspace(0, 100, 30)
             """, level=2),
    ]),

# ---------------------------------------------------------------- 4.05
concept(
    "py.mpl.histograms", "Histograms and binning", ["py.mpl.interfaces", "py.np.aggregates"],
    "4.05", "Histograms, Binnings, and Density", "04.05-histograms-and-binnings",
    """
    A histogram counts how many values fall into each bin.

    ```python
    rng = np.random.default_rng(1)
    wages = rng.lognormal(3, 0.5, 2000)
    counts, edges, bars = ax.hist(wages, bins=30, density=False,
                                  histtype="stepfilled", alpha=0.5)
    counts, edges = np.histogram(wages, bins=30)   # same numbers, no plot
    ```

    `bins` can be a count or a list of edges. n bins have n + 1 edges.
    `density=True` rescales so the bars' total area is 1, which lets you
    overlay groups of different sizes.

    For two variables, bin in 2D: `ax.hist2d(x, y, bins=20)` uses square
    cells, and `ax.hexbin(x, y, gridsize=25)` uses hexagons, which tile the
    plane with less visual bias. `np.histogram2d` gives the counts alone.
    """,
    [
        mc("m1", "`ax.hist(data, bins=12)` returns `(counts, edges, patches)`. How long is `edges`?",
           "13", ["12", "11", "It depends on the data"],
           "Twelve bins need thirteen edges: one start plus one end per bin.", level=1),
        mc("m2", "Why set `density=True` when overlaying histograms of two groups of very different sizes?",
           "Each histogram is scaled to unit area, so the shapes are comparable",
           ["It makes the bins narrower", "It removes outliers", "It sorts the data first"],
           "Raw counts would make the larger group dwarf the smaller one.", level=2),
        mc("m3", "You only need the bin counts, not a picture. What should you call?",
           "`np.histogram(data, bins=...)`", ["`ax.hist` then delete the figure", "`np.bincount(data)` on floats", "`plt.hist2d`"],
           "`np.histogram` returns counts and edges without creating any artists.", level=1),
        mc("m4", "Which call bins two variables into hexagonal cells?",
           "`ax.hexbin(x, y, gridsize=20)`", ["`ax.hist2d(x, y, bins=\"hex\")`", "`ax.hist(x, y, histtype=\"hex\")`", "`np.histogram2d(x, y, shape=6)`"],
           "`hexbin` is the hexagonal binning function; `hist2d` uses rectangles.", level=1),
        out("o1", "What does this print?",
            """
            import numpy as np
            counts, edges = np.histogram([2, 3, 3, 5, 8, 9, 9, 9], bins=4, range=(0, 12))
            print(counts)
            print(edges)
            """,
            "Four bins of width 3 from 0 to 12: [0,3), [3,6), [6,9), [9,12]. The last bin includes its right edge.", level=2),
        out("o2", "What does this print?",
            """
            import matplotlib.pyplot as plt
            fig, ax = plt.subplots()
            counts, edges, bars = ax.hist([1, 1, 2, 4, 4, 4], bins=[0, 2, 4, 6])
            print(counts, len(ax.patches))
            """,
            "Edges 0, 2, 4, 6 make three bins. 4 sits on an inner edge, so it goes to the bin starting at 4.", level=2),
        task("c1",
             """
             `hours` holds weekly hours for 500 workers. Draw a histogram of
             it with 20 bins, scaled with `density=True`, and keep the
             returned counts in `counts`.
             """,
             """
             import matplotlib.pyplot as plt
             fig, ax = plt.subplots()
             counts, edges, bars = ax.hist(hours, bins=20, density=True)
             """,
             [
                 check("bars", """
                     ax = figures()[0].axes[0]
                     check(len(ax.patches) == 20, "Use 20 bins.")
                 """),
                 check("density", """
                     import numpy as np
                     check(len(counts) == 20, "Keep the first return value of hist in `counts`.")
                     widths = np.array([p.get_width() for p in figures()[0].axes[0].patches])
                     check(abs((np.asarray(counts) * widths).sum() - 1) < 1e-6, "Scale the histogram with density=True so its area is 1.")
                 """),
             ],
             "`ax.hist` returns counts, edges and the bar patches; with `density=True` the counts are densities.",
             starter="""
             import matplotlib.pyplot as plt
             fig, ax = plt.subplots()
             counts = None
             """,
             setup="""
             import numpy as np
             hours = np.random.default_rng(3).normal(38, 6, 500)
             """, level=1),
    ]),

# ---------------------------------------------------------------- 4.06
concept(
    "py.mpl.legends", "Customizing legends", ["py.mpl.line_styles"],
    "4.06", "Customizing Plot Legends", "04.06-customizing-legends",
    """
    `ax.legend()` collects every artist that has a `label` and shows them.
    Labels that start with an underscore are left out, which is how you hide
    one series.

    ```python
    ax.plot(t, cpi, label="CPI")
    ax.plot(t, core, label="Core CPI")
    ax.plot(t, target, label="_target")       # not in the legend
    ax.legend(loc="upper left", frameon=False, ncols=2, title="Index")
    ```

    You can also choose exactly which handles appear:
    `ax.legend(handles=[l1, l2])`. For a size legend on a scatter plot,
    plot empty scatters with representative sizes and labels, then call
    `legend`.

    Each axes keeps only one legend: a second `legend()` call replaces the
    first. To keep two, add the first one back as an artist with
    `ax.add_artist(first_legend)` before creating the second.
    """,
    [
        mc("m1", "You plot three lines labelled `\"Exports\"`, `\"Imports\"` and `\"_trend\"` and call `ax.legend()`. How many entries appear?",
           "2", ["3", "1", "0"],
           "Labels starting with an underscore are skipped by `legend`.", level=1),
        mc("m2", "You call `ax.legend(...)` twice with different handles. What happens?",
           "Only the second legend is shown; the first is replaced",
           ["Both legends are shown", "An error is raised", "The entries are merged into one legend"],
           "An axes has one legend slot. Keep the first by `ax.add_artist(first)` before making the second.", level=2),
        mc("m3", "Which keyword lays legend entries out in three columns?",
           "`ncols=3`", ["`columns=3`", "`loc=3`", "`shape=(1, 3)`"],
           "`ncols` sets the number of columns (the older spelling `ncol` still works).", level=2),
        out("o1", "What does this print?",
            """
            import matplotlib.pyplot as plt
            fig, ax = plt.subplots()
            ax.plot([0, 1], [1, 2], label="Urban")
            ax.plot([0, 1], [2, 1], label="_Rural")
            ax.plot([0, 1], [1, 1])
            leg = ax.legend()
            print([t.get_text() for t in leg.get_texts()])
            """,
            "The underscored label and the unlabelled line are both left out.", level=1),
        task("c1",
             """
             Plot `gdp` and `gni` against `year` with labels "GDP" and "GNI",
             and plot `trend` against `year` without it appearing in the
             legend. Add a legend in the lower right with the title
             "Series" and no frame.
             """,
             """
             import matplotlib.pyplot as plt
             fig, ax = plt.subplots()
             ax.plot(year, gdp, label="GDP")
             ax.plot(year, gni, label="GNI")
             ax.plot(year, trend)
             ax.legend(loc="lower right", title="Series", frameon=False)
             """,
             [
                 check("entries", """
                     ax = figures()[0].axes[0]
                     check(len(ax.lines) == 3, "Plot all three series.")
                     leg = ax.get_legend()
                     check(leg is not None, "Add a legend.")
                     check([t.get_text() for t in leg.get_texts()] == ["GDP", "GNI"], "The legend should list GDP and GNI only.")
                 """),
                 check("options", """
                     leg = figures()[0].axes[0].get_legend()
                     check(leg.get_title().get_text() == "Series", "Give the legend the title Series.")
                     check(not leg.get_frame_on(), "Turn the legend frame off.")
                     # Placement read from where the legend is drawn, not from a private attribute.
                     fig = figures()[0]
                     fig.canvas.draw()
                     lb = leg.get_window_extent()
                     ab = fig.axes[0].get_window_extent()
                     cx, cy = (lb.x0 + lb.x1) / 2, (lb.y0 + lb.y1) / 2
                     check(cx > (ab.x0 + ab.x1) / 2 and cy < (ab.y0 + ab.y1) / 2, "Place the legend in the lower right.")
                 """),
             ],
             "Unlabelled lines are skipped; `loc`, `title` and `frameon` control placement and look.",
             starter="""
             import matplotlib.pyplot as plt
             fig, ax = plt.subplots()
             ax.plot(year, gdp)
             ax.plot(year, gni)
             ax.plot(year, trend)
             ax.legend()
             """,
             setup="""
             import numpy as np
             year = np.arange(2015, 2023)
             gdp = np.array([100, 102, 105, 108, 110, 106, 112, 115])
             gni = gdp - np.array([1, 1, 2, 2, 2, 1, 3, 3])
             trend = np.linspace(100, 115, 8)
             """, level=2),
        task("c2",
             """
             Lines `a` and `b` are already plotted on `ax`. Show two
             legends: one with only `a` in the upper left, and another with
             only `b` in the lower right. Both must stay visible.
             """,
             """
             first = ax.legend(handles=[a], loc="upper left")
             ax.add_artist(first)
             ax.legend(handles=[b], loc="lower right")
             """,
             [
                 check("two legends", """
                     from matplotlib.legend import Legend
                     legs = [x for x in ax.get_children() if isinstance(x, Legend)]
                     check(len(legs) == 2, "Both legends should be on the axes. Add the first back with ax.add_artist.")
                 """),
                 check("contents", """
                     from matplotlib.legend import Legend
                     legs = [x for x in ax.get_children() if isinstance(x, Legend)]
                     texts = sorted(tuple(t.get_text() for t in l.get_texts()) for l in legs)
                     check(texts == [("Exports",), ("Imports",)], "Each legend should hold exactly one of the two lines.")
                 """),
             ],
             "A new `legend()` call replaces the stored legend, so re-attach the first with `add_artist`.",
             starter="""
             ax.legend(handles=[a], loc="upper left")
             ax.legend(handles=[b], loc="lower right")
             """,
             setup="""
             import matplotlib.pyplot as plt
             fig, ax = plt.subplots()
             a, = ax.plot([0, 1, 2], [1, 3, 2], label="Exports")
             b, = ax.plot([0, 1, 2], [2, 2, 3], label="Imports")
             """, level=3),
    ]),

# ---------------------------------------------------------------- 4.07
concept(
    "py.mpl.colorbars", "Colorbars and colormaps", ["py.mpl.contour", "py.mpl.scatter"],
    "4.07", "Customizing Colorbars", "04.07-customizing-colorbars",
    """
    When color encodes a number, a colorbar is the key. Pass it the artist
    that holds the color mapping:

    ```python
    im = ax.imshow(rates, cmap="viridis")
    cb = fig.colorbar(im, ax=ax, extend="both")
    cb.set_label("Unemployment rate")
    im.set_clim(2, 10)          # values outside 2..10 get the end colors
    ```

    Choose the colormap to match the data:

    - **sequential** (`viridis`, `Blues`) for values that run low to high,
    - **diverging** (`RdBu`, `coolwarm`) for values on either side of a
      meaningful midpoint, such as growth around zero,
    - **qualitative** (`tab10`) for unordered categories.

    Rainbow-style maps like `jet` have uneven brightness and can invent
    edges that are not in the data. Add `_r` to reverse any map. For a
    discrete key, ask for a map with a few colors: `plt.get_cmap("Blues", 5)`.
    """,
    [
        mc("m1", "You are mapping year-on-year price change, which can be negative or positive. Which kind of colormap fits?",
           "A diverging map such as `RdBu`", ["A sequential map such as `Blues`", "A qualitative map such as `tab10`", "`jet`"],
           "A diverging map gives zero a neutral color and pushes the two signs toward different hues.", level=1),
        mc("m2", "What does `extend=\"both\"` add to a colorbar?",
           "Pointed ends showing that some values lie beyond the color limits",
           ["A second colorbar", "Labels at both ends", "Extra ticks"],
           "The triangles signal clipping, useful after `set_clim` narrows the range.", level=2),
        mc("m3", "How do you get the reversed version of the `viridis` colormap?",
           "`cmap=\"viridis_r\"`", ["`cmap=\"-viridis\"`", "`cmap=\"viridis\", reverse=True`", "`cmap=\"rev_viridis\"`"],
           "Every built-in colormap has an `_r` twin.", level=1),
        mc("m4", "Why is `jet` a poor default for continuous data?",
           "Its brightness does not rise evenly, so it suggests boundaries the data do not have",
           ["It has only eight colors", "It cannot be used with `imshow`", "It is not built into Matplotlib"],
           "Perceptually uniform maps like `viridis` change brightness steadily with the value.", level=2),
        out("o1", "What does this print?",
            """
            import matplotlib.pyplot as plt
            cmap = plt.get_cmap("Blues", 4)
            print(cmap.N, cmap.name)
            """,
            "Asking for 4 colors gives a discrete colormap with `N == 4`.", level=2),
        out("o2", "What does this print?",
            """
            import numpy as np
            import matplotlib.pyplot as plt
            fig, ax = plt.subplots()
            im = ax.imshow(np.arange(12).reshape(3, 4))
            fig.colorbar(im)
            print(len(fig.axes), im.get_clim())
            """,
            "The colorbar lives in its own axes. The color limits default to the data range, 0 to 11.", level=2),
        task("c1",
             """
             `growth` is a 2D array of regional growth rates around zero.
             Show it with `imshow` using the `RdBu` colormap, set the color
             limits to -3 and 3, and add a colorbar with `extend="both"`.
             """,
             """
             import matplotlib.pyplot as plt
             fig, ax = plt.subplots()
             im = ax.imshow(growth, cmap="RdBu")
             im.set_clim(-3, 3)
             fig.colorbar(im, ax=ax, extend="both")
             """,
             [
                 check("image", """
                     fig = figures()[0]
                     ims = [a.images[0] for a in fig.axes if a.images]
                     check(ims, "Show `growth` with imshow.")
                     check(ims[0].get_cmap().name == "RdBu", "Use the RdBu colormap.")
                     check(tuple(float(v) for v in ims[0].get_clim()) == (-3.0, 3.0), "Set the color limits to -3 and 3.")
                 """),
                 check("colorbar", """
                     fig = figures()[0]
                     im = [a.images[0] for a in fig.axes if a.images][0]
                     cb = im.colorbar
                     check(cb is not None, "Add a colorbar for the image.")
                     check(cb.extend == "both", "Use extend=\\"both\\" on the colorbar.")
                 """),
             ],
             "`set_clim` fixes the range the colormap spans; `fig.colorbar(im, extend=...)` draws the key.",
             starter="""
             import matplotlib.pyplot as plt
             fig, ax = plt.subplots()
             im = ax.imshow(growth)
             """,
             setup="""
             import numpy as np
             growth = np.random.default_rng(5).normal(0, 2, (8, 10))
             """, level=2),
    ]),

# ---------------------------------------------------------------- 4.08
concept(
    "py.mpl.subplots", "Multiple subplots and grids", ["py.mpl.limits_labels"],
    "4.08", "Multiple Subplots", "04.08-multiple-subplots",
    """
    Several panels in one figure make comparisons easy. The usual tool is
    `plt.subplots`, which returns the figure and a NumPy array of axes:

    ```python
    fig, axs = plt.subplots(2, 3, sharex=True, sharey=True, figsize=(9, 5))
    axs[0, 2].plot(t, y)        # row 0, column 2
    for ax in axs.flat:
        ax.grid(True)
    ```

    `sharex`/`sharey` link the limits, so zooming one panel zooms them all
    and inner tick labels are hidden. With a single row or column the array
    is 1D (`axs[1]`).

    Other tools: `plt.subplot(rows, cols, k)` adds one panel by a 1-based
    index; `plt.axes([left, bottom, width, height])` places an axes anywhere,
    in figure fractions, which is good for insets; and `GridSpec` lets a
    panel span several cells: `fig.add_subplot(gs[0, :2])`.
    """,
    [
        mc("m1", "What is `axs.shape` after `fig, axs = plt.subplots(3, 2)`?",
           "`(3, 2)`", ["`(2, 3)`", "`(6,)`", "It is a list, so it has no shape"],
           "Rows first, then columns, like any NumPy array.", level=1),
        mc("m2", "In `plt.subplot(2, 2, k)`, which `k` is the bottom-left panel?",
           "3", ["2", "4", "1"],
           "The index starts at 1 and counts across each row: 1 2 / 3 4.", level=1),
        mc("m3", "What does `sharey=True` do in `plt.subplots(1, 3, sharey=True)`?",
           "All three panels use the same y limits, and only the first shows y tick labels",
           ["All panels plot the same y data", "Each panel gets its own y limits", "The panels are stacked vertically"],
           "Shared axes are linked; the inner tick labels are hidden because they would repeat.", level=2),
        mc("m4", "`plt.axes([0.65, 0.6, 0.25, 0.25])` creates an axes where?",
           "An inset in the upper right, a quarter of the figure wide and tall",
           ["At data coordinates (0.65, 0.6)", "In the lower left corner", "As the fourth subplot of a grid"],
           "The list is left, bottom, width and height as fractions of the figure.", level=2),
        out("o1", "What does this print?",
            """
            import matplotlib.pyplot as plt
            fig, axs = plt.subplots(1, 4)
            print(axs.shape, len(fig.axes))
            """,
            "A single row gives a 1D array of four axes.", level=1),
        out("o2", "What does this print?",
            """
            import matplotlib.pyplot as plt
            fig, axs = plt.subplots(2, 2, sharex=True)
            axs[0, 0].set_xlim(0, 50)
            lo, hi = axs[1, 1].get_xlim()
            print(lo, hi)
            """,
            "With `sharex=True`, setting one panel's x limits sets all of them.", level=2),
        task("c1",
             """
             Make a figure with 2 rows and 2 columns of axes that share the
             y axis. In each panel, plot one column of `df` (columns in order:
             top left, top right, bottom left, bottom right) and set the panel
             title to the column name.
             """,
             """
             import matplotlib.pyplot as plt
             fig, axs = plt.subplots(2, 2, sharey=True)
             for ax, col in zip(axs.flat, df.columns):
                 ax.plot(df.index, df[col])
                 ax.set_title(col)
             """,
             [
                 check("grid", """
                     fig = figures()[0]
                     check(len(fig.axes) == 4, "Make a 2 x 2 grid of axes.")
                     a = fig.axes
                     check(a[0].get_shared_y_axes().joined(a[0], a[3]), "Share the y axis across the panels.")
                 """),
                 check("panels", """
                     fig = figures()[0]
                     for ax, col in zip(fig.axes, df.columns):
                         check(ax.get_title() == col, "Title each panel with its column name, in order.")
                         check(len(ax.lines) == 1 and close(ax.lines[0].get_ydata(), df[col]), "Panel " + col + " should plot that column.")
                 """),
             ],
             "`axs.flat` walks the panels row by row; `zip` pairs them with the column names.",
             starter="""
             import matplotlib.pyplot as plt
             fig, axs = plt.subplots(2, 2)
             """,
             setup="""
             import pandas as pd
             df = pd.DataFrame({"North": [5.1, 5.0, 4.8, 4.6], "South": [6.2, 6.4, 6.1, 5.9],
                                "East": [4.0, 4.1, 4.3, 4.2], "West": [5.5, 5.2, 5.0, 5.1]},
                               index=[2021, 2022, 2023, 2024])
             """, level=2),
        task("c2",
             """
             Using `GridSpec` with 2 rows and 3 columns on a new figure, add a
             wide axes `top` spanning the whole first row, and an axes
             `side` in the second row spanning the last two columns.
             """,
             """
             import matplotlib.pyplot as plt
             fig = plt.figure()
             gs = plt.GridSpec(2, 3)
             top = fig.add_subplot(gs[0, :])
             side = fig.add_subplot(gs[1, 1:])
             """,
             [
                 check("top", """
                     spec = top.get_subplotspec()
                     check(spec.rowspan == range(0, 1) and spec.colspan == range(0, 3), "`top` should span all three columns of row 0.")
                 """),
                 check("side", """
                     spec = side.get_subplotspec()
                     check(spec.rowspan == range(1, 2) and spec.colspan == range(1, 3), "`side` should cover row 1, columns 1 and 2.")
                 """),
             ],
             "Slice the GridSpec like an array: `gs[0, :]` is the whole first row, `gs[1, 1:]` the last two cells of the second.",
             starter="""
             import matplotlib.pyplot as plt
             fig = plt.figure()
             gs = plt.GridSpec(2, 3)
             top = fig.add_subplot(gs[0, 0])
             side = fig.add_subplot(gs[1, 0])
             """, level=3),
    ]),

# ---------------------------------------------------------------- 4.09
concept(
    "py.mpl.annotation", "Text, transforms and annotations", ["py.mpl.limits_labels"],
    "4.09", "Text and Annotation", "04.09-text-and-annotation",
    """
    `ax.text(x, y, s)` writes text at a position. By default the position is
    in **data coordinates** (`transform=ax.transData`), so the text moves with
    the data when limits change. Other coordinate systems:

    - `ax.transAxes`: (0, 0) is the axes' lower-left corner and (1, 1) the
      upper-right, whatever the data limits,
    - `fig.transFigure`: the same idea for the whole figure.

    ```python
    ax.plot(year, rate)
    ax.text(0.02, 0.95, "Source: survey", transform=ax.transAxes, va="top")
    ax.annotate("Policy change", xy=(2020, 6.1), xytext=(2016, 8),
                arrowprops=dict(arrowstyle="->"))
    ```

    `annotate` points from the text (`xytext`) to a data point (`xy`) with an
    arrow. `ha` and `va` set horizontal and vertical alignment.
    """,
    [
        mc("m1", "You want a note to sit in the top-left corner of the axes no matter how the data limits change. What transform should it use?",
           "`ax.transAxes`", ["`ax.transData`", "No transform; use the data's maximum", "`plt.gcf()`"],
           "Axes coordinates run from 0 to 1 across the panel, independent of the data.", level=1),
        mc("m2", "In `ax.annotate(\"Peak\", xy=(5, 9), xytext=(2, 11), arrowprops=dict(arrowstyle=\"->\"))`, where does the arrow point?",
           "At the point (5, 9), from the text at (2, 11)", ["At (2, 11), from (5, 9)", "Nowhere; arrowprops only colors the text", "From the origin to (5, 9)"],
           "`xy` is the target; `xytext` is where the label sits.", level=1),
        mc("m3", "What coordinate system does `ax.text(3, 4, \"x\")` use when no transform is given?",
           "Data coordinates", ["Axes fractions", "Figure fractions", "Pixels"],
           "The default transform is `ax.transData`.", level=1),
        out("o1", "What does this print?",
            """
            import matplotlib.pyplot as plt
            fig, ax = plt.subplots()
            ax.plot([0, 10], [0, 10])
            ax.text(5, 5, "mid")
            ax.text(0.5, 0.9, "note", transform=ax.transAxes)
            ax.annotate("end", xy=(10, 10), xytext=(6, 9), arrowprops=dict(arrowstyle="->"))
            print(len(ax.texts))
            """,
            "Text objects and annotations are both stored in `ax.texts`.", level=2),
        task("c1",
             """
             On the plotted series, add the text "Recession" with
             `ax.annotate`, placing the label at (2006, 9) and an arrow
             pointing to the data point (2009, 9.3). Also put the text
             "Monthly data" at the bottom-right corner of the axes,
             position (0.98, 0.02) in axes coordinates.
             """,
             """
             ax.annotate("Recession", xy=(2009, 9.3), xytext=(2006, 9), arrowprops=dict(arrowstyle="->"))
             ax.text(0.98, 0.02, "Monthly data", transform=ax.transAxes, ha="right")
             """,
             [
                 check("annotate", """
                     from matplotlib.text import Annotation
                     anns = [t for t in ax.texts if isinstance(t, Annotation)]
                     check(anns and anns[0].get_text() == "Recession", "Add the Recession annotation with ax.annotate.")
                     a = anns[0]
                     check(tuple(a.xy) == (2009, 9.3), "The arrow should point at (2009, 9.3).")
                     check(tuple(a.xyann) == (2006, 9), "Put the label at (2006, 9).")
                     check(a.arrow_patch is not None, "Give the annotation an arrow with arrowprops.")
                 """),
                 check("corner", """
                     notes = [t for t in ax.texts if t.get_text() == "Monthly data"]
                     check(notes, "Add the text Monthly data.")
                     check(notes[0].get_transform() == ax.transAxes, "Place Monthly data in axes coordinates (transform=ax.transAxes).")
                     check(tuple(notes[0].get_position()) == (0.98, 0.02), "Put Monthly data at (0.98, 0.02).")
                 """),
             ],
             "`xy` is the target in data coordinates; `transform=ax.transAxes` makes (0.98, 0.02) a fraction of the axes.",
             starter="""
             ax.text(2006, 9, "Recession")
             """,
             setup="""
             import numpy as np
             import matplotlib.pyplot as plt
             fig, ax = plt.subplots()
             yr = np.arange(2000, 2016)
             ax.plot(yr, [4.0, 4.7, 5.8, 6.0, 5.5, 5.1, 4.6, 4.6, 5.8, 9.3, 9.6, 8.9, 8.1, 7.4, 6.2, 5.3])
             """, level=2),
        mc("m4", "In `fig.transFigure` coordinates, where is the point (0, 0)?",
           "The lower-left corner of the whole figure",
           ["The lower-left corner of the axes", "The upper-left corner of the figure", "The data point x = 0, y = 0"],
           """
           Figure coordinates are fractions of the figure, measured from its
           lower-left corner, so (1, 1) is the upper-right corner. Axes
           coordinates work the same way but for the axes box.
           """, level=1),
        mc("m5", "One label is placed with the default transform and another with `transform=ax.transAxes`. You then call `ax.set_xlim(0, 100)`. What happens?",
           "The data-coordinate label moves with the data; the axes-coordinate label stays put on the panel",
           ["Both labels move with the data, because every transform follows the axis limits",
            "Neither label moves, because text is fixed once drawn until you call `set_position`",
            "The axes-coordinate label moves with the limits and the data-coordinate label stays put"],
           """
           Data coordinates are tied to the axis limits, so changing the limits
           moves anything positioned in them. Axes and figure coordinates are
           fractions of a box and ignore the limits.
           """, level=2),
        mc("m6", "Why is `ax.annotate` usually a better way to draw an arrow than `plt.arrow`?",
           "`plt.arrow` draws a shape that unequal x and y scales distort; `annotate` arrows keep their look and attach to text",
           ["`plt.arrow` cannot point at a data coordinate, only at axes fractions between 0 and 1",
            "`annotate` is the only call that lets you set the arrow's color and line width",
            "`plt.arrow` was removed from Matplotlib, so only `annotate` still draws arrows"],
           """
           An arrow from `plt.arrow` is a patch drawn in data space, so a plot
           with different x and y scales stretches it. `annotate` styles the
           arrow with `arrowprops` (`arrowstyle`, `connectionstyle`, colors)
           and keeps it tidy.
           """, level=2),
        out("o2", "What does this print?",
            """
            import matplotlib.pyplot as plt
            fig, ax = plt.subplots()
            ax.plot([0, 10], [0, 10])
            note = ax.text(0.5, 0.5, "centre", transform=ax.transAxes)
            ax.set_xlim(0, 100)
            print(note.get_position(), note.get_transform() == ax.transAxes)
            """,
            """
            A text object keeps the position you gave it, in the coordinate
            system of its transform. Changing the data limits does not touch a
            position measured in axes fractions.
            """, level=2),
        task("c2",
             """
             The series `y` is plotted against `x`. Annotate its highest point
             with the text "Peak": the arrow should point at the highest data
             point, the label should sit at (`x` of the peak minus 2, the peak
             value plus 1), the arrow should use `arrowstyle="->"` with a
             curved `connectionstyle="arc3,rad=-0.2"`, and the label should be
             in a rounded box (`bbox=dict(boxstyle="round", fc="w")`).
             """,
             """
             import numpy as np
             i = int(np.argmax(y))
             ax.annotate("Peak", xy=(x[i], y[i]), xytext=(x[i] - 2, y[i] + 1),
                         arrowprops=dict(arrowstyle="->", connectionstyle="arc3,rad=-0.2"),
                         bbox=dict(boxstyle="round", fc="w"))
             """,
             [
                 check("annotation", """
                     from matplotlib.text import Annotation
                     anns = [t for t in ax.texts if isinstance(t, Annotation) and t.get_text() == "Peak"]
                     check(anns, "Add the Peak annotation with ax.annotate.")
                     a = anns[0]
                     check(tuple(float(v) for v in a.xy) == (6.0, 8.5), "The arrow should point at the highest data point, (6, 8.5).")
                     check(tuple(float(v) for v in a.xyann) == (4.0, 9.5), "Place the label 2 left of and 1 above the peak.")
                 """),
                 check("arrow", """
                     from matplotlib.text import Annotation
                     a = [t for t in ax.texts if isinstance(t, Annotation) and t.get_text() == "Peak"][0]
                     props = a.arrowprops or {}
                     check(a.arrow_patch is not None and props.get("arrowstyle") == "->", "Draw the arrow with arrowstyle='->'.")
                     check(props.get("connectionstyle") == "arc3,rad=-0.2", "Curve the arrow with connectionstyle='arc3,rad=-0.2'.")
                 """),
                 check("box", """
                     from matplotlib.text import Annotation
                     a = [t for t in ax.texts if isinstance(t, Annotation) and t.get_text() == "Peak"][0]
                     bp = a.get_bbox_patch()
                     check(bp is not None and type(bp.get_boxstyle()).__name__ == "Round", "Put the label in a rounded box with bbox=dict(boxstyle='round').")
                 """),
             ],
             "`np.argmax` finds the peak; `xy` is the arrow's target, `xytext` the label, `arrowprops` styles the arrow and `bbox` boxes the text.",
             starter="""
             ax.text(4, 9.5, "Peak")
             """,
             setup="""
             import numpy as np
             import matplotlib.pyplot as plt
             fig, ax = plt.subplots()
             x = np.arange(0, 11)
             y = np.array([1.0, 2.0, 3.5, 5.0, 6.5, 8.0, 8.5, 7.0, 5.5, 4.0, 3.0])
             ax.plot(x, y)
             """, level=2),
    ]),

# ---------------------------------------------------------------- 4.10
concept(
    "py.mpl.ticks", "Customizing ticks", ["py.mpl.limits_labels"],
    "4.10", "Customizing Ticks", "04.10-customizing-ticks",
    """
    Every axis has **major** and **minor** ticks. Where they go is decided by
    a *locator*; what their labels say is decided by a *formatter*.

    ```python
    ax.xaxis.set_major_locator(plt.MultipleLocator(5))   # a tick every 5 units
    ax.yaxis.set_major_locator(plt.MaxNLocator(4))       # at most about 4 ticks
    ax.yaxis.set_major_formatter(plt.FuncFormatter(lambda v, pos: f"{v:.0f}%"))
    ax.xaxis.set_major_formatter(plt.NullFormatter())    # keep ticks, hide labels
    ax.yaxis.set_major_locator(plt.NullLocator())        # no ticks at all
    ax.minorticks_on()
    ```

    A `FuncFormatter` function receives the tick value and its position and
    returns the label string. For a quick fixed set, `ax.set_xticks([...])`
    places ticks at exactly those values. Minor ticks are off by default on a
    linear axis (`NullLocator`) and on by default on a log axis.
    """,
    [
        mc("m1", "Which locator puts a tick at every multiple of 0.25?",
           "`plt.MultipleLocator(0.25)`", ["`plt.MaxNLocator(0.25)`", "`plt.FixedLocator(0.25)`", "`plt.LogLocator(0.25)`"],
           "`MultipleLocator(base)` places ticks at integer multiples of `base`.", level=1),
        mc("m2", "You want tick marks but no tick labels on the x axis. What do you set?",
           "`ax.xaxis.set_major_formatter(plt.NullFormatter())`",
           ["`ax.xaxis.set_major_locator(plt.NullLocator())`", "`ax.set_xlim(None)`", "`ax.set_xlabel(\"\")`"],
           "A `NullFormatter` blanks the labels; a `NullLocator` would remove the ticks themselves.", level=2),
        mc("m3", "What arguments does Matplotlib pass to the function inside `plt.FuncFormatter(f)`?",
           "The tick value and the tick position", ["Only the tick label string", "The axes and the value", "The data array"],
           "The function is called as `f(value, pos)` and must return a string.", level=2),
        out("o1", "What does this print?",
            """
            import matplotlib.pyplot as plt
            fig, ax = plt.subplots()
            ax.plot([0, 10], [0, 1])
            ax.set_xlim(0, 10)
            ax.xaxis.set_major_locator(plt.MultipleLocator(4))
            print([float(v) for v in ax.get_xticks()])
            """,
            """
            The locator returns multiples of 4 that cover the view, including
            one just beyond each end (Matplotlib only draws those inside the
            limits).
            """, level=3),
        out("o2", "What does this print?",
            """
            import matplotlib.pyplot as plt
            fig, ax = plt.subplots()
            ax.plot([0, 1], [0, 1])
            ax.set_xticks([0, 0.5, 1])
            ax.xaxis.set_major_formatter(plt.FuncFormatter(lambda v, pos: f"{v:.0%}"))
            fig.canvas.draw()
            print([t.get_text() for t in ax.get_xticklabels()])
            """,
            "The formatter turns each tick value into a percentage string.", level=2),
        task("c1",
             """
             `ax` shows a share between 0 and 1 on the y axis. Put y ticks
             every 0.2 with a `MultipleLocator`, and label them as whole
             percentages ("40%") with a `FuncFormatter`. Hide the x ticks
             entirely with a `NullLocator`.
             """,
             """
             import matplotlib.pyplot as plt
             ax.yaxis.set_major_locator(plt.MultipleLocator(0.2))
             ax.yaxis.set_major_formatter(plt.FuncFormatter(lambda v, pos: f"{v:.0%}"))
             ax.xaxis.set_major_locator(plt.NullLocator())
             """,
             [
                 check("y locator", """
                     import matplotlib.ticker as mt
                     loc = ax.yaxis.get_major_locator()
                     check(isinstance(loc, mt.MultipleLocator), "Use a MultipleLocator on the y axis.")
                     check(close(ax.get_yticks()[1] - ax.get_yticks()[0], 0.2), "Space the y ticks 0.2 apart.")
                 """),
                 check("y labels", """
                     fig.canvas.draw()
                     labels = [t.get_text() for t in ax.get_yticklabels()]
                     check("40%" in labels and "100%" in labels, "Label the y ticks as whole percentages, like 40%.")
                 """),
                 check("x hidden", """
                     check(len(ax.get_xticks()) == 0, "Remove the x ticks with a NullLocator.")
                 """),
             ],
             "Locators decide where, formatters decide what. Format code `.0%` multiplies by 100 and adds a percent sign.",
             starter="""
             import matplotlib.pyplot as plt
             ax.yaxis.set_major_locator(...)
             """,
             setup="""
             import matplotlib.pyplot as plt
             fig, ax = plt.subplots()
             ax.plot([1, 2, 3, 4, 5], [0.31, 0.45, 0.52, 0.66, 0.71])
             ax.set_ylim(0, 1)
             """, level=2),
    ]),

# ---------------------------------------------------------------- 4.11
concept(
    "py.mpl.styles", "Settings and stylesheets", ["py.mpl.line_styles"],
    "4.11", "Customizing Matplotlib: Configurations and Stylesheets", "04.11-settings-and-stylesheets",
    """
    Defaults such as line width, font size and figure size live in the
    dictionary-like `plt.rcParams`. Changing a key affects every figure made
    afterwards in the session:

    ```python
    plt.rcParams["lines.linewidth"] = 2.5
    plt.rcParams["axes.grid"] = True
    plt.rcParams.update({"font.size": 11, "figure.figsize": (7, 4)})
    ```

    A **stylesheet** is a named bundle of these settings. `plt.style.use("ggplot")`
    applies one for the rest of the session; `plt.style.available` lists the
    built-in names. To try a style for one figure only, use it as a context
    manager:

    ```python
    with plt.style.context("bmh"):
        fig, ax = plt.subplots()
        ax.plot(x, y)
    # settings are back to what they were here
    ```

    `plt.rcdefaults()` restores the library defaults.
    """,
    [
        mc("m1", "What is the scope of `plt.rcParams[\"axes.grid\"] = True`?",
           "Every axes created afterwards in this session gets a grid",
           ["Only the current axes", "Only the next figure", "It is saved permanently to all future sessions"],
           "rcParams are session-wide defaults; existing axes keep the settings they were made with.", level=2),
        mc("m2", "How do you apply the `fivethirtyeight` style to one figure without affecting later ones?",
           "Create that figure inside `with plt.style.context(\"fivethirtyeight\"):`",
           ["`plt.style.use(\"fivethirtyeight\", once=True)`", "`fig.style = \"fivethirtyeight\"`", "`ax.set_style(\"fivethirtyeight\")`"],
           "The context manager restores the previous settings when the block exits.", level=1),
        mc("m3", "Where can you see the names of the stylesheets that ship with Matplotlib?",
           "`plt.style.available`", ["`plt.rcParams.keys()`", "`plt.styles()`", "`matplotlib.themes`"],
           "It is a list of style names you can pass to `use` or `context`.", level=1),
        out("o1", "What does this print?",
            """
            import matplotlib.pyplot as plt
            with plt.style.context({"lines.linewidth": 4}):
                fig, ax = plt.subplots()
                a, = ax.plot([0, 1])
            b, = ax.plot([1, 0])
            print(a.get_linewidth(), b.get_linewidth())
            """,
            """
            A dict works as a style too. Inside the block the default width is
            4; after it, the default 1.5 is back, even on the same axes.
            """, level=3),
        out("o2", "What does this print?",
            """
            import matplotlib.pyplot as plt
            plt.rcParams.update({"figure.figsize": (5, 2)})
            fig = plt.figure()
            print(fig.get_size_inches())
            """,
            "New figures take their size from `figure.figsize`, in inches.", level=1),
        task("c1",
             """
             Change the defaults so that every new axes has a grid and every
             line is 2.5 points wide, then create a figure and plot `y`.
             """,
             """
             import matplotlib.pyplot as plt
             plt.rcParams["axes.grid"] = True
             plt.rcParams["lines.linewidth"] = 2.5
             fig, ax = plt.subplots()
             ax.plot(y)
             """,
             [
                 check("rcParams", """
                     import matplotlib.pyplot as plt
                     check(plt.rcParams["axes.grid"] is True, "Turn on axes.grid in rcParams.")
                     check(plt.rcParams["lines.linewidth"] == 2.5, "Set lines.linewidth to 2.5 in rcParams.")
                 """),
                 check("figure", """
                     ax = figures()[0].axes[0]
                     check(len(ax.lines) == 1 and ax.lines[0].get_linewidth() == 2.5, "Plot `y` after changing the settings so the line picks them up.")
                 """),
             ],
             "rcParams only affect objects created after the change, so set them first.",
             starter="""
             import matplotlib.pyplot as plt
             fig, ax = plt.subplots()
             ax.plot(y)
             """,
             setup="y = [3, 1, 4, 1, 5, 9, 2, 6]", level=1),
    ]),

# ---------------------------------------------------------------- 4.12
concept(
    "py.mpl.three_d", "Three-dimensional plotting", ["py.mpl.contour"],
    "4.12", "Three-Dimensional Plotting in Matplotlib", "04.12-three-dimensional-plotting",
    """
    A 3D axes comes from the `projection="3d"` keyword (the `mplot3d`
    toolkit ships with Matplotlib and registers itself):

    ```python
    fig = plt.figure()
    ax = fig.add_subplot(projection="3d")
    ax.plot3D(x, y, z)                    # a path through space
    ax.scatter3D(x, y, z, c=z)            # points, colored by height
    K, L = np.meshgrid(k, l)
    ax.plot_surface(K, L, K**0.4 * L**0.6, cmap="viridis")
    ax.set_zlabel("Output")
    ax.view_init(elev=30, azim=-60)       # camera angle
    ```

    Surfaces need the same 2D grids as contour plots. 3D plots look
    impressive but are hard to read precisely, because depth hides values;
    a contour plot of the same function is often clearer on paper.
    """,
    [
        mc("m1", "How do you create a 3D axes on an existing figure `fig`?",
           "`fig.add_subplot(projection=\"3d\")`", ["`fig.add_subplot(dims=3)`", "`plt.axes3d()`", "`fig.add_axes(z=True)`"],
           "The `projection` keyword selects the 3D axes class.", level=1),
        mc("m2", "What inputs does `ax.plot_surface(X, Y, Z)` expect?",
           "2D arrays of the same shape, as from `np.meshgrid`",
           ["Three 1D arrays of equal length, one point each", "A single 3D array holding x, y and z", "A DataFrame with x, y and z columns"],
           "A surface is defined over a grid, so every argument is a 2D array.", level=2),
        mc("m3", "What does `ax.view_init(elev=90, azim=0)` show?",
           "The plot seen from straight above", ["The plot seen from the side at ground level", "A zoomed view", "The plot rotated 90 degrees about the z axis"],
           "Elevation is the camera angle above the x-y plane; 90 degrees looks straight down.", level=2),
        out("o1", "What does this print?",
            """
            import numpy as np
            import matplotlib.pyplot as plt
            fig = plt.figure()
            ax = fig.add_subplot(projection="3d")
            ax.plot3D([0, 1, 2], [0, 1, 0], [0, 2, 4])
            ax.scatter3D([1], [1], [1])
            print(ax.name, len(ax.lines), len(ax.collections))
            """,
            "A 3D axes reports the name `3d`. The path is a line and the scatter a collection.", level=2),
        task("c1",
             """
             On a new 3D axes, draw a surface of `Z = X**0.5 * Y**0.5` over
             the grid made by `np.meshgrid(cap, lab)` (both 1D arrays are
             defined), with the `viridis` colormap. Label the z axis "Output".
             """,
             """
             import numpy as np
             import matplotlib.pyplot as plt
             X, Y = np.meshgrid(cap, lab)
             Z = X**0.5 * Y**0.5
             fig = plt.figure()
             ax = fig.add_subplot(projection="3d")
             ax.plot_surface(X, Y, Z, cmap="viridis")
             ax.set_zlabel("Output")
             """,
             [
                 check("3d axes", """
                     fig = figures()[0]
                     check(fig.axes and fig.axes[0].name == "3d", "Create the axes with projection=\\"3d\\".")
                 """),
                 check("surface", """
                     from mpl_toolkits.mplot3d.art3d import Poly3DCollection
                     ax = figures()[0].axes[0]
                     surf = [c for c in ax.collections if isinstance(c, Poly3DCollection)]
                     check(surf, "Draw the surface with plot_surface.")
                     check(surf[0].get_cmap().name == "viridis", "Use the viridis colormap.")
                     check(ax.get_zlabel() == "Output", "Label the z axis Output.")
                 """),
             ],
             "Build the grid, compute Z on it elementwise, then call `plot_surface` on a 3D axes.",
             starter="""
             import numpy as np
             import matplotlib.pyplot as plt
             fig = plt.figure()
             ax = fig.add_subplot(projection="3d")
             """,
             setup="""
             import numpy as np
             cap = np.linspace(1, 10, 25)
             lab = np.linspace(1, 10, 25)
             """, level=2),
        mc("m4", "You have 200 scattered measurements as three 1D arrays `x`, `y`, `z`, not on a grid. Which call draws a surface through them?",
           "`ax.plot_trisurf(x, y, z)`", ["`ax.plot_surface(x, y, z)`", "`ax.plot_wireframe(x, y, z)`", "`ax.contour3D(x, y, z)`"],
           """
           `plot_trisurf` joins neighbouring points into triangles, so it does
           not need a regular grid. `plot_surface`, `plot_wireframe` and
           `contour3D` all expect 2D gridded arrays.
           """, level=2),
        mc("m5", "What does `ax.view_init(60, 35)` set on a 3D axes?",
           "The camera: elevation 60 degrees above the x-y plane, azimuth 35 degrees about the z axis",
           ["The axis limits: x runs from 0 to 60 and y runs from 0 to 35 on the 3D axes",
            "The zoom and tilt: a zoom of 60 percent and a tilt of 35 degrees toward the viewer",
            "The figure size: 60 by 35 hundredths of an inch, as `figsize` would set it"],
           """
           The first argument is the elevation and the second the azimuth, so
           `view_init(elev=90, azim=...)` looks straight down at the x-y plane.
           """, level=2),
        mc("m6", "What does `ax.plot_surface(X, Y, Z, cmap='viridis')` add compared with `ax.plot_wireframe(X, Y, Z)` on the same grid?",
           "Each grid cell is filled as a face colored by the colormap",
           ["It joins scattered points into triangles instead of a grid", "It projects contour lines onto the floor of the axes",
            "It draws only the grid lines, but thinner and in gray"],
           """
           A wireframe draws the grid lines of the surface; a surface plot fills
           each cell, and a colormap makes height easier to see.
           """, level=1),
        out("o2", "What does this print?",
            """
            import matplotlib.pyplot as plt
            fig = plt.figure()
            ax = fig.add_subplot(projection="3d")
            ax.view_init(elev=20, azim=45)
            print(ax.name, ax.elev, ax.azim)
            """,
            "A 3D axes is named `3d` and stores the camera angles set by `view_init`.", level=1),
        task("c2",
             """
             Draw a wireframe of the utility function `U = X**0.5 * Y**0.5`
             over the grid made by `np.meshgrid(good_x, good_y)` on a new 3D
             axes, label the z axis "Utility", and set the camera to an
             elevation of 30 and an azimuth of -60.
             """,
             """
             import numpy as np
             import matplotlib.pyplot as plt
             X, Y = np.meshgrid(good_x, good_y)
             U = X**0.5 * Y**0.5
             fig = plt.figure()
             ax = fig.add_subplot(projection="3d")
             ax.plot_wireframe(X, Y, U)
             ax.set_zlabel("Utility")
             ax.view_init(elev=30, azim=-60)
             """,
             [
                 check("3d axes", """
                     fig = figures()[0]
                     check(fig.axes and fig.axes[0].name == "3d", "Create the axes with projection=\\"3d\\".")
                 """),
                 check("wireframe", """
                     from mpl_toolkits.mplot3d.art3d import Line3DCollection
                     ax = figures()[0].axes[0]
                     check([c for c in ax.collections if isinstance(c, Line3DCollection)], "Draw the grid with plot_wireframe.")
                     check(ax.get_zlabel() == "Utility", "Label the z axis Utility.")
                 """),
                 check("view", """
                     ax = figures()[0].axes[0]
                     check(ax.elev == 30 and ax.azim == -60, "Set the camera with view_init(elev=30, azim=-60).")
                 """),
             ],
             "Build the grid with `np.meshgrid`, compute `U` elementwise, then `plot_wireframe` and `view_init` on a 3D axes.",
             starter="""
             import numpy as np
             import matplotlib.pyplot as plt
             fig = plt.figure()
             ax = fig.add_subplot(projection="3d")
             """,
             setup="""
             import numpy as np
             good_x = np.linspace(1, 9, 9)
             good_y = np.linspace(1, 9, 9)
             """, level=2),
    ]),

# ---------------------------------------------------------------- 4.14
concept(
    "py.mpl.distributions", "Distribution and pairwise views from pandas", ["py.mpl.histograms", "py.mpl.subplots", "py.pd.dataframe"],
    "4.14", "Visualization with Seaborn", "04.14-visualization-with-seaborn",
    """
    Seaborn is a higher-level library built on Matplotlib that draws
    statistical views (distributions, pairwise grids, faceted plots) in one
    call. It is not installed here, but pandas covers the core ideas with
    Matplotlib underneath, and each call returns ordinary axes you can style.

    ```python
    df["wage"].plot.hist(bins=30, alpha=0.5)        # a histogram on an axes
    df["wage"].plot.kde()                           # smoothed density (needs SciPy)
    df.plot.scatter(x="educ", y="wage")
    df.boxplot(column="wage", by="sector")          # one box per group
    pd.plotting.scatter_matrix(df[["educ", "exper", "wage"]])
    ```

    `scatter_matrix` is the pandas version of a pair plot: a grid with a
    scatter for every pair of columns and a histogram of each column on the
    diagonal. It returns a 2D NumPy array of axes.
    """,
    [
        mc("m1", "`pd.plotting.scatter_matrix(df[[\"a\", \"b\", \"c\"]])` returns an array of what shape?",
           "`(3, 3)`, one axes per pair of columns", ["`(3,)`", "`(1, 3)`", "`(9, 2)`"],
           "Every column is paired with every column, giving a 3 x 3 grid of axes.", level=1),
        mc("m2", "What goes on the diagonal of a pandas scatter matrix by default?",
           "A histogram of each column", ["A scatter of each column against itself", "The correlation coefficient", "Nothing"],
           "A column against itself would be a straight line, so the diagonal shows its distribution instead.", level=2),
        mc("m3", "You want to compare the distribution of hourly pay across five sectors at a glance. Which view fits best?",
           "Side-by-side box plots, one per sector", ["One scatter plot of pay against row number", "A pie chart of total pay", "A 3D surface"],
           "Box plots show median, spread and outliers for each group in a compact, comparable way.", level=1),
        mc("m4", "What does `df[\"wage\"].plot.hist(bins=20)` return?",
           "The Matplotlib axes it drew on", ["The bin counts", "A new DataFrame", "A seaborn object"],
           "pandas plotting methods return the axes, so you can keep customizing with Matplotlib calls.", level=2),
        out("o1", "What does this print?",
            """
            import pandas as pd
            import matplotlib.pyplot as plt
            df = pd.DataFrame({"educ": [10, 12, 12, 16, 18, 14],
                               "exper": [20, 5, 12, 8, 3, 10],
                               "wage": [15, 17, 21, 30, 34, 22]})
            axs = pd.plotting.scatter_matrix(df)
            print(axs.shape, len(plt.gcf().axes))
            """,
            "Three columns give a 3 x 3 grid; all nine axes live in the current figure.", level=2),
        out("o2", "What does this print?",
            """
            import pandas as pd
            df = pd.DataFrame({"wage": [12, 14, 15, 15, 18, 22, 25, 31]})
            ax = df["wage"].plot.hist(bins=4)
            print(len(ax.patches), [int(p.get_height()) for p in ax.patches])
            """,
            "Four equal-width bins from 12 to 31 (edges about 12, 16.75, 21.5, 26.25, 31) hold 4, 1, 2 and 1 values.", level=2),
    ]),

    ],
}
