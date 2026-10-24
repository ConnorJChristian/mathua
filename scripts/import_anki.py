#!/usr/bin/env python3
"""Import Anki cards into a local Mathua module.

    python3 scripts/import_anki.py <export.colpkg|.apkg> --config modules-local/anki/anki.json

The output is a module in modules-local/ (git-ignored, never pushed):

    modules-local/<id>/module.json        tracks and domain labels from the config
    modules-local/<id>/concepts/*.json    one topic per deck (large decks split)
    modules-local/<id>/bank/*.json        one question per Anki card
    modules-local/<id>/files/media/       the images the cards use

Re-running replaces those generated folders and keeps anything else in the
module (readings.json, files/ PDFs, src/ authored banks).

Each card becomes a typed-answer question graded by Mathua's math graders,
a true/false choice, or a flip card, in that order of preference; see
scripts/anki_typed.py. Typed answers are checked with the real graders first,
which needs Node (web/next-app with its packages installed), Go and SymPy;
--no-typed keeps every card a flip card.

Your Anki scheduling is not imported: Mathua schedules the cards itself from
your answers here, and nothing is written back to Anki.

The config chooses which decks become which domains:

    {
      "module": {"id": "anki", "name": "My Anki cards",
                 "tracks": [{"id": "econometrics", "label": "Econometrics",
                             "no_placement": true, "domains": ["anki_wooldridge"]}]},
      "groups": [
        {"deck": "Wooldridge - Introductory Econometrics 7e",
         "domain": "anki_wooldridge", "label": "Anki · Wooldridge",
         "book": "Wooldridge, Introductory Econometrics 7e"}
      ],
      "max_cards_per_topic": 60
    }

A deck matches a group when its name equals the group's "deck" or starts with
it followed by "::". Decks no group matches are skipped, as are Image
Occlusion notes and note types without a front and back.

Needs the `zstandard` package for current exports (pip install zstandard).
"""

import argparse
import html
import json
import os
import re
import shutil
import sqlite3
import sys
import tempfile
import subprocess
import zipfile
from collections import OrderedDict, defaultdict

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import anki_typed  # noqa: E402

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
RECALL_MARKER = "<!--recall-back-->"  # internal/pytrack/track.go RecallMarker
SEP = "\x1f"


# ---------------------------------------------------------------- reading the export

def zstd_decompress(data):
    try:
        import zstandard
    except ImportError:
        sys.exit("This export is zstd-compressed: run `pip install zstandard` and try again.")
    return zstandard.ZstdDecompressor().decompressobj().decompress(data)


def is_zstd(data):
    return data[:4] == b"\x28\xb5\x2f\xfd"


def read_varint(buf, i):
    shift = result = 0
    while True:
        b = buf[i]
        i += 1
        result |= (b & 0x7F) << shift
        if not b & 0x80:
            return result, i
        shift += 7


def parse_media_protobuf(buf):
    """MediaEntries { repeated MediaEntry entries = 1; }
    MediaEntry { string name = 1; uint32 size = 2; bytes sha1 = 3; optional uint32 legacy_zip_filename = 255; }"""
    entries = []
    i = 0
    while i < len(buf):
        key, i = read_varint(buf, i)
        field, wire = key >> 3, key & 7
        if wire == 2:
            ln, i = read_varint(buf, i)
            chunk, i = buf[i:i + ln], i + ln
            if field == 1:
                name, legacy = None, None
                j = 0
                while j < len(chunk):
                    k, j = read_varint(chunk, j)
                    f, w = k >> 3, k & 7
                    if w == 2:
                        l2, j = read_varint(chunk, j)
                        if f == 1:
                            name = chunk[j:j + l2].decode("utf-8")
                        j += l2
                    elif w == 0:
                        v, j = read_varint(chunk, j)
                        if f == 255:
                            legacy = v
                    else:
                        raise ValueError("unexpected wire type in media map")
                entries.append((name, legacy))
        elif wire == 0:
            _, i = read_varint(buf, i)
        else:
            raise ValueError("unexpected wire type in media map")
    return entries


def open_export(path, workdir):
    """Returns (sqlite connection, {media name: zip member}, zipfile)."""
    z = zipfile.ZipFile(path)
    names = set(z.namelist())
    db_path = os.path.join(workdir, "collection.db")
    if "collection.anki21b" in names:
        data = z.read("collection.anki21b")
        open(db_path, "wb").write(zstd_decompress(data) if is_zstd(data) else data)
    elif "collection.anki21" in names:
        open(db_path, "wb").write(z.read("collection.anki21"))
    elif "collection.anki2" in names:
        open(db_path, "wb").write(z.read("collection.anki2"))
    else:
        sys.exit("No Anki collection found in %s" % path)
    media = {}
    if "media" in names:
        raw = z.read("media")
        if is_zstd(raw):
            for idx, (name, legacy) in enumerate(parse_media_protobuf(zstd_decompress(raw))):
                if name:
                    media[name] = str(legacy if legacy is not None else idx)
        else:
            for k, v in json.loads(raw.decode("utf-8") or "{}").items():
                media[v] = k
    db = sqlite3.connect(db_path)
    db.create_collation("unicase", lambda a, b: (a.lower() > b.lower()) - (a.lower() < b.lower()))
    return db, media, z


def load_notetypes(db):
    out = {}
    tables = {r[0] for r in db.execute("select name from sqlite_master where type='table'")}
    if "notetypes" in tables:
        for ntid, name in db.execute("select id, name from notetypes"):
            fields = [f for (f,) in db.execute("select name from fields where ntid=? order by ord", (ntid,))]
            out[ntid] = (name, fields)
    else:  # legacy schema: models json in col
        models = json.loads(db.execute("select models from col").fetchone()[0])
        for mid, m in models.items():
            out[int(mid)] = (m["name"], [f["name"] for f in m["flds"]])
    return out


def load_decks(db):
    tables = {r[0] for r in db.execute("select name from sqlite_master where type='table'")}
    if "decks" in tables:
        return {did: name.replace(SEP, "::") for did, name in db.execute("select id, name from decks")}
    decks = json.loads(db.execute("select decks from col").fetchone()[0])
    return {int(k): v["name"] for k, v in decks.items()}


# ---------------------------------------------------------------- card text

CLOZE_OPEN_RE = re.compile(r"\{\{c(\d+)::")


def parse_cloze(text):
    """Cloze text as a tree: a list of strings and (number, children, hint).

    Deletions nest ({{c4::setting two {{c1::algebraic expressions}}}}), and a
    deletion ends at the "}}" that balances it, as in Anki. A "::" at the
    deletion's own level starts its hint."""
    def parse(i, top):
        out, buf = [], ""
        while i < len(text):
            m = CLOZE_OPEN_RE.match(text, i)
            if m:
                if buf:
                    out.append(buf)
                    buf = ""
                children, hint, i = parse(m.end(), False)
                out.append((int(m.group(1)), children, hint))
                continue
            if not top and text.startswith("}}", i):
                if buf:
                    out.append(buf)
                hint = None
                # The hint is the part after the last top-level "::".
                for k in range(len(out) - 1, -1, -1):
                    if isinstance(out[k], str) and "::" in out[k]:
                        before, after = out[k].rsplit("::", 1)
                        hint = after + "".join(x for x in out[k + 1:] if isinstance(x, str))
                        out = out[:k] + ([before] if before else [])
                        break
                return out, hint, i + 2
            buf += text[i]
            i += 1
        if buf:
            out.append(buf)
        return out, None, i
    tree, _, _ = parse(0, True)
    return tree


def render_cloze(tree, n, reveal):
    out = []
    for node in tree:
        if isinstance(node, str):
            out.append(node)
            continue
        num, children, hint = node
        if num != n:
            out.append(render_cloze(children, n, reveal))
        elif reveal:
            out.append("<b>" + render_cloze(children, n, True) + "</b>")
        else:
            out.append("<b>[" + (hint or "...") + "]</b>")
    return "".join(out)


def cloze_bodies(tree, n):
    """The text of every deletion numbered n, all other deletions revealed."""
    found = []
    for node in tree:
        if isinstance(node, str):
            continue
        num, children, _ = node
        if num == n:
            found.append(render_cloze(children, n, True))
        else:
            found.extend(cloze_bodies(children, n))
    return found


def cloze_numbers(tree):
    nums = set()
    for node in tree:
        if not isinstance(node, str):
            nums.add(node[0])
            nums |= cloze_numbers(node[1])
    return nums


def media_src(module_id):
    def fix(m):
        name = html.unescape(m.group(2))
        if re.match(r"^[a-z]+:", name):
            return m.group(0)
        return m.group(1) + "/api/modules/%s/files/media/%s" % (module_id, quote(name)) + m.group(3)
    return fix


def quote(name):
    from urllib.parse import quote as q
    return q(name, safe="")


IMG_RE = re.compile(r'(<img[^>]*?\bsrc=")([^"]+)(")', re.I)


def clean(text, module_id):
    text = IMG_RE.sub(media_src(module_id), text)
    text = text.replace(RECALL_MARKER, "")
    # Anki's [sound:...] tags have no player here.
    text = re.sub(r"\[sound:[^\]]+\]", "", text)
    return tidy_html(text).strip()


MATH_RE = re.compile(r"(\\\((?:.|\n)*?\\\)|\\\[(?:.|\n)*?\\\]|\$\$(?:.|\n)*?\$\$)")
TYPOS = {
    "elmination": "elimination", "betwen": "between", "eficient": "efficient",
    "unitialized": "uninitialized", "fillled": "filled", "ovestimate": "overestimate",
    "succint": "succinct", "succintly": "succinctly", "differnce": "difference",
    "indefininite": "indefinite", "ocurrence": "occurrence", "expresssed": "expressed",
    "caracter": "character", "aross": "across", "substraction": "subtraction",
    "propogates": "propagates", "desribed": "described", "acording": "according",
    "frequence": "frequency", "newasis": "newaxis", "styte": "style", "performant": "efficient",
    "foolproofs": "makes foolproof", "neighbourhood": "neighborhood", "behavioural": "behavioral",
}
TYPO_RE = re.compile(r"\b(" + "|".join(map(re.escape, TYPOS)) + r")\b")


def tidy_html(text):
    """Strip the editor residue Anki leaves in a card (fork change).

    Inside LaTeX, HTML entities are turned back into characters: an aligned
    environment exported as `&amp;` would otherwise render "amp;" in every
    column. Outside it, non-breaking spaces become spaces, inline font and
    alignment styling (fonts, colours, sizes) and empty spans go, doubled
    spaces collapse, and a few misspellings that recur across the scans are
    corrected. Structure (br, div, b, i, ol, ul, img, tables) is kept.
    """
    parts = MATH_RE.split(text)
    out = []
    for i, part in enumerate(parts):
        if i % 2 == 1:  # math
            part = html.unescape(part.replace("&nbsp;", " "))
            part = re.sub(r"<br\s*/?>", " ", part)
            out.append(part)
            continue
        part = part.replace("&nbsp;", " ").replace("\u00a0", " ")
        part = re.sub(r"</?(font|span|u)(\s[^>]*)?>", "", part)
        part = re.sub(r'\s+style="[^"]*"', "", part)
        part = re.sub(r'\s+class="(?!src)[^"]*"', "", part)
        part = re.sub(r"<div>\s*</div>", "", part)
        part = re.sub(r"[ \t]{2,}", " ", part)
        part = TYPO_RE.sub(lambda m: TYPOS[m.group(1)], part)
        out.append(part)
    return "".join(out)


def referenced_media(text):
    return [html.unescape(m.group(2)) for m in IMG_RE.finditer(text) if not re.match(r"^[a-z]+:", m.group(2))]


def card_sides(nt_name, fields, flds, ord_):
    """(front, back, answer, solution) for a card, or None. answer is the text
    the card asks for (a cloze's deleted text, or the top of a back), which
    decides whether it can become a typed or a choice question; solution is
    what to show after answering such a question."""
    f = dict(zip(fields, flds))
    vals = flds + [""] * 4
    if nt_name.startswith("Image Occlusion") or nt_name == "Study Path Reading":
        return None
    if "Cloze" in nt_name or "Text" in f and "{{c" in f.get("Text", ""):
        text = f.get("Text", vals[0])
        extra = f.get("Back Extra", f.get("Extra", ""))
        n = ord_ + 1
        tree = parse_cloze(text)
        if n not in cloze_numbers(tree):
            return None
        front = render_cloze(tree, n, reveal=False)
        back = render_cloze(tree, n, reveal=True)
        if extra.strip():
            back += "<br><br>" + extra
        bodies = cloze_bodies(tree, n)
        answer = anki_typed.strip_html(bodies[0]).strip() if len(bodies) == 1 else ""
        return front, back, answer, back
    front, back = f.get("Front", vals[0]), f.get("Back", vals[1])
    if ord_ == 1:  # the reversed card of a "(and reversed card)" note
        front, back = back, front
    if not front.strip() or not back.strip():
        return None
    return front, back, anki_typed.answer_head(back), anki_typed.answer_rest(back)


# ---------------------------------------------------------------- topics

def slug(s, n=40):
    s = re.sub(r"[^a-z0-9]+", "_", s.lower()).strip("_")
    return s[:n].strip("_") or "x"


def topic_tag(tags):
    for t in tags.split():
        if t.lower().startswith("topic::"):
            return t.split("::", 1)[1]
    return ""


class Subjects:
    """Sorting cards by subject rather than by deck (the "subjects" file).

    Each subject names the Mathua domain and subdomain its cards join, and the
    concepts they come after, so a set of cards sits in the graph next to the
    concept it practises:

        {"id": "derivative_rules", "domain": "calculus",
         "subdomain": "calculus.derivatives", "label": "Derivative rules",
         "after": ["calc.deriv.chain_rule", "@limits_continuity"],
         "topics": ["derivative_and_the_rules_of_differentiation"],
         "decks": ["Module 2: One Variable Calculus (basic)"]}

    "@id" in "after" is another subject. A card goes to the subject that lists
    its topic:: tag. A card without one goes by its deck: the subject whose
    "decks" entry matches deepest in the deck path, unless a "routes" entry
    for the same or a deeper part of the path matches the card's text:

        {"decks": ["Module 2: One Variable Calculus (basic)"],
         "match": "\\\\int|[Ii]ntegra", "subject": "integration"}

    A route with "topics" instead of "decks" re-sorts tagged cards: a card
    with that topic tag whose text matches goes to the route's subject.
    """

    def __init__(self, cfg):
        self.subjects = OrderedDict((x["id"], x) for x in cfg["subjects"])
        self.by_topic = {}
        for x in cfg["subjects"]:
            for t in x.get("topics", []):
                if t in self.by_topic:
                    sys.exit("topic %r is listed by subjects %r and %r" % (t, self.by_topic[t], x["id"]))
                self.by_topic[t] = x["id"]
        self.routes = [dict(r, _re=re.compile(r["match"])) for r in cfg.get("routes", [])]
        for r in self.routes:
            if r["subject"] not in self.subjects:
                sys.exit("route to unknown subject %r" % r["subject"])

    @staticmethod
    def _where(deck, pattern):
        """How deep a pattern sits in a deck path: (end, length), or None."""
        i = deck.rfind(pattern)
        return None if i < 0 else (i + len(pattern), len(pattern))

    def of(self, tags, deck, text):
        # A topic tag may nest (topic::calculus::partial_derivatives); the
        # subject is named by its first part.
        t = topic_tag(tags).split("::")[0]
        for r in self.routes:
            if t and t in r.get("topics", ()) and r["_re"].search(text):
                return r["subject"]
        if t in self.by_topic:
            return self.by_topic[t]
        best, best_at = None, None
        for sid, x in self.subjects.items():
            for d in x.get("decks", []):
                at = self._where(deck, d)
                if at and (best_at is None or at > best_at):
                    best, best_at = sid, at
        for r in self.routes:
            for d in r.get("decks", ()):
                at = self._where(deck, d)
                if at and (best_at is None or at >= best_at) and r["_re"].search(text):
                    return r["subject"]
        return best


def known_concepts(module_id):
    """Every concept id outside this module, to check "after" against."""
    import glob
    ids = set()
    paths = glob.glob(os.path.join(ROOT, "data", "concepts", "*.json"))
    for area in ("modules", "modules-local"):
        for d in glob.glob(os.path.join(ROOT, area, "*")):
            if os.path.basename(d) != module_id:
                paths += glob.glob(os.path.join(d, "concepts", "*.json"))
    for p in paths:
        data = json.load(open(p, encoding="utf-8"))
        for c in data if isinstance(data, list) else data.get("concepts", []):
            if isinstance(c, dict) and "id" in c:
                ids.add(c["id"])
    return ids


def build(args):
    cfg = json.load(open(args.config, encoding="utf-8"))
    mod = cfg["module"]
    module_id = mod["id"]
    out = os.path.join(ROOT, "modules-local", module_id)
    max_per = int(cfg.get("max_cards_per_topic", 60))
    groups = cfg["groups"]
    subjects = None
    if cfg.get("subjects_file"):
        subjects = Subjects(json.load(open(os.path.join(os.path.dirname(args.config), cfg["subjects_file"]), encoding="utf-8")))

    with tempfile.TemporaryDirectory() as work:
        db, media_map, z = open_export(args.export, work)
        notetypes = load_notetypes(db)
        decks = load_decks(db)

        def group_of(deck):
            for g in groups:
                if deck == g["deck"] or deck.startswith(g["deck"] + "::"):
                    return g
            return None

        # One item per card, grouped by (group, sub-deck path). Cards in deck
        # order, then by note and card order, so topics read like the deck.
        rows = db.execute(
            "select c.id, c.nid, c.ord, c.did, n.mid, n.flds, n.tags from cards c join notes n on n.id = c.nid order by c.did, n.id, c.ord"
        ).fetchall()
        by_topic = OrderedDict()
        skipped = defaultdict(int)
        media_used = set()
        for cid, nid, ord_, did, mid, flds, tags in rows:
            deck = decks.get(did, "")
            g = group_of(deck)
            if g is None:
                skipped["deck not in config"] += 1
                continue
            nt_name, fields = notetypes.get(mid, ("?", []))
            sides = card_sides(nt_name, fields, flds.split(SEP), ord_)
            if sides is None:
                skipped[nt_name] += 1
                continue
            front, back, answer, solution = sides
            for t in (front, back):
                media_used.update(referenced_media(t))
            rel = deck[len(g["deck"]):].lstrip(":") or g.get("label", g["deck"])
            key = (g["domain"], rel)
            if subjects:
                sid = subjects.of(tags, deck, front + "\n" + back)
                if sid is None:
                    skipped["no subject for deck " + rel.split("::")[0][:50]] += 1
                    continue
                key = ("@", sid)
            card = {"id": "n%dc%d" % (nid, ord_), "front": clean(front, module_id), "back": clean(back, module_id),
                    "tags": tags, "answer": answer, "solution": clean(solution, module_id),
                    "book": g.get("book", g.get("label", ""))}
            by_topic.setdefault(key, {"group": g, "rel": rel, "cards": []})["cards"].append(card)

        # Cards Mathua can ask in its own way: a typed answer graded by the
        # math graders, or a true/false choice. Everything else stays a
        # flip card. See scripts/anki_typed.py.
        stats = defaultdict(int)
        if not args.no_typed:
            all_cards = [c for t in by_topic.values() for c in t["cards"]]
            convert_cards(all_cards, work, stats)
            if args.report:
                rejected = [{"id": c["id"], "front": c["front"][:300], **c["typed_rejected"]}
                            for c in all_cards if "typed_rejected" in c]
                json.dump(rejected, open(args.report, "w", encoding="utf-8"), indent=1, ensure_ascii=False)

        # Split large decks into practice sets of a sensible size. Cards keep
        # deck order; runs that share a topic:: tag stay together where they
        # fit, small runs are packed into one set, and an oversized run is cut
        # into equal parts (60 + 60 + 4 becomes 42 + 41 + 41).
        concepts = defaultdict(list)  # domain -> concept dicts
        banks = defaultdict(list)
        used_ids = set()

        def even(cards):
            k = -(-len(cards) // max_per)
            size = -(-len(cards) // k)
            return [cards[i:i + size] for i in range(0, len(cards), size)]

        if subjects:
            # Subject order, so concepts and banks read in the order the
            # subjects file gives them.
            order = {sid: i for i, sid in enumerate(subjects.subjects)}
            by_topic = OrderedDict(sorted(by_topic.items(), key=lambda kv: order[kv[0][1]]))
            made = {}  # subject -> its first concept id
        for (domain, rel), t in by_topic.items():
            g = t["group"]
            sub_cfg = subjects.subjects[rel] if domain == "@" else None
            if sub_cfg:
                domain = sub_cfg["domain"]
            sets = []  # (cards, tags in it)
            if len(t["cards"]) <= max_per:
                sets.append((t["cards"], []))
            else:
                runs = OrderedDict()
                for c in t["cards"]:
                    runs.setdefault(topic_tag(c["tags"]), []).append(c)
                cur, cur_tags = [], []
                for tag, cards in runs.items():
                    if len(cards) > max_per:
                        if cur:
                            sets.append((cur, cur_tags))
                            cur, cur_tags = [], []
                        for part in even(cards):
                            sets.append((part, [tag] if tag else []))
                        continue
                    if len(cur) + len(cards) > max_per:
                        sets.append((cur, cur_tags))
                        cur, cur_tags = [], []
                    cur = cur + cards
                    if tag:
                        cur_tags.append(tag)
                if cur:
                    sets.append((cur, cur_tags))
            if sub_cfg:
                sources = []
                for c in t["cards"]:
                    b = c.get("book", "")
                    if b and b not in sources:
                        sources.append(b)
                for k, (chunk, _) in enumerate(sets):
                    cid = "anki." + sub_cfg["id"] + ("" if k == 0 else ".%d" % (k + 1))
                    if k == 0:
                        made[sub_cfg["id"]] = cid
                    label = "Practice · " + sub_cfg["label"]
                    if len(sets) > 1:
                        label += " (%d of %d)" % (k + 1, len(sets))
                    concepts[domain].append({
                        "id": cid, "label": label, "domain": domain,
                        "subdomain": sub_cfg["subdomain"], "grading_type": "bank",
                        "prerequisites": list(sub_cfg.get("after", [])),
                        "mastery_threshold": {"streak": 3, "avg_time_seconds": 90.0},
                    })
                    seen = set()
                    items = []
                    for c in chunk:
                        if c["id"] in seen or not c["front"] or not c["back"]:
                            continue
                        seen.add(c["id"])
                        items.append(item_for(c))
                    banks[domain].append({
                        "id": cid,
                        "book": {"name": "; ".join(sources) or "Anki", "section": "",
                                 "title": sub_cfg["label"], "url": ""},
                        "intro": "", "items": items,
                    })
                continue
            leaf = rel.split("::")[-1]
            counts = defaultdict(int)
            for _, tags in sets:
                counts[tuple(tags)] += 1
            seen_label = defaultdict(int)
            for chunk, tags in sets:
                label = leaf
                if len(sets) > 1:
                    if tags:
                        names = [x.split("::")[-1].replace("_", " ") for x in tags]
                        label += " · " + (", ".join(names[:2]) + (" and more" if len(names) > 2 else ""))
                    seen_label[tuple(tags)] += 1
                    if counts[tuple(tags)] > 1:
                        label += " (%d of %d)" % (seen_label[tuple(tags)], counts[tuple(tags)])
                    if not tags and counts[()] == 1:
                        label += " (other)"
                base = "anki.%s.%s" % (slug(domain.replace("anki_", ""), 20), slug(rel, 48))
                cid = base
                n = 1
                while cid in used_ids:
                    n += 1
                    cid = "%s.%d" % (base, n)
                used_ids.add(cid)
                sub = slug(rel.split("::")[0], 40)
                concepts[domain].append({
                    "id": cid, "label": label, "domain": domain,
                    "subdomain": domain + "." + sub, "grading_type": "bank",
                    "prerequisites": [],
                    "mastery_threshold": {"streak": 3, "avg_time_seconds": 90.0},
                })
                seen = set()
                items = []
                for c in chunk:
                    if c["id"] in seen or not c["front"] or not c["back"]:
                        continue
                    seen.add(c["id"])
                    items.append(item_for(c))
                banks[domain].append({
                    "id": cid,
                    "book": {"name": g.get("book", g.get("label", "")), "section": "",
                             "title": rel.replace("::", " › "), "url": ""},
                    "intro": "", "items": items,
                })

        if subjects:
            # "@subject" means that subject's first set. A subject with no
            # cards drops out, and so does any "after" that points at it.
            known = known_concepts(module_id) | {c["id"] for cs in concepts.values() for c in cs}
            missing = set()
            for cs in concepts.values():
                for c in cs:
                    pre = []
                    for p in c["prerequisites"]:
                        if p.startswith("@"):
                            if p[1:] not in subjects.subjects:
                                sys.exit("%s: unknown subject %s" % (c["id"], p))
                            p = made.get(p[1:])
                            if p is None:
                                continue
                        if p not in known:
                            missing.add(p)
                        pre.append(p)
                    c["prerequisites"] = pre
            if missing:
                sys.exit("subjects come after concepts that do not exist: %s" % ", ".join(sorted(missing)))

        # Write: replace only what this importer owns.
        for sub in ("concepts", "bank", os.path.join("files", "media")):
            p = os.path.join(out, sub)
            if os.path.isdir(p):
                shutil.rmtree(p)
            os.makedirs(p, exist_ok=True)
        manifest = dict(mod)
        labels = dict(manifest.get("domain_labels", {}))
        if not subjects:
            for g in groups:
                labels.setdefault(g["domain"], g.get("label", g["domain"]))
        manifest["domain_labels"] = labels
        manifest.setdefault("name", module_id)
        json.dump(manifest, open(os.path.join(out, "module.json"), "w", encoding="utf-8"), indent=2, ensure_ascii=False)
        n_items = 0
        for domain in concepts:
            fname = domain if domain.startswith("anki_") else "anki_" + domain
            json.dump(concepts[domain], open(os.path.join(out, "concepts", fname + ".json"), "w", encoding="utf-8"),
                      indent=1, ensure_ascii=False)
            bank = {"chapter": domain, "title": labels.get(domain, domain),
                    "concepts": [b for b in banks[domain] if b["items"]]}
            n_items += sum(len(b["items"]) for b in bank["concepts"])
            json.dump(bank, open(os.path.join(out, "bank", fname + ".json"), "w", encoding="utf-8"),
                      indent=1, ensure_ascii=False)
        copied = 0
        mdir = os.path.join(out, "files", "media")
        for name in sorted(media_used):
            member = media_map.get(name)
            if member is None or "/" in name or "\\" in name or name.startswith("."):
                continue
            try:
                data = z.read(member)
            except KeyError:
                continue
            if is_zstd(data):
                data = zstd_decompress(data)
            open(os.path.join(mdir, name), "wb").write(data)
            copied += 1

    print("%s: %d topics, %d cards, %d images -> %s" % (
        module_id, sum(len(v) for v in concepts.values()), n_items, copied, os.path.relpath(out, ROOT)))
    for k in ("math", "mc", "recall"):
        print("  %-6s %d" % (k, stats["kind " + k]))
    for k, v in sorted(stats.items(), key=lambda kv: -kv[1]):
        if not k.startswith("kind "):
            print("  kept as a flip card, %d: %s" % (v, k))
    for k, v in sorted(skipped.items(), key=lambda kv: -kv[1]):
        print("  skipped %d cards: %s" % (v, k))


# ---------------------------------------------------------------- Mathua-style questions

OPPOSITE = {"True": "False", "False": "True", "Yes": "No", "No": "Yes"}


def item_for(c):
    """The bank item for one card, in the form convert_cards chose."""
    stats_kind = c.get("kind", "recall")
    if stats_kind == "math":
        return {"id": c["id"], "kind": "math", "level": 2, "prompt": c["front"],
                "parts": c["parts"], "back": c["solution"], "explain": ""}
    if stats_kind == "mc":
        return {"id": c["id"], "kind": "mc", "level": 2, "prompt": c["front"],
                "correct": c["verdict"], "wrong": [OPPOSITE[c["verdict"]]], "explain": c["back"]}
    return {"id": c["id"], "kind": "recall", "level": 2, "prompt": c["front"], "back": c["back"], "explain": ""}


def convert_cards(cards, work, stats):
    """Mark each card as math, mc or recall (the default), checking every
    typed answer with Mathua's own graders first."""
    pending = []  # (card, parts)
    for c in cards:
        v = anki_typed.verdict(c["answer"])
        if v:
            c["kind"], c["verdict"] = "mc", v
            continue
        if not c["answer"]:
            continue
        parts, why = anki_typed.typed_candidate(c["front"], c["answer"])
        if parts:
            pending.append((c, parts))
        elif why not in ("no formula", "words around the formula"):
            stats[why] += 1
    if pending:
        check_typed(pending, work, stats)
    for c in cards:
        stats["kind " + c.get("kind", "recall")] += 1


def check_typed(pending, work, stats):
    """Fill in each part's answer as Mathua's math editor sends it, grade it
    with the real graders, and keep the cards whose every part passes."""
    texs = [p["tex"] for _, parts in pending for p in parts]
    tin, tout = os.path.join(work, "tex.json"), os.path.join(work, "plain.json")
    json.dump(texs, open(tin, "w"))
    app = os.path.join(ROOT, "web", "next-app")
    try:
        subprocess.run(["node", "--experimental-strip-types", "--no-warnings",
                        os.path.join(app, "scripts", "mathlive-plain.mjs"), tin, tout],
                       cwd=app, check=True)
        plain = json.load(open(tout))
    except (OSError, subprocess.CalledProcessError) as ex:
        print("  typed answers skipped: could not run the math editor (%s). Run npm ci in web/next-app." % ex)
        stats["typed answers skipped (no math editor)"] += len(pending)
        return
    lines, k = [], 0
    for ci, (c, parts) in enumerate(pending):
        for pi, p in enumerate(parts):
            p["answer"] = plain[k]
            k += 1
            lines.append(json.dumps({"id": "%d.%d" % (ci, pi), "grading": p["grading"], "expected": p["answer"],
                                     "ref": p["ref"], "wrong": p["wrong"]}))
    try:
        res = subprocess.run(["go", "run", "./scripts/check_typed_answers"], cwd=ROOT, check=True,
                             input="\n".join(lines) + "\n", capture_output=True, text=True)
    except (OSError, subprocess.CalledProcessError) as ex:
        err = getattr(ex, "stderr", "") or str(ex)
        print("  typed answers skipped: the grader check failed to run: %s" % err.strip()[:300])
        stats["typed answers skipped (no grader)"] += len(pending)
        return
    verdicts = {}
    for line in res.stdout.splitlines():
        v = json.loads(line)
        verdicts[v["id"]] = v
    for ci, (c, parts) in enumerate(pending):
        bad = [verdicts.get("%d.%d" % (ci, pi), {"reason": "not checked"}) for pi in range(len(parts))]
        bad = [v for v in bad if not v.get("ok")]
        if bad:
            stats[bad[0]["reason"].split(":")[0]] += 1
            c["typed_rejected"] = {"reason": bad[0]["reason"],
                                   "parts": [{"tex": p["tex"], "sent": p["answer"], "ref": p["ref"]} for p in parts]}
            continue
        c["kind"] = "math"
        c["parts"] = [{"label": p["label"], "answer": p["answer"], "grading": p["grading"], "tex": p["tex"]}
                      for p in parts]


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("export", help=".colpkg or .apkg exported from Anki")
    ap.add_argument("--config", required=True, help="the import config (see the module docstring)")
    ap.add_argument("--no-typed", action="store_true",
                    help="keep every card a flip card (no typed answers or true/false questions)")
    ap.add_argument("--report", metavar="FILE",
                    help="write the typed answers the grader check rejected, with the reason, to FILE")
    build(ap.parse_args())


if __name__ == "__main__":
    main()
