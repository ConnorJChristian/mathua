"""Tests for check_content_refs.py: python3 -m unittest scripts/check_content_refs_test.py"""
import io
import json
import os
import sys
import tempfile
import unittest
from contextlib import redirect_stdout

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import check_content_refs as ccr  # noqa: E402


def write(path, data):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8") as fh:
        if isinstance(data, (dict, list)):
            json.dump(data, fh)
        else:
            fh.write(data)


class CheckerTest(unittest.TestCase):
    def setUp(self):
        self.root = tempfile.mkdtemp()
        write(os.path.join(self.root, "data", "concepts", "x.json"), [{"id": "t.one"}, {"id": "t.two"}])
        write(os.path.join(self.root, "data", "lessons", "lessons.json"), [])
        m = os.path.join(self.root, "modules", "m")
        write(os.path.join(m, "module.json"), {"id": "m", "name": "M"})
        write(os.path.join(m, "files", "book.pdf"), "%PDF-1.4 stub")
        write(os.path.join(m, "files", "notes", "p1.html"), "<p>x</p>")
        self.m = m

    def run_checker(self, *extra):
        argv = ["check_content_refs.py", "--root", self.root, "--pdf-pages", self.pages_file()] + list(extra)
        out = io.StringIO()
        old = sys.argv
        sys.argv = argv
        try:
            with redirect_stdout(out):
                code = ccr.main()
        finally:
            sys.argv = old
        return code, out.getvalue()

    def pages_file(self):
        p = os.path.join(self.root, "pages.tsv")
        if not os.path.exists(p):
            write(p, "book.pdf\t100\n")
        return p

    def test_clean_module_passes(self):
        write(os.path.join(self.m, "readings.json"), {"t.one": {"source": "Ch 1", "file": "book.pdf", "page": 3, "end_page": 9}})
        write(os.path.join(self.m, "notes.json"), {"t.two": [{"title": "n", "file": "notes/p1.html"}]})
        write(os.path.join(self.m, "courses.json"), [{"id": "c", "name": "C", "targets": ["t.one", "t.two"], "plan": ["t.one", "t.two"], "texts": ["m/book.pdf"]}])
        code, out = self.run_checker()
        self.assertEqual(code, 0, out)
        self.assertIn("ok", out)

    def test_page_past_end_and_reversed_range(self):
        write(os.path.join(self.m, "readings.json"), {
            "t.one": {"source": "Ch 1", "file": "book.pdf", "page": 101},
            "t.two": {"source": "Ch 2", "file": "book.pdf", "page": 20, "end_page": 10},
        })
        code, out = self.run_checker()
        self.assertEqual(code, 1)
        self.assertIn("page 101 is past the end of book.pdf (100 pages)", out)
        self.assertIn("end_page 10 is before page 20", out)

    def test_missing_file_unknown_topic_and_bad_url(self):
        write(os.path.join(self.m, "readings.json"), {
            "t.one": {"source": "Ch 1", "file": "gone.pdf", "page": 1},
            "t.nine": {"source": "web", "url": "http://insecure.example/x"},
        })
        code, out = self.run_checker()
        self.assertEqual(code, 1)
        self.assertIn("file missing: modules/m/files/gone.pdf", out)
        self.assertIn("reading for unknown topic t.nine", out)
        self.assertIn("url is not https", out)

    def test_listing_stands_in_for_the_other_machine(self):
        # The PDF is not here but the listing says it is on the owner's PC.
        write(os.path.join(self.m, "readings.json"), {"t.one": {"source": "Ch 1", "file": "hoy.pdf", "page": 5, "end_page": 6}})
        listing = os.path.join(self.root, "files.tsv")
        write(listing, "modules/m/files/hoy.pdf\t123\n")
        write(self.pages_file(), "book.pdf\t100\nhoy.pdf\t5\n")
        code, out = self.run_checker("--files-list", listing)
        self.assertEqual(code, 1)
        self.assertNotIn("file missing", out)
        self.assertIn("end_page 6 is past the end of hoy.pdf (5 pages)", out)

    def test_course_topic_needs_something_to_read(self):
        write(os.path.join(self.m, "courses.json"), [{"id": "c", "name": "C", "targets": ["t.one"], "plan": ["t.one", "t.zzz"]}])
        code, out = self.run_checker()
        self.assertEqual(code, 1)
        self.assertIn("planned topic t.zzz is not in the graph", out)
        self.assertIn("topic t.one has no reading, lecture note, bank intro or lesson", out)

    def test_bank_images_and_book_refs(self):
        write(os.path.join(self.m, "bank", "b.json"), {"concepts": [{
            "id": "t.one", "intro": "see <img src=\"/api/modules/m/files/img%20one.png\">",
            "items": [
                {"id": "i1", "prompt": "![](figures\\1.jpg)", "book_ref": {"file": "book.pdf", "page": 400, "label": "x"}},
                {"id": "i2", "prompt": "`![](in-code.png)` is markdown"},
                {"id": "skipped", "prompt": "![](lost.png)"},
            ]}]})
        write(os.path.join(self.m, "bank", "skip.json"), {"skipped": "quarantined"})
        write(os.path.join(self.m, "files", "img one.png"), "png")
        code, out = self.run_checker()
        self.assertEqual(code, 1)
        self.assertIn("relative image path cannot be served: figures\\1.jpg", out)
        self.assertIn("page 400 is past the end of book.pdf", out)
        self.assertNotIn("img one.png", out.split("errors")[-1])
        self.assertNotIn("in-code.png", out)
        self.assertNotIn("lost.png", out)


if __name__ == "__main__":
    unittest.main()
