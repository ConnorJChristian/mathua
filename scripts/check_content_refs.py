#!/usr/bin/env python3
"""Reference integrity for module content (fork change).

Every pointer from a module's content into a file, a page, a link or a
topic is checked, for the public modules (modules/) and your own
(modules-local/):

  readings.json   the file exists, page and end_page are inside the PDF,
                  page <= end_page, URLs are https, the topic exists
  books.json      the file exists; each contents line's pages are in range
  notes.json      the HTML/ipynb page exists or the link is https; the
  videos.json     topic or course exists
  bank/*.json     book_ref files and pages; <img src> and ![](..) images
                  that point into a module's files/; intro-less concepts
  courses.json    every planned topic exists in the graph and has a reading,
                  lecture notes or a bank intro; the course's texts and
                  topic_texts name books that exist

Files are checked against the filesystem. Because the PDFs may live on
another machine (modules-local/ is git-ignored), a listing of that machine's
files and a table of PDF page counts can stand in for it:

    python3 scripts/check_content_refs.py \
        --files-list pc_files.tsv --pdf-pages pdf_pages.tsv

pc_files.tsv is "<path relative to the repo>\t<size>" per line and
pdf_pages.tsv is "<file name>\t<pages>". With a listing, a file counts as
present when it is in the listing OR on disk (the listing may leave out
images, say); with --listing-only, only the listing counts. Page counts
come from the table, then from the PDF on disk (pypdf), then from the
module's tools/outlines/*.json ("pages"); a PDF with no known count skips
the range check and is reported as unknown.

Exit status 1 when anything is broken, 0 when only warnings remain.
"""
import argparse
import glob
import json
import os
import re
import sys
from urllib.parse import unquote, urlparse

IMG_HTML = re.compile(r"<img[^>]*\ssrc=[\"']([^\"']+)[\"']", re.I)
IMG_MD = re.compile(r"!\[[^\]]*\]\(([^)\s]+)")
CODE = re.compile(r"```.*?```|`[^`\n]*`", re.S)
MODULE_FILE_URL = re.compile(r"^/api/modules/([^/]+)/files/(.+)$")


class Report:
    def __init__(self):
        self.errors = []
        self.warnings = []
        self.counts = {}

    def error(self, where, msg):
        self.errors.append("%s: %s" % (where, msg))

    def warn(self, where, msg):
        self.warnings.append("%s: %s" % (where, msg))

    def count(self, key, n=1):
        self.counts[key] = self.counts.get(key, 0) + n


class Files:
    """Which files exist, and how many pages each PDF has."""

    def __init__(self, root, listing=None, pdf_pages=None, listing_only=False):
        self.root = root
        self.listing = listing  # set of repo-relative paths, or None
        self.listing_only = listing_only and listing is not None
        self.pages = dict(pdf_pages or {})  # file name -> pages
        self._page_cache = {}

    def exists(self, rel):
        rel = rel.replace("\\", "/")
        if self.listing is not None and rel in self.listing:
            return True
        if self.listing_only:
            return False
        p = os.path.join(self.root, rel)
        return os.path.isfile(p)

    def pdf_pages(self, module_dir, rel):
        """Page count of a module file, or None when unknown."""
        name = os.path.basename(rel)
        if name in self.pages:
            return self.pages[name]
        key = os.path.join(module_dir, rel)
        if key in self._page_cache:
            return self._page_cache[key]
        n = None
        p = os.path.join(self.root, module_dir, "files", rel)
        if os.path.isfile(p):
            n = count_pdf_pages(p)
        if n is None:
            n = outline_pages(os.path.join(self.root, module_dir), rel)
        self._page_cache[key] = n
        return n


def count_pdf_pages(path):
    try:
        import pypdf  # noqa: WPS433
    except ImportError:
        return None
    try:
        with open(path, "rb") as fh:
            return len(pypdf.PdfReader(fh).pages)
    except Exception:  # a broken PDF is reported by the caller as unknown
        return None


OUTLINE_ROW = re.compile(r'\(\s*"([^"]+)"\s*,\s*"([^"]+\.pdf)"\s*,\s*"([^"]+)"')


def outline_pages(module_dir, rel):
    """tools/outlines/<name>.json carries {"pages": N} for each book whose
    bookmarks were extracted. tools/texts.py names the outline for each file
    in its BOOKS table (module, file, outline, ...); any module's texts.py
    may describe the file, since one tool writes several modules' books."""
    stem = os.path.splitext(os.path.basename(rel))[0]
    names = {stem}
    area = os.path.dirname(module_dir)
    for tp in glob.glob(os.path.join(area, "*", "tools", "texts.py")):
        try:
            src = open(tp, encoding="utf-8").read()
        except OSError:
            continue
        for _, f, outline in OUTLINE_ROW.findall(src):
            if f == os.path.basename(rel):
                names.add(outline)
    for name in names:
        for cand in glob.glob(os.path.join(area, "*", "tools", "outlines", name + ".json")):
            try:
                n = json.load(open(cand, encoding="utf-8")).get("pages")
            except (OSError, ValueError):
                continue
            if isinstance(n, int) and n > 0:
                return n
    return None


def load_json(path, rep):
    try:
        with open(path, encoding="utf-8") as fh:
            return json.load(fh)
    except ValueError as e:
        rep.error(path, "invalid JSON: %s" % e)
        return None


def discover_modules(root, public_only):
    """[(module id, module dir relative to root, manifest)]"""
    disabled = set()
    cfg = os.path.join(root, "modules.local.json")
    if os.path.isfile(cfg):
        try:
            disabled = set(json.load(open(cfg)).get("disabled") or [])
        except ValueError:
            pass
    out = []
    areas = ["modules"] if public_only else ["modules", "modules-local"]
    for area in areas:
        base = os.path.join(root, area)
        if not os.path.isdir(base):
            continue
        for name in sorted(os.listdir(base)):
            if name.startswith((".", "_")):
                continue
            mdir = os.path.join(base, name)
            manifest = os.path.join(mdir, "module.json")
            if not os.path.isfile(manifest):
                continue
            try:
                man = json.load(open(manifest, encoding="utf-8"))
            except ValueError:
                man = {}
            mid = man.get("id") or name
            if mid in disabled:
                continue
            out.append((mid, os.path.join(area, name), man))
    return out


def concept_ids(root, modules):
    ids = set()
    dirs = [os.path.join(root, "data", "concepts")] + [os.path.join(root, d, "concepts") for _, d, _ in modules]
    for d in dirs:
        for f in glob.glob(os.path.join(d, "*.json")):
            try:
                data = json.load(open(f, encoding="utf-8"))
            except ValueError:
                continue
            if isinstance(data, list):
                for c in data:
                    if isinstance(c, dict) and c.get("id"):
                        ids.add(c["id"])
    return ids


def readings_list(v):
    return v if isinstance(v, list) else [v]


def check_url(where, url, rep, https_only=True):
    u = urlparse(url)
    if https_only and u.scheme != "https":
        rep.error(where, "url is not https: %s" % url)
        return False
    if u.scheme not in ("http", "https") or not u.netloc:
        rep.error(where, "malformed url: %s" % url)
        return False
    if " " in url:
        rep.error(where, "url has a space: %s" % url)
        return False
    return True


def check_pages(where, module_dir, rel, page, end_page, files, rep):
    if page is not None and (not isinstance(page, int) or page < 1):
        rep.error(where, "page %r is not a positive integer" % (page,))
        return
    if end_page is not None and (not isinstance(end_page, int) or end_page < 1):
        rep.error(where, "end_page %r is not a positive integer" % (end_page,))
        return
    if page and end_page and end_page < page:
        rep.error(where, "end_page %d is before page %d" % (end_page, page))
    if not rel.lower().endswith(".pdf"):
        if page or end_page:
            rep.warn(where, "page numbers on a non-PDF file %s" % rel)
        return
    n = files.pdf_pages(module_dir, rel)
    if n is None:
        rep.count("pdf pages unknown")
        rep.unknown_pdfs.add(os.path.join(module_dir, "files", rel))
        return
    for label, p in (("page", page), ("end_page", end_page)):
        if p and p > n:
            rep.error(where, "%s %d is past the end of %s (%d pages)" % (label, p, rel, n))


def check_file(where, module_dir, rel, files, rep, kind="file"):
    rel = rel.replace("\\", "/")
    if rel.startswith("/") or rel.startswith("..") or "/../" in rel:
        rep.error(where, "%s %r leaves files/" % (kind, rel))
        return False
    if not files.exists(os.path.join(module_dir, "files", rel)):
        rep.error(where, "%s missing: %s/files/%s" % (kind, module_dir, rel))
        rep.missing_files.add("%s/files/%s" % (module_dir, rel))
        return False
    return True


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--root", default=os.environ.get("MATHUA_ROOT") or os.getcwd())
    ap.add_argument("--files-list", help="tsv of files on the machine that holds modules-local/")
    ap.add_argument("--pdf-pages", help="tsv of '<pdf file name>\\t<page count>'")
    ap.add_argument("--listing-only", action="store_true", help="a file exists only if the listing has it")
    ap.add_argument("--public-only", action="store_true", help="check modules/ only (what CI sees)")
    ap.add_argument("--quiet", action="store_true", help="summary and errors only")
    args = ap.parse_args()
    root = os.path.abspath(args.root)
    rep = Report()
    rep.missing_files = set()
    rep.unknown_pdfs = set()

    listing = None
    if args.files_list:
        listing = set()
        with open(args.files_list, encoding="utf-8") as fh:
            for line in fh:
                p = line.rstrip("\n").split("\t")[0].strip().replace("\\", "/")
                if p:
                    listing.add(p)
    pdf_pages = {}
    if args.pdf_pages:
        with open(args.pdf_pages, encoding="utf-8") as fh:
            for line in fh:
                parts = line.rstrip("\n").split("\t")
                if len(parts) >= 2 and parts[1].strip().isdigit():
                    pdf_pages[parts[0].strip()] = int(parts[1])
    files = Files(root, listing, pdf_pages, args.listing_only)

    modules = discover_modules(root, args.public_only)
    ids = concept_ids(root, modules)
    rep.count("modules", len(modules))
    rep.count("concepts", len(ids))

    # Books and readings by key "<module>/<file>", for course texts.
    book_keys = set()
    readings_by_topic = {}   # topic -> number of usable readings
    notes_by_id = {}
    intro_topics = set()
    course_ids = set()
    bank_images = []

    for mid, mdir, man in modules:
        base = os.path.join(root, mdir)
        # readings.json
        rpath = os.path.join(base, "readings.json")
        if os.path.isfile(rpath):
            data = load_json(rpath, rep)
            for cid, v in (data or {}).items():
                if cid not in ids:
                    rep.error("%s/readings.json" % mdir, "reading for unknown topic %s" % cid)
                for i, r in enumerate(readings_list(v)):
                    where = "%s/readings.json %s[%d]" % (mdir, cid, i)
                    rep.count("readings")
                    if not isinstance(r, dict):
                        rep.error(where, "not an object")
                        continue
                    if not r.get("source"):
                        rep.error(where, "no source")
                    if r.get("file"):
                        rep.count("readings: file")
                        if check_file(where, mdir, r["file"], files, rep):
                            check_pages(where, mdir, r["file"], r.get("page"), r.get("end_page"), files, rep)
                        book_keys.add(mid + "/" + r["file"])
                        readings_by_topic[cid] = readings_by_topic.get(cid, 0) + 1
                    elif r.get("url"):
                        rep.count("readings: url")
                        if check_url(where, r["url"], rep):
                            readings_by_topic[cid] = readings_by_topic.get(cid, 0) + 1
                    else:
                        rep.count("readings: text only")
                        # A source with neither file nor link is shown as
                        # nothing by the lesson (server readingOptions skips
                        # it) and does not count as a reading.
                        rep.warn(where, "reading has neither file nor url (the lesson cannot open it): %s" % r.get("source"))
        # books.json
        bpath = os.path.join(base, "books.json")
        if os.path.isfile(bpath):
            data = load_json(bpath, rep)
            for f, b in (data or {}).items():
                where = "%s/books.json %s" % (mdir, f)
                rep.count("books")
                book_keys.add(mid + "/" + f)
                if not check_file(where, mdir, f, files, rep, "book"):
                    continue
                if not b.get("title"):
                    rep.warn(where, "book has no title")
                for j, t in enumerate(b.get("toc") or []):
                    rep.count("toc entries")
                    check_pages("%s toc[%d] %s" % (where, j, t.get("title", "")), mdir, f, t.get("page"), t.get("end_page"), files, rep)
        # notes.json / videos.json
        for name, what in (("notes.json", "note"), ("videos.json", "video")):
            npath = os.path.join(base, name)
            if not os.path.isfile(npath):
                continue
            data = load_json(npath, rep)
            for key, lst in (data or {}).items():
                for i, n in enumerate(lst or []):
                    where = "%s/%s %s[%d]" % (mdir, name, key, i)
                    rep.count(what + "s")
                    if n.get("file"):
                        if check_file(where, mdir, n["file"], files, rep, what):
                            if what == "note":
                                notes_by_id[key] = notes_by_id.get(key, 0) + 1
                            ext = os.path.splitext(n["file"])[1].lower()
                            if what == "note" and ext not in (".html", ".htm", ".ipynb", ".pdf", ".md"):
                                rep.warn(where, "note file type %s may not render in the lesson" % ext)
                    elif n.get("url"):
                        if check_url(where, n["url"], rep) and what == "note":
                            notes_by_id[key] = notes_by_id.get(key, 0) + 1
                    else:
                        rep.error(where, "%s has neither file nor url" % what)
                    if key not in ids and not key.startswith("plan.") and key not in course_ids:
                        # Course ids are known only after courses.json; checked below.
                        pass
                    rep.note_keys = getattr(rep, "note_keys", [])
                    rep.note_keys.append((where, key))
        # bank/*.json (bank/skip.json lists items the server never serves)
        skipped = set()
        if os.path.isfile(os.path.join(base, "bank", "skip.json")):
            skipped = set((load_json(os.path.join(base, "bank", "skip.json"), rep) or {}).keys())
        for bf in sorted(glob.glob(os.path.join(base, "bank", "*.json"))):
            if os.path.basename(bf) == "skip.json":
                continue
            data = load_json(bf, rep)
            if not isinstance(data, dict):
                continue
            bwhere = os.path.relpath(bf, root)
            for c in data.get("concepts") or []:
                cid = c.get("id")
                if cid and (c.get("intro") or "").strip():
                    intro_topics.add(cid)
                book = c.get("book") or {}
                if book.get("url"):
                    rep.count("bank book urls")
                    check_url("%s %s book" % (bwhere, cid), book["url"], rep, https_only=False)
                texts = [c.get("intro") or ""]
                for it in c.get("items") or []:
                    if it.get("id") in skipped:
                        continue
                    iw = "%s %s/%s" % (bwhere, cid, it.get("id"))
                    br = it.get("book_ref")
                    if br:
                        rep.count("book_refs")
                        bm = br.get("module") or mid
                        bdir = next((d for m, d, _ in modules if m == bm), None)
                        if bdir is None:
                            rep.error(iw, "book_ref module %s is not an enabled module" % bm)
                        elif not br.get("file"):
                            rep.error(iw, "book_ref has no file")
                        elif check_file(iw, bdir, br["file"], files, rep, "book_ref"):
                            check_pages(iw, bdir, br["file"], br.get("page"), br.get("end_page"), files, rep)
                    for field in ("prompt", "explain", "back", "solution", "correct"):
                        if isinstance(it.get(field), str):
                            texts.append(it[field])
                    for w in it.get("wrong") or []:
                        if isinstance(w, str):
                            texts.append(w)
                    for part in it.get("parts") or []:
                        for field in ("prompt", "answer", "explain"):
                            if isinstance(part.get(field), str):
                                texts.append(part[field])
                for t in texts:
                    # An image written inside code (a Markdown lesson that
                    # shows the syntax, say) is text, not a reference.
                    t = CODE.sub("", t)
                    for src in IMG_HTML.findall(t) + IMG_MD.findall(t):
                        bank_images.append((bwhere, cid, src))
        # courses.json
        cpath = os.path.join(base, "courses.json")
        if os.path.isfile(cpath):
            data = load_json(cpath, rep)
            for co in (data or []):
                course_ids.add(co.get("id"))
                rep.courses = getattr(rep, "courses", [])
                rep.courses.append((mdir, co))

    # Upstream lessons (data/lessons/lessons.json) teach the base graph's
    # topics; a course topic with one has something to read.
    lesson_topics = set()
    lmap = os.path.join(root, "data", "lessons", "lessons.json")
    if os.path.isfile(lmap):
        for m in load_json(lmap, rep) or []:
            src = os.path.join(root, "data", "lessons", m.get("source", ""))
            rep.count("lesson mappings")
            if not os.path.isfile(src):
                rep.error("data/lessons/lessons.json %s" % m.get("concept_id"), "lesson file missing: %s" % m.get("source"))
            elif m.get("concept_id") not in ids:
                rep.warn("data/lessons/lessons.json", "lesson for unknown topic %s" % m.get("concept_id"))
            else:
                lesson_topics.add(m["concept_id"])

    # Base courses (data/courses.json) count for notes/videos keys too.
    base_courses = os.path.join(root, "data", "courses.json")
    if os.path.isfile(base_courses):
        data = load_json(base_courses, rep)
        for co in (data or []):
            course_ids.add(co.get("id"))

    for where, key in getattr(rep, "note_keys", []):
        if key not in ids and key not in course_ids:
            rep.error(where, "attached to unknown topic or course %s" % key)

    # Images in bank text.
    for bwhere, cid, src in bank_images:
        rep.count("bank images")
        where = "%s %s image" % (bwhere, cid)
        m = MODULE_FILE_URL.match(src)
        if m:
            bm, rel = m.group(1), unquote(m.group(2))
            bdir = next((d for mm, d, _ in modules if mm == bm), None)
            if bdir is None:
                rep.error(where, "image in module %s, which is not enabled: %s" % (bm, src))
            else:
                check_file(where, bdir, rel, files, rep, "image")
        elif src.startswith(("http://", "https://")):
            check_url(where, src, rep, https_only=False)
        elif src.startswith("/"):
            if not os.path.isfile(os.path.join(root, "web", "next-app", "public", src.lstrip("/"))):
                rep.error(where, "image not under web/next-app/public: %s" % src)
        else:
            rep.error(where, "relative image path cannot be served: %s" % src)

    # Courses: every planned topic exists and has something to read.
    for mdir, co in getattr(rep, "courses", []):
        cw = "%s/courses.json %s" % (mdir, co.get("id"))
        rep.count("courses")
        plan = co.get("plan") or co.get("targets") or []
        module_topics = [t for m in (co.get("modules") or []) for t in (m.get("topics") or [])]
        for t in co.get("targets") or []:
            if t not in ids:
                rep.error(cw, "target %s is not in the graph" % t)
        seen = set()
        for t in plan:
            rep.count("planned topics")
            if t not in ids:
                rep.error(cw, "planned topic %s is not in the graph" % t)
                continue
            if t in seen:
                rep.error(cw, "planned topic %s listed twice" % t)
            seen.add(t)
            if not (readings_by_topic.get(t) or notes_by_id.get(t) or t in intro_topics):
                if t in lesson_topics:
                    rep.count("planned topics with only the upstream lesson")
                else:
                    rep.error(cw, "topic %s has no reading, lecture note, bank intro or lesson" % t)
        for t in module_topics:
            if t not in seen:
                rep.error(cw, "module topic %s is not in the plan" % t)
        if co.get("modules") and set(module_topics) != seen:
            missing = [t for t in plan if t not in set(module_topics)]
            if missing:
                rep.warn(cw, "%d planned topics are in no module: %s" % (len(missing), ", ".join(missing[:5])))
        for tx in co.get("texts") or []:
            if tx.startswith(("web:", "notes:", "link:")):
                continue
            if tx not in book_keys:
                rep.error(cw, "text %s is not a known book or reading file" % tx)
            else:
                mod, _, rel = tx.partition("/")
                bdir = next((d for m, d, _ in modules if m == mod), None)
                if bdir and not files.exists(os.path.join(bdir, "files", rel)):
                    rep.error(cw, "text file missing: %s" % tx)
        for t, txs in (co.get("topic_texts") or {}).items():
            if t not in seen:
                rep.warn(cw, "topic_texts names %s, which is not in the plan" % t)
            for tx in txs:
                if not tx.startswith(("web:", "notes:", "link:")) and tx not in book_keys:
                    rep.error(cw, "topic_texts %s: %s is not a known book" % (t, tx))

    # Report.
    print("checked %s" % ", ".join("%s %s" % (v, k) for k, v in sorted(rep.counts.items())))
    if rep.unknown_pdfs and not args.quiet:
        print("page count unknown (range not checked): " + ", ".join(sorted(rep.unknown_pdfs)))
    if rep.missing_files:
        print("missing files (%d):" % len(rep.missing_files))
        for f in sorted(rep.missing_files):
            print("  " + f)
    if rep.warnings and not args.quiet:
        print("warnings (%d):" % len(rep.warnings))
        for w in rep.warnings:
            print("  " + w)
    if rep.errors:
        print("errors (%d):" % len(rep.errors))
        for e in rep.errors:
            print("  " + e)
        return 1
    print("ok")
    return 0


if __name__ == "__main__":
    sys.exit(main())
