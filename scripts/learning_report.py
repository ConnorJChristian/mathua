#!/usr/bin/env python3
"""Learning analytics for one Mathua learner: where time goes, where progress
stalls and why, what review is piling up, whether each study plan is on pace,
and which questions look broken rather than hard.

    python3 scripts/learning_report.py DB [--student NAME_OR_ID] [--config FILE]
                                          [--out REPORT.md] [--json OUT.json]
                                          [--days 28]

DB is a SQLite database (a copy is fine; it is opened read-only). Without
--student the learner with the most attempts is used. The optional config is
JSON, e.g. {"deadlines": {"plan.econ5153": "2026-12-06"}}: a deadline turns
a plan's pace into "on track / behind". The concept graph and courses are read
from the repository this script sits in, local modules included.

Every finding is a pointer for a person to judge: the report says what it
saw and why it matters, and suggests a next step. It changes nothing.
"""
import argparse
import collections
import datetime as dt
import glob
import json
import os
import re
import sqlite3
import statistics
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DECAY_DAYS = 14  # internal/mastery/status.go


# ------------------------------------------------------------------ loading

def load_graph():
    concepts = {}
    pats = ["data/concepts/*.json", "modules/*/concepts/*.json", "modules-local/*/concepts/*.json"]
    for pat in pats:
        for f in sorted(glob.glob(os.path.join(ROOT, pat))):
            for c in json.load(open(f, encoding="utf-8")):
                if "label" in c:
                    concepts[c["id"]] = c
    courses = []
    for pat in ["data/courses.json", "modules/*/courses.json", "modules-local/*/courses.json"]:
        for f in sorted(glob.glob(os.path.join(ROOT, pat))):
            courses += json.load(open(f, encoding="utf-8"))
    return concepts, {c["id"]: c for c in courses}


def closure(concepts, targets):
    seen, stack = set(), [t for t in targets if t in concepts]
    while stack:
        x = stack.pop()
        if x in seen:
            continue
        seen.add(x)
        stack += [p for p in concepts[x].get("prerequisites", []) if p in concepts]
    return seen


def parse_ts(s):
    if not s:
        return None
    s = s.replace("Z", "+00:00")
    for fmt in ("%Y-%m-%d %H:%M:%S",):
        try:
            return dt.datetime.strptime(s[:19], fmt).replace(tzinfo=dt.timezone.utc)
        except ValueError:
            pass
    try:
        t = dt.datetime.fromisoformat(s)
        return t if t.tzinfo else t.replace(tzinfo=dt.timezone.utc)
    except ValueError:
        m = re.match(r"(\d{4}-\d{2}-\d{2})[T ](\d{2}:\d{2}:\d{2})", s)
        if m:
            return dt.datetime.fromisoformat(m.group(1) + "T" + m.group(2)).replace(tzinfo=dt.timezone.utc)
    return None


def connect(path):
    return sqlite3.connect("file:%s?mode=ro" % os.path.abspath(path), uri=True)


def pick_student(db, wanted):
    rows = db.execute("""SELECT s.id, s.name, s.course_id, s.daily_xp_goal, s.xp_total,
                                (SELECT COUNT(*) FROM attempts a WHERE a.student_id = s.id) n
                         FROM students s ORDER BY n DESC""").fetchall()
    if not rows:
        sys.exit("no students in this database")
    if wanted:
        rows = [r for r in rows if wanted in (r[0], r[1])]
        if not rows:
            sys.exit("no student named or with id %r" % wanted)
    sid, name, course, goal, xp, n = rows[0]
    return {"id": sid, "name": name, "course": course or "", "goal": goal or 0, "xp": xp or 0, "attempts": n}


# ------------------------------------------------------------------ analysis

def analyse(db, st, concepts, courses, cfg, days, now):
    since = now - dt.timedelta(days=days)
    label = lambda c: concepts[c]["label"] if c in concepts else c  # noqa: E731
    att = [(c, bool(ok), float(el or 0), parse_ts(ts), q or "", a or "", e or "", src or "")
           for c, ok, el, ts, q, a, e, src in db.execute(
               "SELECT concept_id, correct, elapsed_seconds, timestamp, question, answer, expected, source "
               "FROM attempts WHERE student_id = ? ORDER BY timestamp", (st["id"],))]
    att = [x for x in att if x[3]]
    recent = [x for x in att if x[3] >= since]
    prog = {}
    for row in db.execute("SELECT concept_id, status, streak, attempts, avg_response_time, last_attempted, "
                          "last_reviewed, next_review_due, mastered_at, weakness_score, placement_seeded, "
                          "implicit_credit FROM concept_progress WHERE student_id = ?", (st["id"],)):
        (cid, status, streak, n, avg, la, lr, due, mat, weak, placed, credit) = row
        prog[cid] = {"status": status, "streak": streak, "n": n, "avg": avg, "la": parse_ts(la),
                     "lr": parse_ts(lr), "due": parse_ts(due), "mastered_at": parse_ts(mat),
                     "weak": weak or 0, "placed": bool(placed), "credit": credit or 0}

    def mastered(cid):
        p = prog.get(cid)
        return bool(p and p["status"] == "MASTERED")

    def credited(cid):
        p = prog.get(cid)
        return bool(p and p["status"] == "MASTERED" and p["placed"] and not p["mastered_at"])

    out = {"student": st["name"], "generated": now.isoformat(timespec="minutes"), "window_days": days}

    # --- activity
    by_day = collections.defaultdict(lambda: [0, 0, 0.0])
    for c, ok, el, ts, *_ in recent:
        d = by_day[ts.date().isoformat()]
        d[0] += 1
        d[1] += ok
        d[2] += min(el, 600)
    xp_day = collections.Counter()
    for amt, ts in db.execute("SELECT amount, created_at FROM xp_events WHERE student_id = ? AND created_at >= ?",
                              (st["id"], int(since.timestamp()))):
        xp_day[dt.datetime.fromtimestamp(ts, dt.timezone.utc).date().isoformat()] += amt
    weeks = collections.defaultdict(lambda: [0, 0])
    for c, ok, el, ts, *_ in att:
        k = ts.date() - dt.timedelta(days=ts.weekday())
        weeks[k.isoformat()][0] += 1
        weeks[k.isoformat()][1] += ok
    active = sorted(set(by_day) | set(xp_day))
    out["activity"] = {
        "active_days": len(active), "attempts": len(recent),
        "accuracy": round(sum(x[1] for x in recent) / len(recent), 3) if recent else None,
        "minutes": round(sum(v[2] for v in by_day.values()) / 60),
        "xp": sum(xp_day.values()), "daily_goal": st["goal"],
        "days_meeting_goal": sum(1 for v in xp_day.values() if st["goal"] and v >= st["goal"]),
        "weekly": [{"week": k, "attempts": v[0], "accuracy": round(v[1] / v[0], 3)} for k, v in sorted(weeks.items())[-8:]],
        "last_active": active[-1] if active else None,
    }

    # --- stuck topics: lots of tries, no mastery, low accuracy; with the
    # weakest prerequisite as the likely root cause.
    per = collections.defaultdict(lambda: {"n": 0, "ok": 0, "el": [], "wrong": []})
    for c, ok, el, ts, q, a, e, src in att:
        p = per[c]
        p["n"] += 1
        p["ok"] += ok
        p["el"].append(el)
        if not ok:
            p["wrong"].append((q, a, e))

    def acc(c):
        p = per.get(c)
        return p["ok"] / p["n"] if p and p["n"] else None

    stuck = []
    for c, p in per.items():
        if mastered(c) or p["n"] < 8:
            continue
        a = p["ok"] / p["n"]
        if a > 0.65:
            continue
        pres = concepts.get(c, {}).get("prerequisites", [])
        weak_pre = sorted(((acc(x) if acc(x) is not None else (1.0 if mastered(x) else 0.0), x) for x in pres))
        root = [x for s, x in weak_pre if not mastered(x) or (acc(x) is not None and acc(x) < 0.7)]
        stuck.append({"id": c, "label": label(c), "attempts": p["n"], "accuracy": round(a, 2),
                      "suspect_prereqs": [{"id": x, "label": label(x), "mastered": mastered(x),
                                           "accuracy": None if acc(x) is None else round(acc(x), 2)} for x in root[:3]]})
    stuck.sort(key=lambda s: (s["accuracy"], -s["attempts"]))
    out["stuck"] = stuck[:12]

    # --- slow topics: answers much slower than the topic's fluency target
    slow = []
    for c, p in per.items():
        target = concepts.get(c, {}).get("mastery_threshold", {}).get("avg_time_seconds")
        if not target or len(p["el"]) < 5:
            continue
        med = statistics.median(p["el"][-15:])
        if med > 2 * target:
            slow.append({"id": c, "label": label(c), "median_seconds": round(med), "target_seconds": target,
                         "ratio": round(med / target, 1), "mastered": mastered(c)})
    slow.sort(key=lambda s: -s["ratio"])
    out["slow"] = slow[:12]

    # --- review load and decay
    due_now = [c for c, p in prog.items() if p["status"] == "MASTERED" and p["due"] and p["due"] <= now]
    decaying = [c for c, p in prog.items() if p["status"] == "MASTERED" and p["lr"]
                and (now - p["lr"]).days >= DECAY_DAYS]
    out["review"] = {
        "due_now": len(due_now), "decaying": len(decaying),
        "decaying_sample": [label(c) for c in sorted(decaying)[:10]],
        "overdue_7d": sum(1 for c in due_now if (now - prog[c]["due"]).days >= 7),
    }

    # --- mastery flow
    new_m = [c for c, p in prog.items() if p["mastered_at"] and p["mastered_at"] >= since]
    new_credit = [c for c, p in prog.items() if credited(c)]
    out["mastery"] = {"mastered_total": sum(1 for c in prog if mastered(c)),
                      "demonstrated_in_window": len(new_m), "credited_total": len(new_credit),
                      "per_week": round(len(new_m) / (days / 7), 1)}

    # --- plans: progress and pace
    plans = []
    deadlines = cfg.get("deadlines", {})
    pace = out["mastery"]["per_week"]
    for cid, c in courses.items():
        if not c.get("plan") and cid != st["course"]:
            continue
        path = closure(concepts, c.get("plan") or c.get("targets", []))
        done = sum(1 for x in path if mastered(x))
        left = len(path) - done
        # Pace on this plan: topics on its path newly mastered in the window.
        plan_pace = sum(1 for x in new_m if x in path) / (days / 7)
        row = {"id": cid, "name": c["name"], "total": len(path), "done": done, "left": left,
               "pace_per_week": round(plan_pace, 1), "active": cid == st["course"]}
        if cid in deadlines:
            dl = dt.datetime.fromisoformat(deadlines[cid]).replace(tzinfo=dt.timezone.utc)
            wk = max((dl - now).days / 7, 0.1)
            row.update(deadline=deadlines[cid], weeks_left=round(wk, 1), needed_per_week=round(left / wk, 1))
            row["status"] = "done" if left == 0 else ("on track" if plan_pace >= row["needed_per_week"] else "behind")
        nxt = [x for x in (c.get("plan") or []) if not mastered(x)][:5]
        row["next_plan_topics"] = [label(x) for x in nxt]
        plans.append(row)
    plans.sort(key=lambda r: (not r["active"], "deadline" not in r, r["id"]))
    out["plans"] = plans
    out["overall_pace"] = pace

    # --- content: questions that look broken rather than hard
    qstats = collections.defaultdict(lambda: {"n": 0, "ok": 0, "c": "", "answers": collections.Counter(), "expected": ""})
    for c, ok, el, ts, q, a, e, src in att:
        if not q:
            continue
        key = (c, q[:200])
        s = qstats[key]
        s["n"] += 1
        s["ok"] += ok
        s["c"] = c
        s["expected"] = e
        if not ok:
            s["answers"][a.strip()] += 1
    suspicious = []
    for (c, q), s in qstats.items():
        if s["n"] < 3 or s["ok"] / s["n"] > 0.25:
            continue
        same = s["answers"].most_common(1)
        reason = "missed %d of %d times" % (s["n"] - s["ok"], s["n"])
        if same and same[0][1] >= 2:
            reason += "; the same answer %r was given %d times" % (same[0][0][:60], same[0][1])
        # A wrong answer that equals the expected one numerically points at the grader.
        flag = ""
        # A rounded decimal of a fractional key points at the answer format.
        try:
            got, want = eval_num(same[0][0]), eval_num(s["expected"])
            if abs(got - want) < 1e-9:
                flag = "grader"
            elif abs(got - want) <= 1e-3 * max(1, abs(want)):
                flag = "rounding"
        except (ValueError, IndexError, ZeroDivisionError):
            pass
        suspicious.append({"concept": c, "label": label(c), "question": re.sub(r"<!--.*?-->", "", q).strip()[:160],
                           "expected": s["expected"][:80], "reason": reason, "flag": flag})
    suspicious.sort(key=lambda s: (not s["flag"], s["concept"]))
    out["content"] = suspicious[:15]
    try:
        out["open_reports"] = [{"concept": c, "reason": r, "detail": d[:120]} for c, r, d in db.execute(
            "SELECT concept_id, reason, detail FROM question_reports WHERE status = 'open' AND reporter_id = ?",
            (st["id"],))]
    except sqlite3.Error:
        out["open_reports"] = []

    # --- recommendations
    out["actions"] = actions(out)
    return out


def eval_num(s):
    s = s.strip().replace(" ", "")
    if re.fullmatch(r"-?\d+/\d+", s):
        a, b = s.split("/")
        return int(a) / int(b)
    return float(s)


def actions(r):
    acts = []
    a = r["activity"]
    for p in r["plans"]:
        if p.get("status") == "behind":
            acts.append("**%s is behind pace:** %d topics left in %.1f weeks needs about %.0f a week; the last %d days averaged %.1f. "
                        "Either raise the daily XP goal or trim the plan to the exam topics." %
                        (p["name"], p["left"], p["weeks_left"], p["needed_per_week"], r["window_days"], p["pace_per_week"]))
    for s in r["stuck"][:3]:
        if s["suspect_prereqs"]:
            pre = s["suspect_prereqs"][0]
            acts.append("**Stuck on %s** (%d tries, %d%% right). Likely root cause: %s%s. Revisit that first." %
                        (s["label"], s["attempts"], s["accuracy"] * 100, pre["label"],
                         "" if pre["mastered"] else " (not mastered yet)"))
        else:
            acts.append("**Stuck on %s** (%d tries, %d%% right) with its prerequisites solid: re-read the section, "
                        "or the questions may need a worked example." % (s["label"], s["attempts"], s["accuracy"] * 100))
    if r["review"]["overdue_7d"] >= 10:
        acts.append("**Review backlog:** %d reviews are a week or more overdue. Spend the next sessions on reviews "
                    "before new topics, or the decayed topics will need relearning." % r["review"]["overdue_7d"])
    if a["active_days"] and a["days_meeting_goal"] < a["active_days"] / 2:
        acts.append("**Daily goal:** met on %d of %d active days. A lower goal you hit every day beats a high one you miss."
                    % (a["days_meeting_goal"], a["active_days"]))
    if a["last_active"]:
        gap = (dt.date.fromisoformat(r["generated"][:10]) - dt.date.fromisoformat(a["last_active"])).days
        if gap >= 4:
            acts.append("**Gap:** no practice for %d days." % gap)
    graders = [c for c in r["content"] if c["flag"] == "grader"]
    rounding = [c for c in r["content"] if c["flag"] == "rounding"]
    if graders:
        acts.append("**Grader check:** %d question(s) marked a numerically equal answer wrong. Fix the answer format or the grader."
                    % len(graders))
    if rounding:
        acts.append("**Answer format:** %d question(s) rejected a correctly rounded decimal of a fraction key (%s). "
                    "Say \"exact fraction\" in the prompt, or let the key accept the decimal."
                    % (len(rounding), ", ".join(sorted({c["label"] for c in rounding}))))
    elif r["content"] and not rounding:
        acts.append("**Question check:** %d question(s) are missed nearly every time. Check the answer keys and wording."
                    % len(r["content"]))
    for s in r["slow"][:2]:
        if s["mastered"]:
            continue
        acts.append("**Slow on %s:** median %ds against a %ds target. Fluency drills or a worked example would help."
                    % (s["label"], s["median_seconds"], s["target_seconds"]))
    return acts


# ------------------------------------------------------------------ report

def markdown(r):
    L = ["# Learning report: %s" % r["generated"][:10], "",
         "Learner: %s. Window: last %d days." % (r["student"], r["window_days"]), ""]
    L += ["## Do next", ""] + (["- " + x for x in r["actions"]] or ["- Nothing urgent. Keep the current routine."]) + [""]
    a = r["activity"]
    L += ["## Activity", "",
          "| Active days | Attempts | Accuracy | Minutes | XP | Days at goal (%d XP) |" % a["daily_goal"],
          "|---|---|---|---|---|---|",
          "| %d | %d | %s | %d | %d | %d |" % (a["active_days"], a["attempts"],
                                             "—" if a["accuracy"] is None else "%d%%" % (a["accuracy"] * 100),
                                             a["minutes"], a["xp"], a["days_meeting_goal"]), ""]
    if a["weekly"]:
        L += ["Weekly attempts and accuracy: " + ", ".join("%s: %d at %d%%" % (w["week"], w["attempts"], w["accuracy"] * 100)
                                                         for w in a["weekly"]), ""]
    m = r["mastery"]
    L += ["## Mastery", "", "%d topics mastered (%d of them by prerequisite credit); %d newly demonstrated in the window, %.1f a week."
          % (m["mastered_total"], m["credited_total"], m["demonstrated_in_window"], m["per_week"]), ""]
    L += ["## Study plans", "", "| Plan | Done | Left | Pace /wk | Deadline | Needed /wk | Status |", "|---|---|---|---|---|---|---|"]
    for p in r["plans"]:
        L.append("| %s%s | %d/%d | %d | %.1f | %s | %s | %s |" % (
            p["name"].replace(": study plan", ""), " (active)" if p["active"] else "", p["done"], p["total"], p["left"],
            p["pace_per_week"], p.get("deadline", "—"), p.get("needed_per_week", "—"), p.get("status", "—")))
    act = [p for p in r["plans"] if p["active"]]
    if act and act[0]["next_plan_topics"]:
        L += ["", "Next topics on the active plan: " + "; ".join(act[0]["next_plan_topics"])]
    L += [""]
    if r["stuck"]:
        L += ["## Stuck topics", "", "| Topic | Tries | Right | Suspect prerequisite |", "|---|---|---|---|"]
        for s in r["stuck"]:
            sp = ", ".join("%s (%s)" % (x["label"], "mastered" if x["mastered"] else "not mastered") for x in s["suspect_prereqs"]) or "—"
            L.append("| %s | %d | %d%% | %s |" % (s["label"], s["attempts"], s["accuracy"] * 100, sp))
        L += [""]
    if r["slow"]:
        L += ["## Slow topics", "", "| Topic | Median s | Target s | Mastered |", "|---|---|---|---|"]
        for s in r["slow"]:
            L.append("| %s | %d | %d | %s |" % (s["label"], s["median_seconds"], s["target_seconds"], "yes" if s["mastered"] else "no"))
        L += [""]
    rv = r["review"]
    L += ["## Review", "", "%d reviews due now (%d a week or more overdue); %d mastered topics not reviewed in %d+ days."
          % (rv["due_now"], rv["overdue_7d"], rv["decaying"], DECAY_DAYS), ""]
    if r["content"] or r["open_reports"]:
        L += ["## Questions to check", ""]
        for c in r["content"]:
            L.append("- %s%s: \"%s\" (expected %s): %s" % ("**%s?** " % c["flag"] if c["flag"] else "", c["label"], c["question"],
                                                          c["expected"], c["reason"]))
        for o in r["open_reports"]:
            L.append("- Your open report on %s: %s %s" % (o["concept"], o["reason"], o["detail"]))
        L += [""]
    return "\n".join(L)


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("db")
    ap.add_argument("--student")
    ap.add_argument("--config")
    ap.add_argument("--out")
    ap.add_argument("--json")
    ap.add_argument("--days", type=int, default=28)
    ap.add_argument("--now", help="ISO date to report as of (testing)")
    args = ap.parse_args()
    cfg = json.load(open(args.config)) if args.config and os.path.exists(args.config) else {}
    now = dt.datetime.fromisoformat(args.now).replace(tzinfo=dt.timezone.utc) if args.now else dt.datetime.now(dt.timezone.utc)
    concepts, courses = load_graph()
    db = connect(args.db)
    st = pick_student(db, args.student)
    r = analyse(db, st, concepts, courses, cfg, args.days, now)
    md = markdown(r)
    if args.out:
        open(args.out, "w", encoding="utf-8").write(md + "\n")
    else:
        print(md)
    if args.json:
        json.dump(r, open(args.json, "w", encoding="utf-8"), indent=1, ensure_ascii=False, default=str)


if __name__ == "__main__":
    main()
