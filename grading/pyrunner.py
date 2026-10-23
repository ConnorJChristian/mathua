#!/usr/bin/env python3
"""Run one learner Python submission and, optionally, its hidden tests.

The Go server (internal/pytrack/runner.go) starts one process per submission:

    python3 -I grading/pyrunner.py   < request.json

and reads one line from stdout that starts with the request's nonce. Everything
the learner's code prints is captured in memory, never written to the real
stdout, so learner output cannot be mistaken for the verdict line.

Request (JSON object on stdin):
    nonce         random string chosen by the server; prefixes the result line
    mode          "run" (execute and report output) or "grade" (also run tests)
    code          the learner's code
    setup         hidden code executed before the learner's code (fixtures)
    tests         [{"name": str, "code": str, "hint": str}]  (grade mode)
    cpu_seconds   CPU time limit for this process
    memory_mb     address-space limit
    max_output    characters of captured stdout to keep
    figures       true to return open matplotlib figures as base64 PNG
    deny_paths    absolute path prefixes the learner's code may not open

Result (one line on stdout: nonce + JSON):
    stdout, stdout_truncated, error, tests[{name, passed, message}], passed,
    figures[]

Limits are layered. The server runs this process as an unprivileged user when
it is itself root (the Docker image), in its own process group, with a wall
clock deadline. This script adds rlimits and an audit hook that refuses
network access, process creation and writes outside the working directory.
The audit hook is a guard against accidents and casual misuse in a single-user
study tool, not a security boundary against a determined attacker: ctypes is
left open because NumPy, SciPy and scikit-learn load native libraries through it.
"""

import base64
import builtins
import io
import json
import os
import sys
import traceback

USER_FILENAME = "<your code>"
SETUP_FILENAME = "<setup>"


class Fail(AssertionError):
    """Raised by test helpers; the message is shown to the learner."""


def _set_limits(cpu_seconds, memory_mb):
    try:
        import resource
    except ImportError:  # not POSIX; the server's wall clock still applies
        return
    def setlim(which, soft, hard=None):
        try:
            resource.setrlimit(which, (soft, soft if hard is None else hard))
        except (ValueError, OSError):
            pass
    cpu = max(1, int(cpu_seconds))
    setlim(resource.RLIMIT_CPU, cpu, cpu + 1)
    if memory_mb:
        setlim(resource.RLIMIT_AS, int(memory_mb) * 1024 * 1024)
    setlim(resource.RLIMIT_FSIZE, 16 * 1024 * 1024)
    setlim(resource.RLIMIT_CORE, 0)
    if hasattr(resource, "RLIMIT_NPROC") and os.getuid() != 0 and os.environ.get("PYRUNNER_SANDBOX_USER") == "1":
        # Only when running as the dedicated sandbox user: the limit is per
        # user, so applying it to a developer's own account would count every
        # process they have open.
        setlim(resource.RLIMIT_NPROC, 64)


def _install_audit_hook(workdir, deny_paths):
    workdir = os.path.realpath(workdir)
    deny = [os.path.realpath(p) for p in deny_paths if p]

    blocked_events = {
        "os.system", "os.exec", "os.posix_spawn", "os.spawn", "os.fork",
        "os.forkpty", "os.kill", "os.killpg", "subprocess.Popen",
        "pty.spawn", "socket.connect", "socket.bind", "socket.getaddrinfo",
        "socket.gethostbyname", "socket.sendto", "socket.sendmsg",
        "urllib.Request", "webbrowser.open", "os.chroot", "os.setuid",
        "sys.addaudithook",
    }
    write_events = {
        "os.remove", "os.rename", "os.rmdir", "os.mkdir", "os.chmod",
        "os.chown", "os.truncate", "os.link", "os.symlink", "os.utime",
        "shutil.rmtree", "shutil.move", "shutil.copyfile",
    }

    def inside(path, root):
        return path == root or path.startswith(root + os.sep)

    def resolve(p):
        if isinstance(p, int):
            return None
        try:
            p = os.fsdecode(p)
        except TypeError:
            return None
        return os.path.realpath(os.path.join(workdir, p))

    def check_denied(path):
        for d in deny:
            if inside(path, d):
                raise PermissionError("This path is not available to practice code.")

    armed = [False]

    def hook(event, args):
        if not armed[0]:
            return
        if event in blocked_events:
            raise PermissionError(
                "Practice code cannot use %s (no network, no subprocesses)." % event)
        if event == "open":
            path = resolve(args[0])
            if path is None:
                return
            check_denied(path)
            mode = args[1] if len(args) > 1 and isinstance(args[1], str) else "r"
            flags = args[2] if len(args) > 2 and isinstance(args[2], int) else 0
            writing = any(c in mode for c in "wax+") or (flags & (os.O_WRONLY | os.O_RDWR | os.O_CREAT | os.O_TRUNC | os.O_APPEND))
            if writing and not inside(path, workdir):
                raise PermissionError("Practice code can only write files in its own working folder.")
            return
        if event in ("os.listdir", "os.scandir"):
            path = resolve(args[0] if args and args[0] is not None else ".")
            if path is not None:
                check_denied(path)
            return
        if event in write_events:
            for a in args:
                if isinstance(a, (str, bytes, os.PathLike)):
                    path = resolve(a)
                    if path is not None and not inside(path, workdir):
                        raise PermissionError("Practice code can only change files in its own working folder.")

    sys.addaudithook(hook)
    return armed


class _CappedStringIO(io.StringIO):
    def __init__(self, cap):
        super().__init__()
        self.cap = cap
        self.truncated = False

    def write(self, s):
        if not isinstance(s, str):
            raise TypeError("write() argument must be str, not %s" % type(s).__name__)
        room = self.cap - self.tell()
        if room <= 0:
            self.truncated = True
            return len(s)
        if len(s) > room:
            self.truncated = True
            super().write(s[:room])
            return len(s)
        return super().write(s)


def _format_user_error(exc):
    """A traceback that shows the learner's own lines and the error, not the harness."""
    te = traceback.TracebackException.from_exception(exc)
    frames = [f for f in te.stack if f.filename == USER_FILENAME]
    lines = []
    if frames:
        lines.append("Traceback (most recent call last):")
        for f in frames:
            lines.append('  line %d%s' % (f.lineno or 0, (", in " + f.name) if f.name and f.name != "<module>" else ""))
            if f.line:
                lines.append("    " + f.line.strip())
    lines.extend(l.rstrip("\n") for l in te.format_exception_only())
    return "\n".join(lines)


def _helpers(ns, source):
    """Names available to hidden tests. Tests see these plus the learner's names."""

    def get(name):
        if name not in ns:
            raise Fail("Your code does not define `%s`." % name)
        return ns[name]

    def check(cond, msg="A hidden check failed."):
        if not cond:
            raise Fail(msg)

    def close(a, b, rtol=1e-6, atol=1e-9):
        import numpy as np
        try:
            return bool(np.allclose(np.asarray(a, dtype=float), np.asarray(b, dtype=float), rtol=rtol, atol=atol))
        except (TypeError, ValueError):
            return False

    def same_array(a, b):
        import numpy as np
        a = np.asarray(a)
        b = np.asarray(b)
        return a.shape == b.shape and bool(np.array_equal(a, b))

    def same_frame(a, b, check_dtype=False, **kw):
        import pandas as pd
        try:
            if isinstance(b, pd.Series):
                pd.testing.assert_series_equal(a, b, check_dtype=check_dtype, **kw)
            else:
                pd.testing.assert_frame_equal(a, b, check_dtype=check_dtype, **kw)
            return True
        except (AssertionError, TypeError, AttributeError):
            return False

    def uses(text):
        return text in source

    def figures():
        if "matplotlib.pyplot" not in sys.modules:
            return []
        plt = sys.modules["matplotlib.pyplot"]
        return [plt.figure(n) for n in plt.get_fignums()]

    return {
        "Fail": Fail, "get": get, "check": check, "close": close,
        "same_array": same_array, "same_frame": same_frame, "uses": uses,
        "figures": figures, "SOURCE": source,
    }


def _run_tests(tests, ns, source):
    results = []
    for t in tests:
        name = t.get("name") or "check"
        hint = t.get("hint") or ""
        g = dict(ns)
        g.update(_helpers(ns, source))
        try:
            exec(compile(t["code"], "<test %s>" % name, "exec"), g)
            results.append({"name": name, "passed": True, "message": ""})
        except AssertionError as e:
            msg = str(e).strip() or hint or "A hidden check failed."
            results.append({"name": name, "passed": False, "message": msg})
        except NameError as e:
            missing = getattr(e, "name", None)
            msg = ("Your code does not define `%s`." % missing) if missing else (hint or str(e))
            results.append({"name": name, "passed": False, "message": msg})
        except BaseException as e:  # noqa: BLE001 - every failure is a failed check
            msg = hint or ("A hidden check could not use your result (%s: %s)." % (type(e).__name__, e))
            results.append({"name": name, "passed": False, "message": msg})
    return results


def _figures_png(limit=3):
    if "matplotlib.pyplot" not in sys.modules:
        return []
    plt = sys.modules["matplotlib.pyplot"]
    out = []
    for n in plt.get_fignums()[:limit]:
        buf = io.BytesIO()
        try:
            plt.figure(n).savefig(buf, format="png", dpi=72, bbox_inches="tight")
        except Exception:  # noqa: BLE001 - a figure that cannot render is skipped
            continue
        out.append(base64.b64encode(buf.getvalue()).decode("ascii"))
    return out


def main():
    raw = sys.stdin.buffer.read()
    req = json.loads(raw.decode("utf-8"))
    del raw
    nonce = str(req.get("nonce", ""))
    mode = req.get("mode", "run")
    code = req.get("code", "")
    setup = req.get("setup", "") or ""
    tests = req.get("tests", []) or []
    workdir = os.getcwd()

    # Keep a private handle on the real stdout for the verdict, then point fds
    # 1 and 2 at /dev/null so nothing a C extension writes can reach the pipe.
    result_fd = os.dup(1)
    devnull = os.open(os.devnull, os.O_WRONLY)
    os.dup2(devnull, 1)
    os.dup2(devnull, 2)
    os.close(devnull)
    sys.stdin = io.StringIO("")

    _set_limits(req.get("cpu_seconds", 8), req.get("memory_mb", 1024))
    armed = _install_audit_hook(workdir, req.get("deny_paths", []) or [])

    captured = _CappedStringIO(int(req.get("max_output", 20000)))
    sys.stdout = captured
    sys.stderr = captured
    ns = {"__name__": "__main__", "__builtins__": builtins}
    error = None
    setup_error = None

    try:
        if setup:
            exec(compile(setup, SETUP_FILENAME, "exec"), ns)
    except BaseException as e:  # noqa: BLE001
        setup_error = "".join(traceback.format_exception_only(type(e), e)).strip()

    # Register the learner's source so tracebacks can quote their lines.
    import linecache
    linecache.cache[USER_FILENAME] = (len(code), None, code.splitlines(True), USER_FILENAME)

    armed[0] = True
    if setup_error is None:
        try:
            exec(compile(code, USER_FILENAME, "exec"), ns)
        except SyntaxError as e:
            error = "".join(traceback.format_exception_only(type(e), e)).rstrip()
        except BaseException as e:  # noqa: BLE001 - SystemExit and friends are learner errors too
            error = _format_user_error(e)

    test_results = []
    if mode == "grade" and setup_error is None:
        test_results = _run_tests(tests, ns, code)

    figures = _figures_png() if req.get("figures") else []
    armed[0] = False

    passed = (mode == "grade" and setup_error is None and error is None
              and len(test_results) > 0 and all(t["passed"] for t in test_results))
    out = {
        "stdout": captured.getvalue(),
        "stdout_truncated": captured.truncated,
        "error": error,
        "setup_error": setup_error,
        "tests": test_results,
        "passed": passed,
        "figures": figures,
    }
    line = (nonce + json.dumps(out) + "\n").encode("utf-8")
    view = memoryview(line)
    while view:
        n = os.write(result_fd, view)
        view = view[n:]
    os._exit(0)


if __name__ == "__main__":
    main()
