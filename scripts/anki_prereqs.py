#!/usr/bin/env python3
"""Re-apply an Anki module's subject prerequisites without re-importing.

    python3 scripts/anki_prereqs.py <module> [--check]

scripts/import_anki.py writes a module's concepts/ from the collection and
the subjects file (anki.json "subjects_file"): each subject's "after" list
becomes the prerequisites of its sets (anki.<subject>, anki.<subject>.2,
...). Editing "after" by hand (a link found by scripts/audit_graph, say)
would otherwise wait for the next import. This rewrites the prerequisites
of the sets that exist from the subjects file, as the importer would:
"@subject" is that subject's first set, and every id must be in the graph.
--check only reports sets whose prerequisites differ from the file.

MATHUA_ROOT names the install whose modules-local/ holds the module.
"""

import glob
import json
import os
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
LOCAL_ROOT = os.path.abspath(os.environ.get("MATHUA_ROOT") or ROOT)


def known_ids(module_dir):
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


def main(argv):
    args = [a for a in argv[1:] if not a.startswith("--")]
    check = "--check" in argv
    if len(args) != 1:
        print(__doc__)
        return 2
    mdir = os.path.join(LOCAL_ROOT, "modules-local", args[0])
    if not os.path.isdir(mdir):
        mdir = os.path.join(ROOT, "modules", args[0])
    cfg_path = os.path.join(mdir, "anki.json")
    if not os.path.isfile(cfg_path):
        sys.exit("%s has no anki.json" % mdir)
    cfg = json.load(open(cfg_path, encoding="utf-8"))
    subjects_file = cfg.get("subjects_file")
    if not subjects_file:
        sys.exit("%s names no subjects_file" % cfg_path)
    subjects = json.load(open(os.path.join(mdir, subjects_file), encoding="utf-8"))["subjects"]
    by_id = {s["id"]: s for s in subjects}
    known = known_ids(mdir)

    def set_id(cid):
        """The subject a set id belongs to, or None."""
        if not cid.startswith("anki."):
            return None
        rest = cid[5:]
        base, _, n = rest.rpartition(".")
        if n.isdigit() and base in by_id:
            return base
        return rest if rest in by_id else None

    changed, errors = 0, []
    for path in sorted(glob.glob(os.path.join(mdir, "concepts", "*.json"))):
        nodes = json.load(open(path, encoding="utf-8"))
        dirty = False
        for c in nodes:
            sid = set_id(c["id"])
            if sid is None:
                continue
            want = []
            for p in by_id[sid].get("after", []):
                if p.startswith("@"):
                    if p[1:] not in by_id:
                        errors.append("%s: unknown subject %s" % (sid, p))
                        continue
                    p = "anki." + p[1:]
                    if p not in known:
                        continue  # a subject with no cards has no set
                if p not in known:
                    errors.append("%s: after %s, which is not in the graph" % (sid, p))
                    continue
                if p not in want:
                    want.append(p)
            if c["prerequisites"] != want:
                print("%s: %s -> %s" % (c["id"], c["prerequisites"], want))
                c["prerequisites"] = want
                dirty = True
                changed += 1
        if dirty and not check:
            with open(path, "w", encoding="utf-8") as f:
                json.dump(nodes, f, indent=1, ensure_ascii=False)  # as the importer writes it
    for e in errors:
        print("ERROR", e)
    if errors:
        return 1
    if check:
        print("%d sets differ from the subjects file" % changed if changed else "ok")
        return 1 if changed else 0
    print("%d sets updated" % changed)
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
