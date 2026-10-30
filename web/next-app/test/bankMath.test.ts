import { describe, it, expect } from 'vitest'
import fs from 'node:fs'
import path from 'node:path'
import katex from 'katex'
import { lessonMacros, prepareLessonMath } from '../lib/lessonMath'

// Every string a question bank puts on the page (a concept's intro, each
// item's prompt, choices, explanation, flip-card back, solution, and the
// parts of a multi-part item), plus each proof problem's statement, is pushed
// through the lesson math pipeline (prepareLessonMath) and every math span
// is rendered by KaTeX with throwOnError, as the lesson renders it (fork
// change). A span KaTeX rejects reaches the learner as a dashed fallback box
// of raw TeX, so the public modules (modules/) may not have any.
//
// Your own modules (modules-local/, git-ignored) are checked too when they
// are there: their failures are listed, and fail the test only with
// MATHUA_LOCAL_MATH_STRICT=1, so a machine with a half-built bank still runs
// the suite. Strict-mode warnings (Unicode in math, a \\ outside an
// environment) never fail: the page renders them with strict: 'warn'.
const REPO_ROOT = path.join(__dirname, '../../..')
const SPAN = /<span class="math-(display|inline)">([\s\S]*?)<\/span>/g

function unescapeHtml(s: string): string {
  return s
    .replace(/&amp;/g, '&')
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/&#(\d+);/g, (_, n: string) => String.fromCharCode(Number(n)))
}

type Failure = { where: string; kind: string; span: string }
type Report = { strings: number; spans: number; failures: Failure[]; warnings: number }

export function checkText(where: string, text: string, rep: Report): void {
  if (!text) return
  rep.strings++
  const prepared = prepareLessonMath(text)
  let m: RegExpExecArray | null
  SPAN.lastIndex = 0
  while ((m = SPAN.exec(prepared)) !== null) {
    rep.spans++
    const inner = unescapeHtml(m[2])
    try {
      katex.renderToString(inner, {
        displayMode: m[1] === 'display',
        throwOnError: true,
        trust: false,
        macros: lessonMacros,
        strict: () => {
          rep.warnings++
          return 'ignore'
        },
      })
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e)
      rep.failures.push({ where, kind: msg.replace(/ at position \d+:[\s\S]*$/, '').trim(), span: inner.slice(0, 120) })
    }
  }
}

// Every text field of a bank item, however it is answered.
function itemTexts(it: Record<string, unknown>): string[] {
  const out: string[] = []
  for (const k of ['prompt', 'explain', 'back', 'solution', 'correct', 'expected']) {
    if (typeof it[k] === 'string') out.push(it[k] as string)
  }
  for (const k of ['wrong', 'accept']) {
    const v = it[k]
    if (Array.isArray(v)) for (const w of v) if (typeof w === 'string') out.push(w)
  }
  const parts = it.parts
  if (Array.isArray(parts)) {
    for (const p of parts) {
      if (p && typeof p === 'object') for (const k of ['prompt', 'answer', 'explain']) if (typeof (p as Record<string, unknown>)[k] === 'string') out.push((p as Record<string, unknown>)[k] as string)
    }
  }
  return out
}

function checkModule(dir: string, rep: Report): void {
  const bank = path.join(dir, 'bank')
  if (fs.existsSync(bank)) {
    // bank/skip.json lists items the server never serves (quarantined cards).
    const skipPath = path.join(bank, 'skip.json')
    const skipped = new Set<string>(fs.existsSync(skipPath) ? Object.keys(JSON.parse(fs.readFileSync(skipPath, 'utf8'))) : [])
    for (const f of fs.readdirSync(bank).filter((n) => n.endsWith('.json') && n !== 'skip.json').sort()) {
      const data = JSON.parse(fs.readFileSync(path.join(bank, f), 'utf8'))
      for (const c of data.concepts ?? []) {
        const base = `${path.basename(dir)}/bank/${f} ${c.id}`
        checkText(base + ' intro', c.intro ?? '', rep)
        for (const it of c.items ?? []) {
          if (skipped.has(it.id)) continue
          for (const t of itemTexts(it)) checkText(`${base}/${it.id}`, t, rep)
        }
      }
    }
  }
  const proofs = path.join(dir, 'proofs')
  if (fs.existsSync(proofs)) {
    for (const f of fs.readdirSync(proofs).filter((n) => n.endsWith('.json')).sort()) {
      const data = JSON.parse(fs.readFileSync(path.join(proofs, f), 'utf8'))
      const probs = Array.isArray(data) ? data : data.problems ?? []
      for (const p of probs) {
        const base = `${path.basename(dir)}/proofs/${f} ${p.id ?? ''}`
        for (const k of ['statement', 'prompt', 'hint', 'model', 'solution']) if (typeof p[k] === 'string') checkText(`${base} ${k}`, p[k], rep)
        for (const s of p.scheme ?? p.marking ?? []) if (s && typeof s.text === 'string') checkText(`${base} scheme`, s.text, rep)
      }
    }
  }
}

function modulesIn(area: string): string[] {
  const base = path.join(REPO_ROOT, area)
  if (!fs.existsSync(base)) return []
  return fs
    .readdirSync(base)
    .filter((n) => !n.startsWith('.') && !n.startsWith('_') && fs.existsSync(path.join(base, n, 'module.json')))
    .map((n) => path.join(base, n))
}

function format(rep: Report, label: string): string {
  const byKind = new Map<string, number>()
  for (const f of rep.failures) byKind.set(f.kind, (byKind.get(f.kind) ?? 0) + 1)
  const lines = [
    `${label}: ${rep.strings} strings, ${rep.spans} math spans, ${rep.failures.length} KaTeX errors, ${rep.warnings} strict warnings`,
    ...[...byKind.entries()].sort((a, b) => b[1] - a[1]).map(([k, n]) => `  ${n}× ${k}`),
    ...rep.failures.slice(0, 60).map((f) => `  ${f.where}: ${f.kind} — ${f.span}`),
  ]
  return lines.join('\n')
}

describe('question banks render through KaTeX', () => {
  it('public modules have no math span KaTeX rejects', { timeout: 120_000 }, () => {
    const rep: Report = { strings: 0, spans: 0, failures: [], warnings: 0 }
    for (const dir of modulesIn('modules')) checkModule(dir, rep)
    expect(rep.strings, 'the public banks have text').toBeGreaterThan(0)
    const out = format(rep, 'modules/')
    console.log(out)
    expect(rep.failures, out).toEqual([])
  })

  it('your own modules list every span KaTeX rejects', { timeout: 600_000 }, () => {
    const dirs = modulesIn('modules-local')
    if (dirs.length === 0) return
    const rep: Report = { strings: 0, spans: 0, failures: [], warnings: 0 }
    for (const dir of dirs) checkModule(dir, rep)
    const out = format(rep, 'modules-local/')
    console.log(out)
    const reportPath = path.join(REPO_ROOT, 'local-data', 'qa', 'bank_math_report.txt')
    if (fs.existsSync(path.dirname(reportPath))) {
      fs.writeFileSync(reportPath, rep.failures.map((f) => `${f.where}\t${f.kind}\t${f.span}`).join('\n') + '\n')
    }
    if (process.env.MATHUA_LOCAL_MATH_STRICT === '1') expect(rep.failures, out).toEqual([])
  })
})
