#!/usr/bin/env python3
"""Build and check question banks written as Python authoring files.

A module (see internal/modules) that authors questions keeps them in

    modules/<id>/src/<NN>_<chapter>.py          (or modules-local/<id>/src/...)

and this script writes, next to them,

    modules/<id>/bank/<NN>_<chapter>.json       the question bank the server reads
    modules/<id>/concepts/<domain>.json         the chapter's concept-graph nodes

    python3 scripts/python_bank.py build              # every module with a src/
    python3 scripts/python_bank.py build numpy        # files or modules matching "numpy"
    python3 scripts/python_bank.py check              # verify the JSON is current

With MATHUA_ROOT set, modules-local/ is read from that install instead of
this checkout (a worktree builds the main checkout's local modules).

Every snippet runs through grading/pyrunner.py, the same runner the server
uses, so the bank is checked in the environment that grades learners:

  * "output" questions: the code is executed and its printed output becomes
    `expected`. If the author also wrote `expected`, the two must agree.
  * "code" tasks: the reference solution must pass every hidden test, and the
    starter code must fail at least one (otherwise the task grades itself).

Run it with the interpreter whose library versions the Docker image pins
(grading/requirements-python.txt); `check` fails if an output changed.

Authoring files are plain Python that call the helpers below. They are our own
source files, executed with runpy, never learner input.
"""

import concurrent.futures as cf
import glob
import json
import os
import runpy
import secrets
import subprocess
import sys
import tempfile

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
RUNNER = os.path.join(ROOT, "grading", "pyrunner.py")
# MATHUA_ROOT names the install whose modules-local/ to build, as it does for
# the server (internal/modules): a git worktree of this repository has an
# empty modules-local/, so its script can build the main checkout's local
# modules. Committed modules always come from this checkout.
LOCAL_ROOT = os.path.abspath(os.environ.get("MATHUA_ROOT") or ROOT)
BOOK = "https://jakevdp.github.io/PythonDataScienceHandbook/"


# ---------------------------------------------------------------- authoring API

def concept(id, label, prereqs, section, title, page, intro, items):
    """One concept. `page` is the book page name without .html.

    `label` and `prereqs` go to the concept graph (data/concepts/python_*.json);
    the rest goes to the question bank.
    """
    return {
        "id": id,
        "_label": label,
        "_prereqs": list(prereqs),
        "book": {"section": section, "title": title, "url": BOOK + page + ".html"},
        "intro": reflow(dedent(intro)),
        "items": items,
    }


def topic(id, label, prereqs, subdomain, source, intro, items):
    """One concept from any source, for modules other than the handbook.

    `subdomain` is the graph subdomain (e.g. "econometrics_python.regression");
    `source` names what the concept follows: a dict with "name" (the book or
    deck), "title" and optionally "section" and "url". It is shown with the
    note; quote nothing from a book here, link or name it.
    """
    return {
        "id": id,
        "_label": label,
        "_prereqs": list(prereqs),
        "_subdomain": subdomain,
        "book": {"name": source.get("name", ""), "section": source.get("section", ""),
                 "title": source["title"], "url": source.get("url", "")},
        "intro": reflow(dedent(intro)),
        "items": items,
    }


def mc(id, prompt, correct, wrong, explain, level=2):
    return {"id": id, "kind": "mc", "level": level, "prompt": reflow(dedent(prompt)),
            "correct": correct, "wrong": list(wrong), "explain": reflow(dedent(explain))}


def out(id, prompt, code, explain, level=2, expected=None):
    item = {"id": id, "kind": "output", "level": level, "prompt": reflow(dedent(prompt)),
            "code": dedent(code), "explain": reflow(dedent(explain))}
    if expected is not None:
        item["expected"] = expected
    return item


def task(id, prompt, solution, tests, explain, starter="", setup="", level=2, starter_may_pass=False):
    item = {"id": id, "kind": "code", "level": level, "prompt": reflow(dedent(prompt)),
            "starter": dedent(starter), "setup": dedent(setup), "solution": dedent(solution),
            "tests": tests, "explain": reflow(dedent(explain))}
    if starter_may_pass:
        item["_starter_may_pass"] = True
    return item


def book_ref(file, module, page, label, printed_page=None, end_page=None):
    """A page of the learner's own book as an item's prompt (fork change).

    Set it as an item's "book_ref" field. The bank then stores only this
    reference, the answer key and an explanation in the bank's own words:
    never the exercise's text. "unlocks_after": "<item id>" on another item
    of the same concept keeps that item back until this one is answered
    correctly (practice versions of a book exercise).
    """
    ref = {"file": file, "module": module, "page": page, "label": label}
    if end_page and end_page != page:
        ref["end_page"] = end_page
    if printed_page:
        ref["printed_page"] = printed_page
    return ref


def check(name, code, hint=""):
    t = {"name": name, "code": dedent(code)}
    if hint:
        t["hint"] = hint
    return t


def dedent(s):
    import textwrap
    return textwrap.dedent(s).strip("\n")


def reflow(md):
    """Join hard-wrapped prose lines into paragraphs.

    The pages render this Markdown with `white-space: pre-wrap`, so a line
    break in the source is a line break on screen. Authors wrap at 80 columns
    for readability here; the learner should see paragraphs. Code fences,
    tables, list items and headings keep their own lines.
    """
    out, para, in_fence = [], [], False

    def flush():
        if para:
            out.append(" ".join(para))
            para.clear()

    for line in md.split("\n"):
        stripped = line.strip()
        if stripped.startswith("```"):
            flush()
            in_fence = not in_fence
            out.append(line)
            continue
        if in_fence:
            out.append(line)
            continue
        structural = (not stripped or stripped.startswith(("|", "#", "- ", "* ", ">"))
                      or (stripped[:1].isdigit() and ". " in stripped[:4]))
        if structural:
            flush()
            if stripped.startswith(("- ", "* ")) or (stripped[:1].isdigit() and ". " in stripped[:4]):
                para.append(stripped)  # a list item; continuation lines join onto it
                continue
            out.append(line)
            continue
        para.append(stripped)
    flush()
    return "\n".join(out)


HELPERS = {"concept": concept, "topic": topic, "mc": mc, "out": out, "task": task, "check": check,
           "book_ref": book_ref}


def check_gating(cid, items):
    """book_ref and unlocks_after, as internal/pytrack/bank.go validates them."""
    errs = []
    ids = {it["id"] for it in items}
    for it in items:
        where = "%s#%s" % (cid, it["id"])
        ref = it.get("book_ref")
        if ref is not None:
            unknown = set(ref) - {"file", "module", "page", "end_page", "printed_page", "label"}
            if unknown:
                errs.append("%s: book_ref has unknown fields %s" % (where, sorted(unknown)))
            if not all(str(ref.get(k, "")).strip() for k in ("file", "module", "label")):
                errs.append("%s: book_ref needs file, module and label" % where)
            if "\n" in ref.get("label", "") or "-->" in ref.get("label", ""):
                errs.append("%s: book_ref label must be one line" % where)
            if not isinstance(ref.get("page"), int) or ref["page"] < 1:
                errs.append("%s: book_ref page must be 1 or more" % where)
            elif ref.get("end_page") is not None and ref["end_page"] < ref["page"]:
                errs.append("%s: book_ref end_page is before page" % where)
            if it["kind"] in ("output", "code"):
                errs.append("%s: a book reference takes a typed, term, choice or recall answer" % where)
        after = it.get("unlocks_after")
        if after is not None and (after == it["id"] or after not in ids):
            errs.append("%s: unlocks_after %r must name another item of the concept" % (where, after))
    return errs


# ---------------------------------------------------------------- running

def run(mode, code, setup="", tests=None, cpu=20):
    nonce = "B" + secrets.token_hex(8) + ":"
    req = {"nonce": nonce, "mode": mode, "code": code, "setup": setup,
           "tests": tests or [], "cpu_seconds": cpu, "memory_mb": 2048, "max_output": 200000}
    env = {
        "PATH": os.environ.get("PATH", ""), "LANG": "C.UTF-8", "LC_ALL": "C.UTF-8",
        "MPLBACKEND": "Agg", "OPENBLAS_NUM_THREADS": "1", "OMP_NUM_THREADS": "1",
        "PYTHONDONTWRITEBYTECODE": "1",
    }
    if os.environ.get("MPLCONFIGDIR"):
        env["MPLCONFIGDIR"] = os.environ["MPLCONFIGDIR"]
    with tempfile.TemporaryDirectory() as work:
        env["HOME"] = work
        p = subprocess.run([sys.executable, "-I", RUNNER], input=json.dumps(req).encode(),
                           capture_output=True, cwd=work, env=env, timeout=cpu + 30)
    text = p.stdout.decode("utf-8", "replace")
    i = text.find(nonce)
    if i < 0:
        raise RuntimeError("runner produced no result (exit %s): %s" % (p.returncode, p.stderr.decode()[-500:]))
    return json.loads(text[i + len(nonce):].splitlines()[0])


def check_item(cid, item):
    """Returns (item, errors)."""
    errs = []
    where = "%s#%s" % (cid, item["id"])
    # Mirrors internal/pytrack/bank.go: the server refuses to start on a bank
    # whose non-recall item has no explanation, so catch it here instead.
    if item["kind"] not in ("recall", "math") and not item.get("explain", "").strip():
        errs.append("%s: explain is required" % where)
    if item["kind"] == "output":
        res = run("run", item["code"])
        if res["error"]:
            errs.append("%s: snippet raised %s" % (where, res["error"].splitlines()[-1]))
            return item, errs
        got = res["stdout"].rstrip("\n")
        if not got.strip():
            errs.append("%s: snippet printed nothing" % where)
        if "expected" in item and item["expected"].rstrip("\n") != got:
            errs.append("%s: authored expected %r but the code prints %r" % (where, item["expected"], got))
        item["expected"] = got
    elif item["kind"] == "code":
        res = run("grade", item["solution"], item["setup"], item["tests"])
        if not res["passed"]:
            bad = [t["message"] for t in res["tests"] if not t["passed"]]
            errs.append("%s: reference solution fails: %s %s" % (where, res["error"] or res["setup_error"] or "", bad))
        starter_ok = item.pop("_starter_may_pass", False)
        res2 = run("grade", item["starter"] or "pass", item["setup"], item["tests"])
        if res2["passed"] and not starter_ok:
            errs.append("%s: the starter code already passes every test" % where)
        if res2.get("setup_error"):
            errs.append("%s: setup failed: %s" % (where, res2["setup_error"]))
    for field in ("prompt", "explain", "code", "starter", "solution", "correct"):
        if "$" in item.get(field, ""):
            errs.append("%s: '$' in %s would be rendered as math" % (where, field))
    return item, errs


def build_chapter(path):
    ns = runpy.run_path(path, init_globals=dict(HELPERS))
    chapter = ns["CHAPTER"]
    errors = []
    jobs = []
    with cf.ThreadPoolExecutor(max_workers=os.cpu_count() or 4) as pool:
        for c in chapter["concepts"]:
            errors.extend(check_gating(c["id"], c["items"]))
            for idx, item in enumerate(c["items"]):
                jobs.append((c, idx, pool.submit(check_item, c["id"], item)))
        for c, idx, fut in jobs:
            item, errs = fut.result()
            c["items"][idx] = item
            errors.extend(errs)
    return chapter, errors


def chapter_files(only=None, include_local=True):
    areas = [os.path.join(ROOT, "modules")] + ([os.path.join(LOCAL_ROOT, "modules-local")] if include_local else [])
    files = []
    for area in areas:
        files += sorted(glob.glob(os.path.join(area, "*", "src", "[0-9][0-9]_*.py")))
    if only:
        files = [f for f in files
                 if only in os.path.basename(f) or only == os.path.basename(module_dir(f))]
    return files


def rel(path):
    """A path for messages: relative to whichever root holds it."""
    for root in (ROOT, LOCAL_ROOT):
        r = os.path.relpath(path, root)
        if not r.startswith(".."):
            return r
    return path


def out_path(src):
    return os.path.join(module_dir(src), "bank", os.path.basename(src)[:-3] + ".json")


def module_dir(src):
    """modules/<id>/src/<file>.py -> modules/<id>"""
    return os.path.dirname(os.path.dirname(src))


def graph_path(src, chapter):
    """concepts/<domain>.json, or concepts/<graph_file>.json when the chapter
    names one: two chapters in one domain need separate files, or the second
    build overwrites the first's concepts."""
    return os.path.join(module_dir(src), "concepts", chapter.get("graph_file", chapter["domain"]) + ".json")


def render_graph(chapter):
    """The concept-graph file for a chapter, in the corpus's own format."""
    nodes = []
    for c in chapter["concepts"]:
        nodes.append({
            "id": c["id"],
            "label": c["_label"],
            "domain": chapter["domain"],
            "subdomain": c.get("_subdomain") or chapter["domain"] + "." + c["book"]["url"].rsplit("/", 1)[-1][:-5].replace("-", "_").replace(".", "_"),
            "grading_type": "bank",
            "prerequisites": c["_prereqs"],
            "mastery_threshold": {"streak": chapter.get("streak", 4), "avg_time_seconds": chapter.get("avg_time_seconds", 120.0)},
        })
    return json.dumps(nodes, indent=2, ensure_ascii=False) + "\n"


def render(chapter):
    bank = {
        "chapter": chapter["chapter"],
        "title": chapter["title"],
        "concepts": [{k: v for k, v in c.items() if not k.startswith("_")} for c in chapter["concepts"]],
    }
    return json.dumps(bank, indent=1, ensure_ascii=False) + "\n"


def main(argv):
    if len(argv) < 2 or argv[1] not in ("build", "check"):
        print(__doc__)
        return 2
    mode = argv[1]
    args = [a for a in argv[2:] if not a.startswith("--")]
    only = args[0] if args else None
    # --public: only committed modules (the Docker image's self-test, which
    # never sees your modules-local/).
    include_local = "--public" not in argv
    failed = False
    files = chapter_files(only, include_local)
    if not files:
        print("no authoring files found" + (" matching %r" % only if only else ""))
        return 1
    for src in files:
        chapter, errors = build_chapter(src)
        n = sum(len(c["items"]) for c in chapter["concepts"])
        for e in errors:
            print("ERROR", e)
        failed |= bool(errors)
        text = render(chapter)
        graph = render_graph(chapter)
        dst = out_path(src)
        gdst = graph_path(src, chapter)
        os.makedirs(os.path.dirname(dst), exist_ok=True)
        os.makedirs(os.path.dirname(gdst), exist_ok=True)
        if mode == "build":
            if not errors:
                with open(dst, "w", encoding="utf-8") as f:
                    f.write(text)
                with open(gdst, "w", encoding="utf-8") as f:
                    f.write(graph)
            print("%s: %d concepts, %d questions%s" % (rel(src), len(chapter["concepts"]), n,
                                                      "" if not errors else " (not written: errors)"))
        else:
            try:
                current = open(dst, encoding="utf-8").read()
            except FileNotFoundError:
                current = ""
            try:
                gcurrent = open(gdst, encoding="utf-8").read()
            except FileNotFoundError:
                gcurrent = ""
            if current != text or gcurrent != graph:
                print("STALE %s: run `python3 scripts/python_bank.py build %s`" % (rel(dst), os.path.basename(module_dir(src))))
                failed = True
            else:
                print("%s: ok (%d questions)" % (rel(src), n))
    if mode == "build" and not failed:
        # The browser bundles its own copy of the concept graph; refresh it so
        # new concepts get labels on /learn and appear in the topic browser.
        import shutil
        if shutil.which("go"):
            subprocess.run(["go", "run", "./scripts/merge_concepts"], cwd=ROOT, check=False)
        else:
            print("now run: go run ./scripts/merge_concepts   (refreshes web/next-app/data/concepts.json)")
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
