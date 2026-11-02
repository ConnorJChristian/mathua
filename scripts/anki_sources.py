#!/usr/bin/env python3
"""Name the book each imported Anki card comes from.

    python3 scripts/anki_sources.py <module> [--check]

The order of practice inside a topic serves the course's own book first,
then the same topic in other books, then generated problem sets (FORK.md,
"The order of practice inside a topic"). A card's book is, in order:

  1. its `source::<Book>::...` tag (Schaums_Calc_Econ, Axler_...);
  2. a tag the module's subjects.json maps to a book, under "books":
         {"books": {"tags": {"mmbef": "schaums-math-methods"},
                    "decks": {"Python Data Science Handbook": "pdsh"}}}
  3. the deepest segment of its deck path that "books.decks" maps, or that
     names a book of the registry (internal/books/registry.json: "ISLP",
     "Wooldridge - Introductory Econometrics 7e");
  4. none: the card is unsourced and is served with the generated questions.

Names resolve to the registry's stable ids the way the server resolves them
(internal/books). The result is the module's sources.json, {card id: book
id}, which the server reads next to bank/; a module with a sources.json
takes its cards' books from it only. scripts/anki_levels.py writes it on
every import and every re-application of levels (from levels.json's decks
and tags). --check prints the counts and writes nothing.

MATHUA_ROOT names the install whose modules-local/ holds the module.
"""

import json
import os
import re
import sys
from collections import Counter

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
LOCAL_ROOT = os.path.abspath(os.environ.get("MATHUA_ROOT") or ROOT)
REGISTRY = os.path.join(ROOT, "internal", "books", "registry.json")
SOURCES_FILE = "sources.json"


def key(s):
    """internal/books.Key."""
    s = (s or "").strip()
    for p in ("web:", "notes:", "file:"):
        if s.startswith(p):
            s = s[len(p):]
    if "/" in s and " " not in s:
        s = s.rsplit("/", 1)[-1]
    s = re.sub(r"\.pdf$", "", s, flags=re.I)
    s = s.replace("'", "").replace("’", "")
    s = re.sub(r"[\W_]+", "-", s.lower())
    return s.strip("-")


class Registry:
    def __init__(self, path=REGISTRY):
        with open(path, encoding="utf-8") as f:
            reg = json.load(f)
        self.aliases = []
        self.labels = {}
        for b in reg["books"]:
            self.labels[b["id"]] = b["label"]
            self.aliases.append((b["id"], b["id"]))
            for a in b.get("aliases", []):
                self.aliases.append((key(a), b["id"]))
        self.aliases.sort(key=lambda a: -len(a[0]))

    def known(self, name):
        k = key(name)
        if not k:
            return ""
        for a, bid in self.aliases:
            if k == a or k.startswith(a + "-"):
                return bid
        return ""

    def id(self, name):
        return self.known(name) or key(name)


def load_rules(module_dir):
    p = os.path.join(module_dir, "subjects.json")
    if not os.path.exists(p):
        return {}, {}
    with open(p, encoding="utf-8") as f:
        cfg = json.load(f)
    b = cfg.get("books") or {}
    return b.get("tags") or {}, b.get("decks") or {}


def source_of(meta, reg, tag_rules, deck_rules):
    """The book id of one card from its levels.json record (deck, tags)."""
    tags = (meta.get("tags") or "").split()
    for t in tags:
        if t.startswith("source::"):
            parts = t.split("::")
            if len(parts) > 1 and parts[1]:
                return reg.id(parts[1])
    for t in tags:
        head = t.split("::")[0]
        if head in tag_rules:
            return reg.id(tag_rules[head])
    deck = meta.get("deck") or ""
    segs = deck.split("::")
    for i in range(len(segs), 0, -1):
        seg = segs[i - 1]
        prefix = "::".join(segs[:i])
        for k in (prefix, seg):
            if k in deck_rules:
                return reg.id(deck_rules[k])
        bid = reg.known(seg)
        if bid:
            return bid
    return ""


def derive(records, module_dir, reg=None):
    """{card id: book id} for every card of records ({id: {deck, tags}})
    that has a book."""
    reg = reg or Registry()
    tag_rules, deck_rules = load_rules(module_dir)
    out = {}
    for cid in sorted(records):
        bid = source_of(records[cid], reg, tag_rules, deck_rules)
        if bid:
            out[cid] = bid
    return out


def write(module_dir, sources):
    with open(os.path.join(module_dir, SOURCES_FILE), "w", encoding="utf-8") as f:
        json.dump(sources, f, indent=1, ensure_ascii=False, sort_keys=True)
        f.write("\n")


def apply(records, module_dir, write_file=True):
    sources = derive(records, module_dir)
    if write_file:
        write(module_dir, sources)
    return sources


def main(argv):
    args = [a for a in argv[1:] if not a.startswith("--")]
    if len(args) != 1:
        print(__doc__)
        return 2
    mdir = os.path.join(LOCAL_ROOT, "modules-local", args[0])
    p = os.path.join(mdir, "levels.json")
    if not os.path.exists(p):
        sys.exit("no %s: import the module first" % p)
    with open(p, encoding="utf-8") as f:
        records = json.load(f)
    sources = apply(records, mdir, write_file="--check" not in argv)
    reg = Registry()
    counts = Counter(sources.values())
    for bid, n in counts.most_common():
        print("%6d  %-28s %s" % (n, bid, reg.labels.get(bid, "")))
    print("%6d  (no book: served with the generated questions)" % (len(records) - len(sources)))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
