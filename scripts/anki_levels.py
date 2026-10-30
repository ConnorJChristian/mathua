#!/usr/bin/env python3
"""Put imported Anki cards on Mathua's three scaffold steps.

    python3 scripts/anki_levels.py <module> [--export X.colpkg] [--check] [--sample N] [--seed S]

A bank item's `level` is its scaffold step (FORK.md, "The order of practice
inside a topic"): 1 apply the rule, 2 worked cases, 3 applications. The
importer used to give every card level 2, so an Anki topic never climbed.
This guesses each card's step from what the card is, and
scripts/import_anki.py applies it on every import:

  step 1  one fact, one definition or one step: a short front and a short
          back, a cloze or a term with one item hidden, a true/false check,
          a single-rule computation ("differentiate x^5").
  step 2  worked cases: a book's worked example, a computation of several
          steps, a cloze that hides several items, anything in between.
  step 3  derivations, proofs, word problems and applications, multi-part
          problems: a card tagged format::proof, one that asks to prove,
          derive, show or explain why, an economic setting worded as a
          problem, a prompt with parts (a), (b), ...

The signals are the card's fields and their length, its cloze count, its
tags (format::proof, format::worked_example, format::review_question,
econ::...), its deck path (a deck called Exercises, Problems or Proofs is
step 3, one called Review or Definitions step 1) and the kind the grading
rules gave it (scripts/audit_grading_style.py: a written task is step 3).

Overrides, in the module's level_overrides.json:

    {"cards": {"n1786236482047c0": 1},
     "decks": {"Wooldridge - Introductory Econometrics 7e::Ch02 The Simple Regression Model": 2,
               "Practice": 3}}

A card override wins over a deck override; a deck key matches the card's
deck path exactly, as a prefix ending at "::", or as one whole "::" segment
(the longest match wins).

Each card's level and the rule that chose it are recorded in the module's
levels.json, with the card's deck, tags, note type and cloze count (no card
text), so the levels can be re-applied without the export after an
override is edited: run this with no --export. --export reads the cards'
deck and tags from the collection again (read-only). --check reports what
would change and writes nothing. The histogram is printed per subject.

MATHUA_ROOT names the install whose modules-local/ holds the module.
"""

import glob
import html
import json
import os
import random
import re
import sys
from collections import Counter, OrderedDict, defaultdict

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
LOCAL_ROOT = os.path.abspath(os.environ.get("MATHUA_ROOT") or ROOT)

LEVELS_FILE = "levels.json"
OVERRIDES_FILE = "level_overrides.json"

TAG = re.compile(r"<[^>]+>")
SPACE = re.compile(r"\s+")
MATH = re.compile(r"\\\(.*?\\\)|\\\[.*?\\\]|\$\$.*?\$\$", re.S)
BLANK = re.compile(r"\[\.\.\.\]|\[[^\]]{1,40}\.\.\.\]|_{3,}")

# Step 3: a proof, a derivation, an argument.
WRITTEN = re.compile(
    r"\b(prove|proof|show that|show how|demonstrate that|derive|derivation|verify that|explain why|"
    r"explain how|justify|argue|why does|why is|why are|why do|in what sense|discuss|interpret|"
    r"compare and contrast)\b", re.I)
# Step 3: a problem set in the world (an economic or applied setting).
SETTING = re.compile(
    r"\b(a firm|the firm|a company|the company|a monopol|a consumer|the consumer|a producer|a worker|"
    r"households?|an investor|a bank|a government|the government|a city|a country|a farmer|a store|"
    r"a manufacturer|a researcher|a survey|a sample of|an economist|a student|"
    r"dollars?|\$\s?\d|per unit|per week|per month|per year|per hour|wage|salary|price of|"
    r"profit|revenue|cost of|costs? function|demand (function|curve|for)|supply (function|curve)|"
    r"interest rate|compounded|loan|invest|tax|subsid|budget|income|utility|output of|"
    r"population|bacteria|tank|ladder|balloon|car|machine|depreciat|sales|market)\b", re.I)
# Step 3: a prompt in parts.
PART = re.compile(r"(?:^|[\s>(])\(?([a-f]|i{1,3}|iv|v|[1-6])\)\s", re.I)
NUMBERED = re.compile(r"(?:^|[\s.?:])([1-6])\.\s+[A-Z]")
CHOICE = re.compile(r"\b(which of (the following|these)|which one|select (one|all)|choose)\b", re.I)
FILL = re.compile(r"which fills the blank\??\s*$", re.I)
VERDICTS = ("True", "False", "Yes", "No")
# Step 1: what a thing is called, a definition, one fact.
DEFINE = re.compile(
    r"^(define|definition of|state the|state |what is (a|an|the) [^?]{1,60}\?$|what does [^?]{1,60} (mean|stand for)|"
    r"what is meant by|what are (the )?[^?]{1,40}\?$|name the|what is the name|what do we call|"
    r"which (function|method|attribute|module|keyword|operator)|what (function|method|attribute) )", re.I)
TERMISH = re.compile(r"\b(is called|is known as|is termed|is defined as|we call|is referred to as)\b", re.I)
# Step 2 or more: a computation of several steps.
MULTISTEP = re.compile(
    r"\b(solve|find (all|the (equilibrium|maximum|minimum|extrem|critical|inverse|eigen|general|particular|"
    r"optimal|solution)|and)|integrate|optimi[sz]e|maximi[sz]e|minimi[sz]e|subject to|"
    r"(use|using) (cramer|lagrange|l'h|the (definition|quotient|chain|product) rule|implicit)|lagrang|hessian|jacobian|"
    r"system of|simultaneous|comparative static|kuhn|tucker|by parts|substitution|eigen|determinant of|"
    r"inverse of|gaussian|row reduc|test (for|whether)|estimate|sketch|graph|draw|plot|construct|"
    r"coordinate vector|basis|kernel|null ?space|rank of|span)\b", re.I)
ONESTEP = re.compile(
    r"^(differentiate|find (the )?(first )?derivative|find f'|find dy/dx|compute|calculate|evaluate|simplify|"
    r"expand|factor|write|express|convert|what is|find)\b", re.I)

DECK3 = re.compile(r"\b(exercises?|problems?|proofs?|applications?|word problems?)\b", re.I)
DECK1 = re.compile(r"^(key )?(definitions?|vocabulary|terms|glossary|notation|flashcards?)$", re.I)


def plain(s):
    """Card text without HTML, entities decoded, LaTeX kept."""
    s = TAG.sub(" ", s or "")
    s = html.unescape(s)
    return SPACE.sub(" ", s).strip()


def words(s):
    """Card text with the math folded to M, for the word rules."""
    return SPACE.sub(" ", MATH.sub(" M ", plain(s))).strip()


def tag_heads(tags):
    out = set()
    for t in (tags or "").split():
        t = t.lower()
        out.add(t)
        parts = t.split("::")
        for i in range(1, len(parts)):
            out.add("::".join(parts[:i]))
    return out


def level_of(item, meta):
    """(level, rule) for a bank item of an imported card.

    item is the bank item (kind, prompt, back or explain, parts); meta is
    {"deck", "tags", "notetype", "cloze"} from the collection, any of it
    possibly missing."""
    meta = meta or {}
    kind = item.get("kind", "recall")
    front = item.get("prompt", "")
    back = item.get("back") or item.get("explain") or ""
    fw, bw = words(front), words(back)
    fp, bp = plain(front), plain(back)
    tags = tag_heads(meta.get("tags", ""))
    deck = meta.get("deck", "")
    leaf = deck.split("::")[-1] if deck else ""
    cloze = int(meta.get("cloze") or 0)
    blanks = len(BLANK.findall(fp))
    parts = len(item.get("parts") or [])
    labels = {m.group(1).lower() for m in PART.finditer(" " + fw)}
    numbered = {m.group(1) for m in NUMBERED.finditer(" " + fw)}
    choice = bool(CHOICE.search(fw))
    # A multiple-choice item that is a fill-in or a true/false check of one
    # statement (scripts/anki_choice.py) rather than a problem with options.
    oneline = kind == "mc" and (FILL.search(fw) or (item.get("correct") in VERDICTS and not MULTISTEP.search(fw)))

    # Step 3, from the card's own markings first.
    if "format::proof" in tags:
        return 3, "tag format::proof"
    if leaf and DECK3.search(leaf):
        return 3, "deck %s" % leaf
    w = WRITTEN.search(fw)
    if w and not oneline and not choice:
        return 3, "asks to %s" % w.group(0).lower()
    inparts = (len(labels) >= 2 or len(numbered) >= 2) and not choice
    if inparts and len(fw) >= 160:
        return 3, "a problem in parts"
    setting = SETTING.search(fw)
    if setting and len(fw) >= 160 and not FILL.search(fw):
        return 3, "a problem set in the world (%s)" % setting.group(0).lower()
    if any(t.startswith("econ::") for t in tags) and len(fw) >= 220 and not FILL.search(fw):
        return 3, "an economics application"
    if setting and len(fw) >= 80 and not FILL.search(fw) and not choice:
        return 2, "a short applied problem (%s)" % setting.group(0).lower()
    if len(fw) >= 420 and blanks <= 1:
        return 3, "a long problem"

    if inparts:
        return 2, "a short problem in parts"

    # Step 1: one fact, one item, one step.
    if kind != "math" and DEFINE.search(fw) and len(fw) <= 120:
        return 1, "a definition or a name"
    if leaf and DECK1.search(leaf) and len(fw) < 220:
        return 1, "deck %s" % leaf
    if "format::review_question" in tags and len(bw) < 200:
        return 1, "tag format::review_question"
    if oneline and len(fw) <= 260 and "format::worked_example" not in tags:
        return 1, "a fill-in or true/false check"
    if choice and len(fw) <= 260 and not MULTISTEP.search(fw):
        return 1, "a choice among statements"
    if cloze or blanks:
        hidden = max(cloze, blanks)
        if hidden <= 1 and len(fw) <= 320:
            return 1, "a cloze of one item"
        if hidden <= 2 and len(fw) <= 200:
            return 1, "a short cloze"
        return 2, "a cloze of %d items" % hidden
    if "format::worked_example" in tags:
        return 2, "tag format::worked_example"
    if kind != "math" and (DEFINE.search(fw) or TERMISH.search(fw)) and len(fw) <= 200 and len(bw) <= 260:
        return 1, "a definition or a name"
    if kind == "math":
        if parts <= 1 and len(fw) <= 140 and not MULTISTEP.search(fw):
            return 1, "a single-rule computation"
        return 2, "a computation of several steps" if parts <= 1 else "a computation of %d parts" % parts
    if len(fw) <= 90 and len(bw) <= 90 and not MULTISTEP.search(fw):
        return 1, "a short front and back"
    if MULTISTEP.search(fw):
        return 2, "a computation of several steps"
    if len(fw) <= 140 and len(bw) <= 160:
        return 1, "a short card"
    return 2, "a worked case"


# ---------------------------------------------------------------- overrides

def deck_override(deck, decks):
    best, best_len = None, -1
    for key, lv in decks.items():
        hit = (deck == key or deck.startswith(key + "::") or deck.endswith("::" + key)
               or ("::" + key + "::") in deck)
        if hit and len(key) > best_len:
            best, best_len = (lv, key), len(key)
    return best


def decide(item, meta, overrides):
    cards = overrides.get("cards", {})
    if item["id"] in cards:
        return int(cards[item["id"]]), "override (card)"
    d = deck_override((meta or {}).get("deck", ""), overrides.get("decks", {}))
    if d:
        return int(d[0]), "override (deck %s)" % d[1]
    return level_of(item, meta)


def load_overrides(module_dir):
    p = os.path.join(module_dir, OVERRIDES_FILE)
    if not os.path.exists(p):
        return {}
    with open(p, encoding="utf-8") as f:
        ov = json.load(f)
    for section in ("cards", "decks"):
        for k, v in ov.get(section, {}).items():
            if v not in (1, 2, 3):
                sys.exit("%s: %s %r has level %r, want 1, 2 or 3" % (p, section, k, v))
    return ov


# ---------------------------------------------------------------- applying

def subject_of(concept_id):
    """anki.algebra_review.6 -> algebra_review."""
    s = concept_id.split(".", 1)[-1]
    return re.sub(r"\.\d+$", "", s)


def apply(concepts, meta, module_dir, write=True):
    """Set every item's level in concepts (bank concept dicts) and record
    them in levels.json. Returns ({card id: (old, new)} changed, histogram
    {subject: Counter}, records)."""
    overrides = load_overrides(module_dir)
    changed, hist, records = {}, defaultdict(Counter), OrderedDict()
    for c in concepts:
        for it in c["items"]:
            m = meta.get(it["id"], {})
            lv, rule = decide(it, m, overrides)
            if it.get("level") != lv:
                changed[it["id"]] = (it.get("level"), lv)
            it["level"] = lv
            hist[subject_of(c["id"])][lv] += 1
            rec = {"level": lv, "rule": rule}
            for k in ("deck", "tags", "notetype", "cloze"):
                if m.get(k):
                    rec[k] = m[k]
            records[it["id"]] = rec
    if write:
        with open(os.path.join(module_dir, LEVELS_FILE), "w", encoding="utf-8") as f:
            json.dump(records, f, indent=1, ensure_ascii=False, sort_keys=True)
            f.write("\n")
    return changed, hist, records


def print_histogram(hist, out=sys.stdout):
    tot = Counter()
    out.write("%-34s %6s %6s %6s %6s\n" % ("subject", "step 1", "step 2", "step 3", "cards"))
    for s in sorted(hist):
        h = hist[s]
        tot.update(h)
        out.write("%-34s %6d %6d %6d %6d\n" % (s[:34], h[1], h[2], h[3], sum(h.values())))
    out.write("%-34s %6d %6d %6d %6d\n" % ("all", tot[1], tot[2], tot[3], sum(tot.values())))


def rule_kind(rule):
    """A rule without its particulars, for the summary."""
    rule = re.sub(r" \(.*\)$", "", rule)
    rule = re.sub(r"^(deck|asks to) .*", r"\1 ...", rule)
    return re.sub(r"\d+", "N", rule)


def cloze_count(text, ord_):
    """How many deletions card ord_ (0-based) of a cloze note hides."""
    return len(re.findall(r"\{\{c%d::" % (ord_ + 1), text or ""))


def meta_from_export(export):
    """{card id: meta} from an Anki export, opened read-only in a temp dir."""
    import tempfile
    sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
    import import_anki  # noqa: E402
    out = {}
    with tempfile.TemporaryDirectory() as work:
        db, _, _ = import_anki.open_export(export, work)
        notetypes = import_anki.load_notetypes(db)
        decks = import_anki.load_decks(db)
        for nid, ord_, did, mid, flds, tags in db.execute(
                "select c.nid, c.ord, c.did, n.mid, n.flds, n.tags from cards c join notes n on n.id = c.nid"):
            name, fields = notetypes.get(mid, ("?", []))
            f = dict(zip(fields, flds.split(import_anki.SEP)))
            m = {"deck": decks.get(did, ""), "tags": tags.strip(), "notetype": name}
            if "Cloze" in name:
                m["cloze"] = cloze_count(f.get("Text", flds.split(import_anki.SEP)[0]), ord_)
            out["n%dc%d" % (nid, ord_)] = m
        db.close()
    return out


def meta_from_levels(module_dir):
    p = os.path.join(module_dir, LEVELS_FILE)
    if not os.path.exists(p):
        return {}
    with open(p, encoding="utf-8") as f:
        recs = json.load(f)
    return {k: {x: v[x] for x in ("deck", "tags", "notetype", "cloze") if x in v} for k, v in recs.items()}


def main(argv):
    args, export, sample, seed = [], None, 0, 1
    i = 1
    while i < len(argv):
        a = argv[i]
        if a in ("--export", "--sample", "--seed"):
            v = argv[i + 1]
            if a == "--export":
                export = v
            elif a == "--sample":
                sample = int(v)
            else:
                seed = int(v)
            i += 2
            continue
        if not a.startswith("--"):
            args.append(a)
        i += 1
    check = "--check" in argv
    if len(args) != 1:
        print(__doc__)
        return 2
    mdir = os.path.join(LOCAL_ROOT, "modules-local", args[0])
    if not os.path.isdir(mdir):
        sys.exit("no module %s" % mdir)
    meta = meta_from_export(export) if export else meta_from_levels(mdir)
    if not meta:
        sys.exit("no card metadata: run with --export once (or re-import)")
    files = [f for f in sorted(glob.glob(os.path.join(mdir, "bank", "*.json"))) if os.path.basename(f) != "skip.json"]
    data = {}
    for f in files:
        with open(f, encoding="utf-8") as fh:
            data[f] = json.load(fh)
    concepts = [c for d in data.values() for c in d["concepts"]]
    before = Counter(it.get("level") for c in concepts for it in c["items"])
    changed, hist, records = apply(concepts, meta, mdir, write=not check)
    print("levels before:", dict(sorted(before.items())))
    print_histogram(hist)
    rules = Counter(rule_kind(r["rule"]) for r in records.values())
    print("\nrules:")
    for k, v in rules.most_common():
        print("  %5d  %s" % (v, k))
    print("\n%d cards change level%s" % (len(changed), " (--check: nothing written)" if check else ""))
    if not check:
        for f, d in data.items():
            with open(f, "w", encoding="utf-8") as out:
                json.dump(d, out, indent=1, ensure_ascii=False)
                out.write("\n")
    if sample:
        rnd = random.Random(seed)
        items = [(c["id"], it) for c in concepts for it in c["items"]]
        for cid, it in rnd.sample(items, min(sample, len(items))):
            r = records[it["id"]]
            print("\n[%d] %s %s#%s (%s) — %s" % (r["level"], it["kind"], cid, it["id"], r.get("tags", "")[:80], r["rule"]))
            print("   Q:", words(it.get("prompt", ""))[:300])
            print("   A:", words(it.get("back") or it.get("explain") or "")[:160])
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
