#!/usr/bin/env python3
"""List question-bank items whose prompt asks for one kind of answer and whose
grading is another (fork change; the rules are in FORK.md, "Which grader a
question gets").

    python3 scripts/audit_grading_style.py                 # every module
    python3 scripts/audit_grading_style.py anki ma-program # some modules
    python3 scripts/audit_grading_style.py --json out.json

With MATHUA_ROOT set, modules-local/ is read from that install. The output
is a list, by rule, of "<module>/<bank> <concept>#<item> [kind]: <reason>",
with the first line of the prompt. Recall cards that look like a typed
question are listed under "could be typed" as suggestions, not defects:
only the author can say whether the card's back is one checkable value.
"""

import glob
import json
import os
import re
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
LOCAL_ROOT = os.path.abspath(os.environ.get("MATHUA_ROOT") or ROOT)

MATH_SPAN = re.compile(r"\\\(.*?\\\)|\\\[.*?\\\]|\$\$.*?\$\$|`[^`]*`", re.S)
TAG = re.compile(r"<[^>]+>")
SPACE = re.compile(r"\s+")

# A task that needs an argument, not a value.
WRITTEN = re.compile(
    r"\b(prove|show that|demonstrate that|derive|verify that|explain why|explain how|justify|argue|"
    r"why does|why is|why are|why do|in what sense|discuss|interpret|compare and contrast|describe)\b", re.I)
# A task that is a value after all, even when a written word appears.
VALUE = re.compile(
    r"\b(what is|find|compute|calculate|evaluate|solve|give|enter|type|state the value|how many|how much|"
    r"determine|express|write the|simplify|differentiate|integrate|the value of|equals?)\b", re.I)
CHOICE = re.compile(r"\b(which of (the following|these)|which one|select one|choose one)\b", re.I)
# "the number of ways a student can choose one of the texts" is counting.
COUNTING = re.compile(r"\b(number (M )?of ways|how many ways|in how many)\b[^.?]{0,80}\b(choose|select) one\b", re.I)
# Phrases that mention a written keyword without asking for an argument, as
# internal/pytrack/written.go removes them before it classifies a card: a
# method question ("how does one prove"), a purpose clause ("to verify that
# A is idempotent, it is enough to check..."), a subject that shows ("do the
# rates not show that", "two arrays describe the same people"), and
# "compute it, or show that it is impossible".
NOT_ASKED = re.compile(
    r"\bhow (do|would|can|could|should|might|does|did|to) (you |we |one |i )?(\w+ ){0,2}(prove|show|verify|derive|justify|demonstrate)\b"
    r"|^to (prove|show|verify|derive) (that )?[^,]{0,120},"
    r"|\b(not|never|n't) (show|prove|demonstrate)s?\b( that)?"
    r"|\bor (to )?(show|prove|explain)\b[^.?!;]*"
    r"|\b(\w+s|M|they|these|both) (describe|show) (the same|how|that)\b", re.I)
# A term the prompt asks to name ("Name the process the authors describe...").
NAME_ASK = re.compile(r"^\s*name\b|\bwhat is (it|this|the process|the term) called\b", re.I)
# A numbered part label ("1.") at the top of a back is not an answer.
PART_LABEL = re.compile(r"^\d+\.$")
YESNO = re.compile(r"\b(true or false|yes or no|\(yes/no\)|\(true/false\)|is it (true|possible|the case))\b", re.I)
# A prompt that is one closed question ("Is M invertible?"), nothing else.
CLOSED = re.compile(r"^(is|are|does|do|can|will|has|have|should|must|could) [^?]{3,160}\?$", re.I)
SETLIKE = re.compile(r"\b(the set of|all (values|x|points) (such that|for which|where)|interval|domain|range)\b", re.I)
NAME = re.compile(r"\b(what is (the name|this|the term)|is called|the term for|name (of|for|the)|which term|what do we call)\b", re.I)
NUMBERWORDS = re.compile(r"\b(how many|compute|calculate|the value of|evaluate|what is the (derivative|integral|limit|sum|product|determinant|rank|eigenvalue)|numerically)\b", re.I)


def clean(s):
    s = TAG.sub(" ", s or "")
    s = MATH_SPAN.sub(" M ", s)
    return SPACE.sub(" ", s).strip()


def first_line(s, n=110):
    s = clean(s)
    return s if len(s) <= n else s[: n - 1] + "…"


def bank_files(only):
    out = []
    for area in (os.path.join(ROOT, "modules"), os.path.join(LOCAL_ROOT, "modules-local")):
        for f in sorted(glob.glob(os.path.join(area, "*", "bank", "*.json"))):
            if os.path.basename(f) == "skip.json":
                continue
            module = os.path.basename(os.path.dirname(os.path.dirname(f)))
            if only and module not in only:
                continue
            out.append((module, f))
    return out


def audit_item(module, bank, cid, it):
    """Yield (rule, reason) for one item."""
    kind = it["kind"]
    prompt = clean(it.get("prompt", ""))
    low = prompt.lower()
    asked = NOT_ASKED.sub(" ", prompt)
    written = WRITTEN.search(asked)
    value = VALUE.search(prompt)
    if kind == "term" and NAME_ASK.search(prompt):
        # "Name the process the authors describe as...": the task is the name.
        written, value = None, NAME_ASK.search(prompt)
    if kind in ("math", "term", "output"):
        if written and not value:
            yield "written task graded as a typed value", "asks to %s" % written.group(0).lower()
        elif written and value:
            yield "ambiguous: written words and a value", "asks to %s and to %s" % (written.group(0).lower(), value.group(0).lower())
        if CHOICE.search(prompt) and kind != "mc" and not COUNTING.search(prompt):
            yield "a choice graded as typed", "asks to %s" % CHOICE.search(prompt).group(0).lower()
        if YESNO.search(prompt) or CLOSED.match(prompt):
            yield "a yes/no question graded as typed", "asks %s" % (YESNO.search(prompt) or CLOSED.match(prompt)).group(0).lower()[:60]
    if kind == "math":
        gradings = {p["grading"] for p in it["parts"]}
        if SETLIKE.search(prompt) and gradings <= {"numeric", "expression"}:
            m = SETLIKE.search(prompt).group(0).lower()
            # A domain or interval asked for as a typed expression is only a
            # mismatch when the key is a set or an interval.
            if any(re.search(r"[{}\[\]∪∞]|oo|\binf", p["answer"]) for p in it["parts"]):
                yield "a set or interval graded as an expression", "asks for %s, key %r" % (m, it["parts"][0]["answer"])
        if NAME.search(prompt) and gradings <= {"numeric", "expression"}:
            yield "a name graded as math", "asks %s" % NAME.search(prompt).group(0).lower()
    if kind == "term":
        # Outside quoted text: a cloze sentence may itself say "how many".
        if NUMBERWORDS.search(re.sub(r"\"[^\"]*\"|“[^”]*”", " ", prompt)):
            yield "a value graded as a term", "asks %s" % NUMBERWORDS.search(prompt).group(0).lower()
    if kind == "mc":
        # "Why does X?" with reasons as options is a choice; a proof is not.
        strong = re.search(r"\b(prove|show that|derive|justify|verify that|explain (why|how))\b", asked, re.I)
        if strong and not CHOICE.search(prompt) and not re.search(r"\b(best|correct|true|false|which)\b", low):
            yield "written task graded as a choice", "asks to %s" % strong.group(0).lower()
        if (re.match(r"^(true or false|yes or no)\b", prompt, re.I) or re.search(r"\((yes/no|true/false)\)\s*$", prompt, re.I)) and len(it.get("wrong", [])) >= 2:
            yield "a yes/no question with invented options", "a true/false or yes/no stem with %d options" % (1 + len(it["wrong"]))
    if kind == "recall":
        if it.get("grading") != "llm" and written and not value:
            pass  # internal/pytrack/written.go classifies these for the judge
        if CHOICE.search(prompt):
            yield "could be a choice", "a recall card that asks to %s" % CHOICE.search(prompt).group(0).lower()
        elif YESNO.search(prompt) and re.match(r"^\s*(yes|no|true|false)\b", clean(it.get("back", "")), re.I):
            yield "could be yes/no", "a recall card whose back starts with the verdict"
        elif NUMBERWORDS.search(prompt) and not written and re.match(r"^[-\d./]+$", clean(it.get("back", "")).split(" ")[0]) \
                and not PART_LABEL.match(clean(it.get("back", "")).split(" ")[0]):
            yield "could be typed", "a recall card whose back starts with a number"
    if kind == "code":
        if not it.get("tests"):
            yield "a code task without tests", "nothing checks it"
        if written and not re.search(r"\b(write|implement|define|return|print|compute|fit|plot|build|create|produce)\b", low):
            yield "written task graded as code", "asks to %s" % written.group(0).lower()


def main(argv):
    out_json = None
    args = []
    i = 0
    while i < len(argv):
        if argv[i] == "--json":
            out_json = argv[i + 1]
            i += 2
            continue
        args.append(argv[i])
        i += 1
    only = set(args)
    by_rule = {}
    n = 0
    for module, f in bank_files(only):
        bank = os.path.basename(f)[:-5]
        data = json.load(open(f, encoding="utf-8"))
        for c in data["concepts"]:
            for it in c["items"]:
                n += 1
                for rule, reason in audit_item(module, bank, c["id"], it):
                    by_rule.setdefault(rule, []).append({
                        "module": module, "bank": bank, "concept": c["id"], "item": it["id"],
                        "kind": it["kind"], "reason": reason, "prompt": first_line(it.get("prompt", "")),
                    })
    for rule in sorted(by_rule, key=lambda r: -len(by_rule[r])):
        rows = by_rule[rule]
        print("== %s (%d)" % (rule, len(rows)))
        for r in rows:
            print("  %s/%s %s#%s [%s]: %s\n      %s" % (r["module"], r["bank"], r["concept"], r["item"], r["kind"], r["reason"], r["prompt"]))
    print("\n%d items, %d flagged" % (n, sum(len(v) for v in by_rule.values())))
    if out_json:
        with open(out_json, "w", encoding="utf-8") as fh:
            json.dump(by_rule, fh, indent=1, ensure_ascii=False)
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
