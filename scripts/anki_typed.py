"""Turn Anki cards into Mathua-style questions where the card allows it.

Used by scripts/import_anki.py. Three shapes are recognised:

  - typed answers: the answer at the top of the back is one formula or number,
    or a few labelled ones ("x_0 = 60, y_0 = 40"). The learner types each one
    into Mathua's math editor and Mathua's own graders check it for
    equivalence, exactly as for Mathua's own questions;
  - true/false and yes/no: the back opens with the verdict, which becomes a
    two-option multiple choice question;
  - everything else stays a flip card.

A typed candidate is only a candidate. The importer checks every one with the
real graders (scripts/check_typed_answers) and keeps it only when

  - the book's answer, as Mathua's math editor (MathLive) turns it into text,
    is graded correct against an independent reading of the same LaTeX
    (SymPy's LaTeX parser), and
  - a wrong answer (the same value plus one) is graded wrong.

So a card never becomes a typed question the learner cannot answer, or one
that accepts anything.

Needs SymPy with its LaTeX parser: pip install sympy "antlr4-python3-runtime==4.11.*"
"""

import html
import re

SPAN_RE = re.compile(r"\\\((.*?)\\\)|\\\[(.*?)\\\]", re.S)

# A value built from these is not one expression: a choice of sign, a set, an
# inequality, words, a matrix or an operation left to do.
REJECT_TOKENS = (
    r"\pm", r"\mp", r"\text", r"\mathrm", r"\operatorname", r"\Rightarrow", r"\implies",
    r"\iff", r"\to", r"\rightarrow", r"\leq", r"\geq", r"\le", r"\ge", r"\lt", r"\gt", r"\neq",
    r"\in", r"\begin", r"\cup", r"\cap", r"\infty", r"\int", r"\sum", r"\prod",
    r"\lim", r"\partial", r"\mid", r"\forall", r"\exists", r"\%", r"\$", r"\vec", r"\mathbf",
    r"\bar", r"\hat", r"\tilde", r"\dot", r"\binom", r"\choose", r"\det", r"\max", r"\min",
    r"\cdots", r"\ldots", r"\dots", r"\vdots", r"\sim", r"\equiv", r"\propto", r"\perp",
    r"\angle", r"\circ", r"\overline", r"\underline", r"\boxed", r"\tag", r"\label", r"\\",
    r"\underset", r"\overset", r"\mathcal", r"\langle", r"\rangle", r"\{", r"\}",
)
REJECT_CHARS = set(";<>!&%$[]")

NAME_RE = re.compile(r"^[A-Za-z](?:_[A-Za-z0-9]{1,3})?$")
GREEK = {"alpha", "beta", "gamma", "delta", "theta", "lambda", "mu", "sigma", "rho", "tau", "phi",
         "omega", "epsilon", "eta", "kappa", "nu", "xi", "psi", "chi", "zeta"}

MAX_PARTS = 4


def strip_html(s):
    s = re.sub(r"<br\s*/?>", "\n", s, flags=re.I)
    s = re.sub(r"</(?:div|p)>", "\n", s, flags=re.I)
    s = re.sub(r"<[^>]+>", "", s)
    return html.unescape(s).replace("\xa0", " ")


def answer_head(back):
    """The answer at the top of a back, before the worked solution: everything
    up to the first blank line."""
    parts = re.split(r"(?:<br\s*/?>\s*){2,}|\n\s*\n", back, maxsplit=1, flags=re.I)
    head = strip_html(parts[0]).strip()
    head = re.sub(r"^(?:Answer|Ans\.?)\s*:\s*", "", head, flags=re.I)
    return head


def clean_tex(tex):
    tex = tex.replace("\n", " ").strip()
    tex = tex.replace(r"\displaystyle", "")
    tex = re.sub(r"\\[,;:! ]", " ", tex)
    tex = tex.replace(r"\left", "").replace(r"\right", "")
    tex = tex.replace(r"\dfrac", r"\frac").replace(r"\tfrac", r"\frac")
    # Thousands separators: 34{,}650 is 34650.
    tex = re.sub(r"(?<=\d)\{,\}(?=\d{3})", "", tex)
    tex = re.sub(r"\s+", " ", tex).strip()
    tex = re.sub(r"[.,]\s*$", "", tex).strip()
    return tex


def strip_constant(tex):
    """'+ C', '+ c', '+ C_1': the constant of integration, which Mathua's
    grader ignores on both sides."""
    return re.sub(r"\s*\+\s*[Cc](?:_\{?\d\}?)?\s*$", "", tex)


def top_level_split(tex, seps):
    """Split on any of seps outside braces and parentheses."""
    out, depth, cur, i = [], 0, "", 0
    while i < len(tex):
        ch = tex[i]
        if ch in "{([":
            depth += 1
        elif ch in "})]":
            depth -= 1
        if depth == 0:
            hit = next((s for s in seps if tex.startswith(s, i)), None)
            if hit:
                out.append(cur)
                cur = ""
                i += len(hit)
                continue
        cur += ch
        i += 1
    out.append(cur)
    return [p.strip() for p in out]


def split_relation(tex):
    """'\\pi = -x^2 + 30x' -> ('\\pi', '-x^2 + 30x'). The value is the last
    exact part of a chain; an approximation after it is dropped, and a name on
    the left is kept as the label shown before the input."""
    tex = top_level_split(tex, [r"\approx"])[0]
    segs = top_level_split(tex, ["="])
    if segs and segs[-1] == "":  # "x^{a+b}=": the card ends on '='
        segs = segs[:-1]
    if not segs:
        return "", ""
    if len(segs) == 1:
        return "", segs[0]
    return segs[0], segs[-1]


def is_label(tex):
    """A left-hand side worth showing: a name, possibly decorated (y, \\pi,
    z_{xx}, f'(x), y_c, TR, \\bar Y, \\Delta \\bar Y, |A|, MP_K). Not an
    expression to simplify, which would give the answer away."""
    t = tex.replace(" ", "")
    if not t or len(t) > 32:
        return False
    t = re.sub(r"\\(?:bar|hat|tilde|mathrm|text|operatorname|mathbf)\{([^{}]*)\}", r"\1", t)
    t = re.sub(r"\\(?:bar|hat|tilde)", "", t)
    t = re.sub(r"^\|(.*)\|$", r"\1", t)
    t = re.sub(r"^\\Delta", "", t)
    return re.fullmatch(
        r"(?:\\[A-Za-z]+|[A-Za-z]{1,3})(?:'{1,2})?(?:\^\{?[*A-Za-z0-9()]{1,4}\}?)?"
        r"(?:_\{?[A-Za-z0-9,]{1,4}\}?)?(?:\^\{?[*A-Za-z0-9()]{1,4}\}?)?(?:\([A-Za-z0-9_,{}]{1,10}\))?", t) is not None


def value_ok(tex):
    if not tex:
        return False
    if any(t in tex for t in REJECT_TOKENS) or set(tex) & REJECT_CHARS:
        return False
    if re.search(r"(?<![\\{])[A-Za-z]\s*,\s*[A-Za-z0-9]", tex) or re.search(r"\d\s*,\s*-?\d", tex):
        return False
    if tex.count("|") % 2:
        return False
    if len(top_level_split(tex, [","])) > 1:  # a list, not one value
        return False
    return True


def math_parts(head):
    """[(label LaTeX, value LaTeX)] when the head is only formulas, else
    (None, reason)."""
    spans = list(SPAN_RE.finditer(head))
    if not spans:
        return None, "no formula"
    # Only separators between the formulas: the answer is not prose.
    rest = SPAN_RE.sub("\x00", head)
    between = rest.replace("\x00", "")
    if re.sub(r"(?i)\b(?:and)\b|[\s,;.]", "", between):
        return None, "words around the formula"
    pieces = []
    for m in spans:
        tex = clean_tex(m.group(1) or m.group(2) or "")
        for p in top_level_split(tex, [r"\qquad", r"\quad", ";", r"\text{ and }", r"\text{and}"]):
            p = p.strip().rstrip(",").strip()
            if p:
                # "x = 1, y = 2" inside one formula: a comma between two
                # labelled relations.
                sub = top_level_split(p, [","])
                if len(sub) > 1 and all("=" in s for s in sub):
                    pieces.extend(sub)
                else:
                    pieces.append(p)
    if not pieces or len(pieces) > MAX_PARTS:
        return None, "too many parts" if pieces else "empty formula"
    parts = []
    for p in pieces:
        label, value = split_relation(p)
        value = strip_constant(value)
        if not value_ok(value):
            return None, "a part is not one expression"
        if label and not is_label(label):
            label = ""
        parts.append((label, value))
    if len(parts) > 1 and not all(l for l, _ in parts):
        return None, "parts without labels"
    if len({l for l, _ in parts}) != len(parts):
        return None, "repeated labels"
    return parts, ""


VERDICT_RE = re.compile(r"^(True|False|Yes|No)\b")


def verdict(head):
    """'True'/'False'/'Yes'/'No' when the back opens with a verdict."""
    m = VERDICT_RE.match(head.strip())
    return m.group(1) if m else ""


# ---------------------------------------------------------------- SymPy side

_parse_latex = None


def _parser():
    global _parse_latex
    if _parse_latex is None:
        from sympy.parsing.latex import parse_latex
        _parse_latex = parse_latex
    return _parse_latex


MIXED_RE = re.compile(r"(?<![\w.^}])(\d+)\s*\\frac\{(\d+)\}\{(\d+)\}")


def sympy_of(tex):
    """The expression SymPy reads from the LaTeX, normalised so its str() is
    something Mathua's grader parses: e is Euler's number, subscripted names
    are written x_0, natural logs are log(x)."""
    import sympy
    from sympy.core.function import AppliedUndef

    # A mixed number, 7\frac{2}{3}, is 7 + 2/3; SymPy's parser reads it as a
    # product.
    tex = MIXED_RE.sub(r"(\1+\\frac{\2}{\3})", tex)
    expr = _parser()(tex)
    if isinstance(expr, (sympy.logic.boolalg.Boolean, sympy.core.relational.Relational)):
        raise ValueError("relation")
    if not isinstance(expr, sympy.Expr):
        raise ValueError("not an expression")
    if expr.atoms(sympy.Integral, sympy.Derivative, sympy.Limit, sympy.Sum, sympy.Product):
        raise ValueError("unevaluated operation")
    if expr.atoms(AppliedUndef):
        raise ValueError("unknown function")
    subs = {}
    for s in expr.free_symbols:
        name = s.name
        if name == "e":
            subs[s] = sympy.E
            continue
        plain = re.sub(r"_\{(\w+)\}", r"_\1", name)
        if not NAME_RE.match(plain) and plain not in GREEK:
            raise ValueError("unusual name")
        if plain != name:
            subs[s] = sympy.Symbol(plain)
    if subs:
        expr = expr.xreplace(subs)
    expr = expr.replace(lambda e: isinstance(e, sympy.log) and len(e.args) == 2 and e.args[1] == sympy.E,
                        lambda e: sympy.log(e.args[0]))
    if expr.has(sympy.zoo, sympy.nan, sympy.oo, -sympy.oo):
        raise ValueError("not finite")
    return expr


def equivalent(a, b):
    """Whether two expressions are equal for positive values of their
    variables. Positive, because "simplify x^2 x^{1/2}" has the answer
    sqrt(x^5), which differs from x^{5/2} only for negative x, and copying
    the question would still be graded correct."""
    import random
    import sympy
    syms = a.free_symbols | b.free_symbols
    pos = {s: sympy.Dummy(s.name, positive=True) for s in syms}
    a, b = a.xreplace(pos), b.xreplace(pos)
    try:
        d = sympy.simplify(a - b)
        if d == 0:
            return True
        if not d.free_symbols:
            return abs(complex(d)) < 1e-9
    except Exception:
        return False
    rnd = random.Random(7)
    try:
        for _ in range(4):
            vals = {s: rnd.uniform(0.3, 3) for s in pos.values()}
            va, vb = complex(a.subs(vals)), complex(b.subs(vals))
            if abs(va - vb) > 1e-8 * max(1.0, abs(va), abs(vb)):
                return False
        return True
    except Exception:
        return False


def gives_answer_away(front, ref):
    """True when an expression on the front already equals the answer, so
    copying it would be graded correct ("simplify x^10/x^6", "factor ...")."""
    for m in SPAN_RE.finditer(front):
        tex = clean_tex(m.group(1) or m.group(2) or "")
        for part in top_level_split(tex, ["=", ",", r"\quad", r"\qquad", ";"]):
            part = part.strip()
            if not part:
                continue
            try:
                e = sympy_of(part)
            except Exception:
                continue
            if equivalent(e, ref):
                return True
    return False


NUMBER_TEX_RE = re.compile(r"^-?\s*(?:\d+(?:\.\d+)?|\.\d+|\d*\s*\\frac\{\d+\}\{\d+\}|\\frac\{-?\d+\}\{\d+\}|\d+/\d+|\\frac\d\d)$")


def grading_for(tex):
    """'numeric' when the book wrote a plain number (an integer, a decimal, a
    fraction or a mixed number), which the numeric grader reads; 'expression'
    for everything else, including numbers written as expressions such as
    e^{1.8} or 81^{1/4}, which SymPy grades."""
    return "numeric" if NUMBER_TEX_RE.match(tex.strip()) else "expression"


def answer_rest(back):
    """The back after its answer line: the worked solution, if any."""
    parts = re.split(r"(?:<br\s*/?>\s*){2,}|\n\s*\n", back, maxsplit=1, flags=re.I)
    return parts[1].strip() if len(parts) > 1 else ""


def typed_candidate(front_html, head):
    """(parts, reason) for an answer head (the text of the answer, with its
    formulas in \\( \\) or \\[ \\]). Each part is a dict with label, tex,
    grading, ref and wrong; the importer fills in the answer from MathLive
    and checks it."""
    import sympy
    parts, why = math_parts(head)
    if not parts:
        return None, why
    front = strip_html(front_html)
    out = []
    for label, tex in parts:
        try:
            ref = sympy_of(tex)
        except Exception as ex:
            return None, "SymPy: " + (str(ex).split("\n")[0][:40] or type(ex).__name__)
        if gives_answer_away(front, ref):
            return None, "the question already shows an equivalent expression"
        g = grading_for(tex)
        if g == "numeric" and not ref.has(sympy.Float):
            ref_s = sympy.sstr(sympy.nsimplify(ref))
        else:
            ref_s = sympy.sstr(ref)
        out.append({"label": label, "tex": tex, "grading": g, "ref": ref_s, "wrong": sympy.sstr(ref + 1)})
    return out, ""
