"""Turn flip cards into multiple-choice questions where the card allows it.

Used by scripts/import_anki.py after scripts/anki_typed.py, and runnable over
a bank already imported (modules-local/anki/tools/convert_cards.py). A flip
card ("recall") makes the learner grade themselves; a question they answer is
graded for them, so a card becomes one whenever it can be done honestly:

  - Cloze cards with one blank (or one hidden phrase repeated): the blank's
    text is the answer, and the wrong options are the blanks of other cards
    from the same deck (the same note's other blanks first: the other parts
    of the same definition, which are the natural confusions). Math blanks
    only compete with math blanks, words with words.
  - Problems whose worked solution opens with the final answer (one formula,
    a few joined by "or"/"and", a number): the answer is the correct option,
    and the wrong options are the same answer with one number changed (off by
    one, sign flipped, doubled, halved), the slips a solver actually makes.
    If two options came out equal, or fewer than three wrong ones exist, the
    card stays a flip card.

Proofs, "show that", "explain", sketches and graphs stay flip cards: their
answer is an argument, and an option list would give it away. The flip card
is the last resort, not the default.

The card keeps its id (so cards.json and skip.json still point at it) and
its full worked solution as the explanation shown after answering.
"""

import hashlib
import html
import random
import re

BLANK_RE = re.compile(r"(?:<b>\s*)?\[\.\.\.\](?:\s*</b>)?")
SPAN_RE = re.compile(r"\\\((.+?)\\\)|\\\[(.+?)\\\]", re.S)
KEEP_FLIP = re.compile(
    r"\b(prove|show that|show how|explain|describe|why|derive|sketch|graph|plot|draw|interpret|discuss|"
    r"compare|state (the|a)|what is meant|justify|verify|check (this|the|that)|give an example|"
    r"give two|construct|outline|summari[sz]e|list)\b", re.I)
NUM_RE = re.compile(r"(?<![_\\a-zA-Z\d.,])(?<!_\{)(?<!\{,\})(-?)(\d{1,3}(?:(?:,|\{,\})\d{3})+(?!\d)|\d+(?:\.\d+)?)")
MAX_OPTION = 160


def strip_tags(s):
    s = re.sub(r"<br\s*/?>", " ", s or "", flags=re.I)
    s = re.sub(r"<[^>]+>", "", s)
    return re.sub(r"\s+", " ", html.unescape(s)).strip()


def norm(s):
    s = strip_tags(s).lower()
    s = re.sub(r"\\[,;:! ]|\\(left|right)|[{}\s$]", "", s)
    return s


def is_math(s):
    return bool(SPAN_RE.search(s or ""))


def seeded(item_id):
    return random.Random(int(hashlib.md5(item_id.encode()).hexdigest()[:12], 16))


def one_line(s):
    s = re.sub(r"<br\s*/?>", " ", s, flags=re.I)
    s = re.sub(r"</?(div|p)[^>]*>", " ", s, flags=re.I)
    return re.sub(r"\s+", " ", s).strip()


# ---------------------------------------------------------------- cloze cards

def cloze_answer(prompt, back):
    """The text the card's blank hides, or None when it cannot be read off."""
    blanks = BLANK_RE.findall(prompt)
    if not blanks:
        return None
    parts = BLANK_RE.split(prompt)
    pre, post = parts[0], parts[1]
    if not back.startswith(pre[: max(0, len(pre) - 2)]):
        i = back.find(strip_tags(pre)[-30:]) if strip_tags(pre) else 0
        if i < 0:
            return None
    start = len(pre)
    # The answer runs from the end of the shared prefix to the next shared
    # text after the blank.
    tail = post[:25]
    if tail:
        j = back.find(tail, start)
        if j < 0:
            return None
        ans = back[start:j]
    else:
        ans = back[start:]
    ans = re.sub(r"^\s*<b>|</b>\s*$", "", ans.strip()).strip()
    # A blank inside a formula hides LaTeX without its delimiters: give the
    # option its own, so it renders as mathematics.
    before = BLANK_RE.split(prompt)[0]
    if before.count("\\(") > before.count("\\)") or before.count("\\[") > before.count("\\]"):
        if "\\(" in ans or "\\)" in ans:
            return None
        ans = "\\(" + ans + "\\)"
    if ans.count("{") != ans.count("}") or ans.count("\\(") != ans.count("\\)") or ans.count("\\[") != ans.count("\\]"):
        return None
    if len(blanks) > 1:
        # Several blanks: only when every blank hides the same phrase.
        filled = BLANK_RE.sub(lambda _m: "<b>" + ans + "</b>", prompt)
        if norm(filled) != norm(back[: len(filled) + 40])[: len(norm(filled))]:
            return None
    if not ans or len(strip_tags(ans)) > 80 or "[...]" in ans:
        return None
    return one_line(ans)


def cloze_options(item, ans, pool, rng):
    note = item["id"].split("c")[0]
    want_math = is_math(ans)
    n_ans = norm(ans)
    # A wrong option that is written in the question gives itself away.
    n_prompt = norm(item.get("prompt", ""))
    la = max(1, len(strip_tags(ans)))
    seen = {n_ans}
    siblings, others = [], []
    for other_id, other in pool:
        if other_id == item["id"]:
            continue
        no = norm(other)
        if not no or no in seen or no in n_ans or n_ans in no or no in n_prompt:
            continue
        if is_math(other) != want_math:
            continue
        lo = max(1, len(strip_tags(other)))
        if not (la / 3 <= lo <= la * 3):
            continue
        seen.add(no)
        (siblings if other_id.split("c")[0] == note else others).append(other)
    # Prefer wrong options shaped like the answer: similar length and word
    # count, same capitalisation. A fragment of another sentence is an easy
    # elimination, which teaches nothing.
    import math as _m

    def shape(o):
        so, sa = strip_tags(o), strip_tags(ans)
        lr = abs(_m.log(max(1, len(so)) / max(1, len(sa))))
        wd = abs(len(so.split()) - len(sa.split())) / max(1, len(sa.split()))
        cap = 0 if (so[:1].isupper() == sa[:1].isupper()) else 0.4
        return lr + wd + cap + rng.random() * 0.35

    siblings.sort(key=shape)
    others.sort(key=shape)
    picks = siblings[:2]
    picks += [o for o in others if o not in picks][: 3 - len(picks)]
    return picks[:3]


# ---------------------------------------------------------------- problems

def first_paragraph(back):
    """The answer paragraph a worked solution opens with (up to the first
    blank line), when it is short, balanced and states numbers."""
    b = re.sub(r"^(\s|<br\s*/?>|<div>|</div>)+", "", back or "")
    parts = re.split(r"(?:<br\s*/?>\s*){2,}|</div>\s*<div>\s*<br\s*/?>", b, maxsplit=1)
    if len(parts) < 2:
        return None
    head = one_line(parts[0])
    plain = strip_tags(head)
    if not plain or len(plain) > MAX_OPTION or "<img" in head or not re.search(r"\d", plain):
        return None
    if head.count("{") != head.count("}") or head.count("\\(") != head.count("\\)") or head.count("\\[") != head.count("\\]"):
        return None
    return head


def answer_head(back):
    """The final answer a worked solution opens with: a run of math spans
    joined by short connectives ("or", "and", ","), or its first short
    paragraph, or None."""
    b = re.sub(r"^(\s|<br\s*/?>|<div>|</div>)+", "", back or "")
    m = SPAN_RE.match(b)
    if not m:
        return first_paragraph(back)
    end = m.end()
    while True:
        rest = b[end:]
        j = re.match(r"\s*(,|or|and|;)?\s*", rest)
        nxt = SPAN_RE.match(rest[j.end():]) if j else None
        if nxt and j.group(1):
            end = end + j.end() + nxt.end()
            continue
        break
    head = b[:end]
    after = strip_tags(b[end:end + 40])
    # The head must stand alone: followed by the end, a full stop, a line
    # break or the worked solution, not by more of the same sentence.
    if after and not re.match(r"^([.;]|$|[A-Z(]|Since|From|Using|By|With|Because|Thus|So\b|Hence)", after):
        return first_paragraph(back)
    if len(head) > MAX_OPTION or "\n" in head:
        return None
    return one_line(head)


LABEL_BEFORE = re.compile(r"(Rule|Problem|Prob\.|Example|PDSH|Section|Chapter|§|p\.|pp\.|Theorem|Exercise|Figure|Table|step|Step|\(|\[)\s*$")
RESULT_BEFORE = re.compile(r"(=|\\approx|≈|is|are|of)\s*(\\[a-z]+\s*|\{|\$|\\\(|\\,)*\s*$")


def last_step(head):
    """A chain of working ("\\(4x-2x=9+3\\) \\(2x=12\\) \\(x=6\\)") is answered by
    its last step."""
    spans = list(SPAN_RE.finditer(head))
    if len(spans) >= 2:
        between = SPAN_RE.sub("", head).strip()
        if not between:
            return head[spans[-1].start():spans[-1].end()]
    return head


def perturb(head, rng):
    """Wrong answers: the head with one result number changed."""
    nums = [m for m in NUM_RE.finditer(head) if not LABEL_BEFORE.search(head[: m.start()])]
    # Numbers that are results (after "=" or "approx") are the ones a slip
    # changes; when there are some, vary only those.
    results = [m for m in nums if RESULT_BEFORE.search(head[: m.start()])]
    if results:
        nums = results
    # The result is stated last: "square both sides: y+3=16, so y=13" is
    # answered by y=13, and a slip in the working is not a wrong answer.
    cut = 0
    for sep in re.finditer(r",\s*so\b|;\s*|\bso\b|\bgiving\b|\btherefore\b|\bhence\b|\bthus\b", head):
        cut = sep.end()
    tail = [m for m in nums if m.start() >= cut]
    if cut and tail:
        nums = tail
    signed = "-" in head or "−" in head
    if not nums:
        return []
    out, seen = [], {norm(head)}
    cands = []
    for m in nums:
        sign, txt = m.group(1), m.group(2)
        if "," in txt:
            # Thousands separators (60,000 or 60{,}000): vary the value, keep the style.
            sep = "{,}" if "{,}" in txt else ","
            v = int(txt.replace("{,}", "").replace(",", ""))
            step = 10 ** max(0, len(str(v)) - 2)
            for val in (v + step, v - step, 2 * v):
                if val > 0:
                    cands.append(head[: m.start()] + sign + "{:,}".format(val).replace(",", sep) + head[m.end():])
            continue
        if "." in txt:
            v = float(txt)
            places = len(txt.split(".")[1])
            vals = [round(v * 1.1, places), round(v * 0.9, places), round(v + 10 ** -places, places)]
            vals = [("%." + str(places) + "f") % x for x in vals]
        else:
            v = int(txt)
            vals = [str(v + 1), str(v - 1) if v > 1 else str(v + 2), str(2 * v) if v else "1"]
            if v % 2 == 0 and v > 2:
                vals.append(str(v // 2))
        for val in vals:
            cands.append(head[: m.start()] + sign + val + head[m.end():])
        # Sign flip, only where signs are in play.
        if sign:
            cands.append(head[: m.start()] + txt + head[m.end():])
        elif signed and m.start() > 0 and head[m.start() - 1] in "=( " and txt != "0":
            cands.append(head[: m.start()] + "-" + txt + head[m.end():])
    rng.shuffle(cands)
    # Spread the changes over different numbers first.
    for c in cands:
        n = norm(c)
        if n not in seen:
            seen.add(n)
            out.append(c)
        if len(out) >= 4:
            break
    return out[:3]


# ---------------------------------------------------------------- driver

def convert(concepts, stats=None):
    """Convert flip cards in place. concepts: the bank's concept dicts (one
    file's or all files'). Returns the number of cards converted."""
    stats = stats if stats is not None else {}
    # Cloze answers by deck, for distractors.
    pools = {}
    answers = {}
    for c in concepts:
        deck = c["id"].split(".")[0] + "." + c["id"].split(".")[1] if c["id"].count(".") else c["id"]
        for it in c["items"]:
            if it.get("kind") != "recall" or "[...]" not in it.get("prompt", ""):
                continue
            a = cloze_answer(it["prompt"], it.get("back", ""))
            if a:
                answers[it["id"]] = a
                pools.setdefault(deck, []).append((it["id"], a))
    done = 0
    for c in concepts:
        deck = c["id"].split(".")[0] + "." + c["id"].split(".")[1] if c["id"].count(".") else c["id"]
        for it in c["items"]:
            if it.get("kind") != "recall":
                continue
            prompt, back = it.get("prompt", ""), it.get("back", "")
            rng = seeded(it["id"])
            if "[...]" in prompt:
                ans = answers.get(it["id"])
                if not ans:
                    stats["cloze kept: answer not readable"] = stats.get("cloze kept: answer not readable", 0) + 1
                    continue
                wrong = cloze_options(it, ans, pools.get(deck, []), rng)
                if len(wrong) < 2 and NUM_RE.search(ans):
                    wrong = perturb(ans, rng)
                if len(wrong) < 2:
                    stats["cloze kept: no distractors"] = stats.get("cloze kept: no distractors", 0) + 1
                    continue
                it["kind"] = "mc"
                it["prompt"] = BLANK_RE.sub("<b>_____</b>", prompt) + "<br><br>Which fills the blank?"
                it["correct"] = ans
                it["wrong"] = wrong
                it["explain"] = it.get("explain") or back
                stats["cloze -> choice"] = stats.get("cloze -> choice", 0) + 1
                done += 1
                continue
            if KEEP_FLIP.search(strip_tags(prompt)):
                stats["kept: proof or explanation"] = stats.get("kept: proof or explanation", 0) + 1
                continue
            head = answer_head(back)
            if head:
                head = last_step(head)
            if head and (len(strip_tags(head)) > 110 or re.search(r"Skills this exercise|Before you start", head)):
                head = None
            if not head:
                stats["kept: no final answer at the top"] = stats.get("kept: no final answer at the top", 0) + 1
                continue
            wrong = perturb(head, rng)
            if len(wrong) < 3:
                stats["kept: answer has no numbers to vary"] = stats.get("kept: answer has no numbers to vary", 0) + 1
                continue
            it["kind"] = "mc"
            it["correct"] = head
            it["wrong"] = wrong
            it["explain"] = back if not it.get("explain") else back + "<br><br>" + it["explain"]
            it["back"] = ""
            stats["problem -> choice"] = stats.get("problem -> choice", 0) + 1
            done += 1
    return done
