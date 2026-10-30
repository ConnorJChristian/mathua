#!/usr/bin/env python3
"""Give imported Anki cards the grader their prompt asks for.

    python3 scripts/anki_grading.py <module> [--check]

FORK.md, "Which grader a question gets": a card that asks for an argument
("prove", "show that", "explain why", "justify", "derive") is a recall card,
which the language-model judge grades when it has a key
(internal/pytrack/written.go) and the learner marks otherwise; it is never a
true/false choice or a typed value that checks only the verdict or the last
line. scripts/import_anki.py follows that when it converts cards (a card
whose front asks for an argument keeps its flip card), and this applies the
same rule to a bank imported before it, using the same test as
scripts/audit_grading_style.py.

The author's decisions per card go in the module's grading_overrides.json,
which the import and this script both apply:

    {"n1784414355713c0": {"why": "the task is the equation",
                          "replace": [["Derive the equation", "Find the equation"]]},
     "n1791497845888c0": {"why": "an argument; the key 0 checks nothing", "kind": "recall"},
     "n1791424910971c0": {"why": "one option among four", "kind": "mc",
                          "prompt": "Which of these slopes is the steepest positive slope?",
                          "correct": "\\\\(7.5\\\\)", "wrong": ["\\\\(-2.5\\\\)", "\\\\(1.25\\\\)", "\\\\(-6.3\\\\)"]},
     "n1790688760948c0": {"why": "one number", "kind": "math",
                          "parts": [{"label": "", "answer": "2", "grading": "numeric", "tex": "2"}]}}

"replace" rewrites the prompt (each old text must be in it) and comes
first; then the rule; then "kind" (recall, mc or math) with its fields.
--check reports what would change and writes nothing. Re-run
scripts/anki_levels.py afterwards: a card's kind is one of its signals.

MATHUA_ROOT names the install whose modules-local/ holds the module.
"""

import glob
import json
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import audit_grading_style as audit  # noqa: E402

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
LOCAL_ROOT = os.path.abspath(os.environ.get("MATHUA_ROOT") or ROOT)
OVERRIDES_FILE = "grading_overrides.json"

# The audit's rules for a card that asks for an argument and is graded as a
# value or a choice.
WRITTEN_RULES = {"written task graded as a choice", "written task graded as a typed value", "a name graded as math"}


def written_rules(item):
    """The audit rules in WRITTEN_RULES that flag item."""
    return [r for r, _ in audit.audit_item("anki", "", "", item) if r in WRITTEN_RULES]


def asks_written(front, kind, verdict=None):
    """Whether a card front would be flagged as a written task if it became
    a choice (kind "mc", with its verdict) or a typed value (kind "math")."""
    if kind == "mc":
        opposite = {"True": "False", "False": "True", "Yes": "No", "No": "Yes"}.get(verdict, "No")
        item = {"kind": "mc", "prompt": front, "correct": verdict or "Yes", "wrong": [opposite]}
    else:
        item = {"kind": "math", "prompt": front, "parts": [{"label": "", "answer": "x", "grading": "expression"}]}
    return bool(written_rules(item))


def as_recall(it):
    """The flip card of a converted card: the prompt, and the full back."""
    if it["kind"] == "mc":
        back = it.get("explain") or it.get("back") or ""
        if not back:
            back = it.get("correct", "")
    elif it["kind"] == "math":
        shown = []
        for p in it.get("parts", []):
            tex = p.get("tex") or p.get("answer", "")
            shown.append("\\(%s = %s\\)" % (p["label"], tex) if p.get("label") else "\\(%s\\)" % tex)
        back = ", ".join(shown)
        if it.get("back"):
            back += "<br><br>" + it["back"]
    else:
        return it
    out = {"id": it["id"], "kind": "recall", "level": it.get("level", 2), "prompt": it["prompt"],
           "back": back, "explain": ""}
    for k in ("grading", "book_ref", "unlocks_after"):
        if k in it:
            out[k] = it[k]
    return out


def replace_prompt(it, pairs, where):
    p = it["prompt"]
    for old, new in pairs:
        if old not in p:
            if new in p:
                continue  # already applied
            sys.exit("%s: %r is not in the prompt of %s" % (where, old, it["id"]))
        p = p.replace(old, new)
    it["prompt"] = p


def with_kind(it, ov, where):
    kind = ov["kind"]
    if it["kind"] == kind:
        base = {"prompt": it["prompt"], "back": it.get("back") or it.get("explain", "")}
    else:
        base = as_recall(it) if it["kind"] != "recall" else dict(it)
    if kind == "recall":
        out = dict(it) if it["kind"] == "recall" else base
        if "grading" in ov:
            out["grading"] = ov["grading"]
        return out
    out = {"id": it["id"], "kind": kind, "level": it.get("level", 2), "prompt": ov.get("prompt", base["prompt"])}
    if kind == "mc":
        for k in ("correct", "wrong"):
            if k not in ov:
                sys.exit("%s: %s needs %r for a choice" % (where, it["id"], k))
        out["correct"], out["wrong"] = ov["correct"], ov["wrong"]
        out["explain"] = ov.get("explain", base.get("back", ""))
    elif kind == "math":
        if not ov.get("parts"):
            sys.exit("%s: %s needs parts for a typed answer" % (where, it["id"]))
        out["parts"] = ov["parts"]
        out["back"] = ov.get("back", base.get("back", ""))
        out["explain"] = ""
    else:
        sys.exit("%s: %s has kind %r; want recall, mc or math" % (where, it["id"], kind))
    return out


def load_overrides(module_dir):
    p = os.path.join(module_dir, OVERRIDES_FILE)
    if not os.path.exists(p):
        return {}
    with open(p, encoding="utf-8") as f:
        return json.load(f)


def apply(concepts, module_dir, overrides=None):
    """Apply the overrides and the rule to the bank concepts in place.
    Returns [(card id, what changed)]."""
    if overrides is None:
        overrides = load_overrides(module_dir)
    where = os.path.join(module_dir, OVERRIDES_FILE)
    changes = []
    seen = set()
    for c in concepts:
        items = c["items"]
        for i, it in enumerate(items):
            ov = overrides.get(it["id"], {})
            seen.add(it["id"])
            before = json.dumps(it, sort_keys=True)
            what = []
            if ov.get("replace"):
                replace_prompt(it, ov["replace"], where)
            if not ov.get("kind") and it["kind"] in ("mc", "math") and written_rules(it):
                it = as_recall(it)
                what.append("rule: written task -> recall")
            if ov.get("kind"):
                it = with_kind(it, ov, where)
            items[i] = it
            if json.dumps(it, sort_keys=True) != before:
                if ov:
                    what.append("override: " + (ov.get("kind") or "prompt"))
                changes.append((it["id"], ", ".join(what) or "unchanged kind"))
    missing = sorted(set(overrides) - seen)
    return changes, missing


def main(argv):
    args = [a for a in argv[1:] if not a.startswith("--")]
    check = "--check" in argv
    if len(args) != 1:
        print(__doc__)
        return 2
    mdir = os.path.join(LOCAL_ROOT, "modules-local", args[0])
    files = [f for f in sorted(glob.glob(os.path.join(mdir, "bank", "*.json"))) if os.path.basename(f) != "skip.json"]
    data = {}
    for f in files:
        with open(f, encoding="utf-8") as fh:
            data[f] = json.load(fh)
    before = {f: json.dumps(d, sort_keys=True) for f, d in data.items()}
    concepts = [c for d in data.values() for c in d["concepts"]]
    changes, missing = apply(concepts, mdir)
    for cid, what in changes:
        print("  %s: %s" % (cid, what))
    print("%d cards changed%s" % (len(changes), " (--check: nothing written)" if check else ""))
    if missing:
        print("overrides for cards not in the bank: %s" % ", ".join(missing))
    if not check:
        for f, d in data.items():
            if json.dumps(d, sort_keys=True) != before[f]:
                with open(f, "w", encoding="utf-8") as out:
                    json.dump(d, out, indent=1, ensure_ascii=False)
                    out.write("\n")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
