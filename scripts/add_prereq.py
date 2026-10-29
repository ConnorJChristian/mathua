#!/usr/bin/env python3
"""Add prerequisites to a topic authored with scripts/python_bank.py.

    python3 scripts/add_prereq.py <module> <topic id> <prerequisite id>...
    python3 scripts/add_prereq.py --dry-run ma-program ma.hoy.2_order calc.limit.supremum

Links live in the authoring source: the prerequisite list is the third
argument of the `topic(...)` or `concept(...)` call in `modules/<id>/src/` or
`modules-local/<id>/src/` (MATHUA_ROOT names the install whose
modules-local/ to edit, as for python_bank.py). This edits that list in
place and leaves everything else as written, so the file stays the source
of truth. Rebuild the bank afterwards:

    python3 scripts/python_bank.py build <module>

The script refuses an id already in the list and a prerequisite that is not
in the graph (data/concepts plus every module's concepts/).
"""

import ast
import glob
import json
import os
import re
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
LOCAL_ROOT = os.path.abspath(os.environ.get("MATHUA_ROOT") or ROOT)


def module_dir(module):
    for area in (os.path.join(ROOT, "modules"), os.path.join(LOCAL_ROOT, "modules-local")):
        d = os.path.join(area, module)
        if os.path.isfile(os.path.join(d, "module.json")):
            return d
    sys.exit("no module %r under modules/ or modules-local/" % module)


def known_ids():
    ids = set()
    paths = glob.glob(os.path.join(ROOT, "data", "concepts", "*.json"))
    for area in (os.path.join(ROOT, "modules"), os.path.join(LOCAL_ROOT, "modules-local")):
        paths += glob.glob(os.path.join(area, "*", "concepts", "*.json"))
    for p in paths:
        if os.path.basename(p) == "enrichment.json":
            continue
        for c in json.load(open(p, encoding="utf-8")):
            ids.add(c["id"])
    return ids


def rel(path):
    for root in (ROOT, LOCAL_ROOT):
        r = os.path.relpath(path, root)
        if not r.startswith(".."):
            return r
    return path


def find_call(text, topic):
    """The span of the prerequisite list of topic's authoring call.

    The call starts with `topic(` or `concept(`, its first argument is the
    id and the third the list. Returns (list_start, list_end) or None.
    """
    for m in re.finditer(r'\b(?:topic|concept)\(\s*(?:#[^\n]*\n\s*)?(["\'])' + re.escape(topic) + r'\1\s*,', text):
        i = m.end()
        # Skip the label (a string literal, possibly with comments or
        # newlines around it), then the list.
        j = skip_ws(text, i)
        q = text[j]
        if q not in "\"'":
            continue
        j = string_end(text, j)
        j = skip_ws(text, j)
        if text[j] != ",":
            continue
        j = skip_ws(text, j + 1)
        if text[j] != "[":
            continue
        k = j
        depth = 0
        while k < len(text):
            if text[k] == "[":
                depth += 1
            elif text[k] == "]":
                depth -= 1
                if depth == 0:
                    return j, k + 1
            k += 1
    return None


def skip_ws(text, i):
    while i < len(text):
        if text[i] in " \t\r\n":
            i += 1
        elif text[i] == "#":
            while i < len(text) and text[i] != "\n":
                i += 1
        else:
            break
    return i


def string_end(text, i):
    q = text[i]
    if text.startswith(q * 3, i):
        return text.index(q * 3, i + 3) + 3
    k = i + 1
    while k < len(text):
        if text[k] == "\\":
            k += 2
            continue
        if text[k] == q:
            return k + 1
        k += 1
    raise ValueError("unterminated string")


def add(module, topic, prereqs, dry_run=False):
    mdir = module_dir(module)
    for src in sorted(glob.glob(os.path.join(mdir, "src", "[0-9][0-9]_*.py"))):
        text = open(src, encoding="utf-8").read()
        span = find_call(text, topic)
        if span is None:
            continue
        a, b = span
        have = ast.literal_eval(text[a:b])
        new = [p for p in prereqs if p not in have]
        if not new:
            print("%s: %s already lists %s" % (rel(src), topic, ", ".join(prereqs)))
            return True
        quote = "'" if text[a:b].count("'") > text[a:b].count('"') else '"'
        items = [quote + p + quote for p in have + new]
        text = text[:a] + "[" + ", ".join(items) + "]" + text[b:]
        print("%s: %s += %s" % (rel(src), topic, ", ".join(new)))
        if not dry_run:
            open(src, "w", encoding="utf-8").write(text)
        return True
    sys.exit("no topic(...) or concept(...) call for %r in %s/src" % (topic, mdir))


def main(argv):
    dry = "--dry-run" in argv
    args = [a for a in argv[1:] if not a.startswith("--")]
    if len(args) < 3:
        print(__doc__)
        return 2
    module, topic, prereqs = args[0], args[1], args[2:]
    ids = known_ids()
    missing = [p for p in prereqs if p not in ids]
    if missing:
        sys.exit("not in the graph: %s" % ", ".join(missing))
    if topic in prereqs:
        sys.exit("a topic cannot be its own prerequisite")
    add(module, topic, prereqs, dry)
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
