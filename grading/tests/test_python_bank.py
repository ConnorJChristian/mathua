"""scripts/python_bank.py: the authoring helpers and the build-time runner.

Run with: python3 -m pytest grading/tests/test_python_bank.py
"""
import importlib.util
import os
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent.parent
spec = importlib.util.spec_from_file_location("python_bank", ROOT / "scripts" / "python_bank.py")
pb = importlib.util.module_from_spec(spec)
spec.loader.exec_module(pb)


def test_out_setup_is_build_time_only():
    """`setup` on an output item runs before the snippet when the bank is
    built, and is not stored: the learner predicts the snippet's output."""
    item = pb.out("o1", "What does this print?", "print(n + 1)", "n is 41.", setup="n = 41")
    assert item["_setup"] == "n = 41"
    checked, errs = pb.check_item("c", item)
    assert errs == []
    assert checked["expected"] == "42"
    assert "_setup" not in checked
    rendered = pb.render({"chapter": "x", "title": "x", "concepts": [{"id": "c", "items": [checked]}]})
    assert "_setup" not in rendered and '"expected": "42"' in rendered


def test_out_without_setup_still_builds():
    item = pb.out("o1", "What does this print?", "print(2 * 3)", "Six.")
    checked, errs = pb.check_item("c", item)
    assert errs == [] and checked["expected"] == "6"


def test_run_puts_temp_files_in_the_working_folder():
    """The server (internal/pytrack/runner.go) sets TMPDIR to the working
    folder, so libraries that write temporary files (openpyxl saves one per
    sheet) pass the audit hook. The build-time runner must match, or a bank
    that builds here fails in the app, and the other way round."""
    res = pb.run("run", """
import os, tempfile
print(os.path.realpath(tempfile.gettempdir()) == os.path.realpath(os.getcwd()))
with tempfile.NamedTemporaryFile(mode="w", suffix=".xml") as f:
    f.write("ok")
print("written")
""")
    assert res["error"] is None, res["error"]
    assert res["stdout"].split() == ["True", "written"]


def test_setup_failure_is_reported_for_code_items():
    item = pb.task("c1", "Set x.", "x = 1", [pb.check("x", "check(get('x') == 1)")], "x is 1.", setup="raise ValueError('boom')")
    _, errs = pb.check_item("c", item)
    assert any("setup failed" in e for e in errs)
