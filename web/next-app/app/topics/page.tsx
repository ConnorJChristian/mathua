'use client'

import { Suspense, useEffect, useMemo, useState } from 'react'
import Link from 'next/link'
import { useRouter, useSearchParams } from 'next/navigation'
import Header from '../../components/Header'
import BottomTabs from '../../components/BottomTabs'
import Footer from '../../components/Footer'
import Loading from '../../components/Loading'
import SectionHeader from '../../components/SectionHeader'
import { useTheme } from '../../hooks/useTheme'
import { getCourseCatalog, getProgress, getTranscript, setActiveCourses, type ConceptProgress, type CourseStatus } from '../../lib/api'
import { getUserInfo } from '../../lib/auth'
import { concepts, type ConceptRecord } from '../../lib/conceptData'
import { domainLabel } from '../../lib/domains'
import { masteryState, type MasteryState } from '../../lib/progress'
import { learnConceptPath, studyConceptPath } from '../../lib/learnReturn'

// Fork change: the topic library. Every topic in the install, searchable and
// filterable, each one a click away from its lesson (Read) and its practice
// (Practise). Study plans sit on top: pick one to list its topics in teaching
// order, or make it the plan the recommendation shelf follows.
//
// Upstream closed the browsable library (ADR-047) so that reading could not
// stand in for practice. That still holds here: reading awards nothing, and
// practice still runs through /learn's eligibility check (with its own
// "practise anyway" choice for a locked topic).

const PAGE = 150
const byId = new Map<string, ConceptRecord>(concepts.map((c) => [c.id, c]))

const depth = new Map<string, number>()
function depthOf(id: string, seen = new Set<string>()): number {
  const d = depth.get(id)
  if (d !== undefined) return d
  if (seen.has(id)) return 0
  seen.add(id)
  let v = 0
  for (const p of byId.get(id)?.prerequisites ?? []) if (byId.has(p)) v = Math.max(v, depthOf(p, seen) + 1)
  depth.set(id, v)
  return v
}

/** A planned course's path: each plan topic preceded by its unlisted prerequisites. */
function planPath(plan: string[]): string[] {
  const out: string[] = []
  const seen = new Set<string>()
  const visit = (id: string) => {
    if (seen.has(id) || !byId.has(id)) return
    seen.add(id)
    for (const p of byId.get(id)?.prerequisites ?? []) visit(p)
    out.push(id)
  }
  plan.forEach(visit)
  return out
}

function stateWord(s: MasteryState): string {
  switch (s) {
    case 'mastered': return 'Mastered'
    case 'decaying': return 'Due a review'
    case 'practicing':
    case 'learning': return 'In progress'
    default: return ''
  }
}

function shortName(name: string): string {
  return name.replace(/: study plan$/, '')
}

function TopicsContent() {
  const { mounted } = useTheme()
  const router = useRouter()
  const params = useSearchParams()
  const [q, setQ] = useState(params.get('q') ?? '')
  const [domain, setDomain] = useState(params.get('domain') ?? '')
  const [planId, setPlanId] = useState(params.get('plan') ?? '')
  const [withPrereqs, setWithPrereqs] = useState(params.get('all') === '1')
  const [limit, setLimit] = useState(PAGE)
  const [progress, setProgress] = useState<Record<string, ConceptProgress>>({})
  const [courses, setCourses] = useState<CourseStatus[]>([])
  const [following, setFollowing] = useState<string[]>([])
  const [signedIn, setSignedIn] = useState(false)
  const [busy, setBusy] = useState('')
  const [notice, setNotice] = useState('')

  useEffect(() => {
    const sid = getUserInfo()?.student_id
    setSignedIn(!!sid)
    if (sid) getProgress(sid).then(setProgress).catch(() => {})
    void (async () => {
      // The catalog and the transcript are independent: asked for together, and
      // a failed one (the server down) leaves the list of topics as it is.
      const [cat, withProgress] = await Promise.all([
        getCourseCatalog().catch(() => ({ courses: [] as CourseStatus[], active: '', activeIds: [] as string[] })),
        sid ? getTranscript().catch(() => [] as CourseStatus[]) : Promise.resolve([] as CourseStatus[]),
      ])
      setFollowing(cat.activeIds)
      // Opened plainly (no filters in the link): start on the first plan you follow.
      const first = cat.activeIds.find((id) => cat.courses.some((c) => c.id === id && (c.plan?.length ?? 0) > 0))
      if (first && !window.location.search) setPlanId(first)
      setCourses(withProgress.length ? withProgress : cat.courses)
    })()
  }, [])

  // Keep the filters in the URL so a filtered list can be bookmarked.
  useEffect(() => {
    const u = new URLSearchParams()
    if (q) u.set('q', q)
    if (domain) u.set('domain', domain)
    if (planId) u.set('plan', planId)
    if (withPrereqs) u.set('all', '1')
    const s = u.toString()
    router.replace(s ? `/topics?${s}` : '/topics', { scroll: false })
    setLimit(PAGE)
  }, [q, domain, planId, withPrereqs, router])

  const plans = useMemo(() => courses.filter((c) => (c.plan?.length ?? 0) > 0), [courses])
  const plan = plans.find((c) => c.id === planId) ?? null

  const domains = useMemo(() => {
    const set = new Set<string>()
    for (const c of concepts) set.add(c.domain)
    return [...set].sort((a, b) => domainLabel(a).localeCompare(domainLabel(b)))
  }, [])

  const rows = useMemo(() => {
    const needle = q.trim().toLowerCase()
    let ids: string[]
    let planned = new Set<string>()
    if (plan) {
      planned = new Set(plan.plan)
      ids = withPrereqs ? planPath(plan.plan ?? []) : (plan.plan ?? []).filter((id) => byId.has(id))
    } else {
      ids = concepts
        .map((c) => c.id)
        .sort((a, b) => {
          const ca = byId.get(a)!, cb = byId.get(b)!
          return domainLabel(ca.domain).localeCompare(domainLabel(cb.domain)) || depthOf(a) - depthOf(b) || ca.label.localeCompare(cb.label)
        })
    }
    return ids.filter((id) => {
      const c = byId.get(id)!
      if (domain && c.domain !== domain) return false
      if (needle && !c.label.toLowerCase().includes(needle) && !c.id.toLowerCase().includes(needle)) return false
      return true
    }).map((id) => ({ c: byId.get(id)!, planned: planned.has(id) }))
  }, [q, domain, plan, withPrereqs])

  async function toggleFollow(id: string) {
    const next = following.includes(id) ? following.filter((x) => x !== id) : [...following, id]
    setBusy(id)
    setNotice('')
    try {
      setFollowing(await setActiveCourses(next))
      setNotice(next.length === 0
        ? 'You follow no plans now; recommendations come from the whole graph.'
        : `Recommendations now mix ${next.length === 1 ? 'this plan' : `your ${next.length} plans`}.`)
    } catch {
      setNotice('Could not save. Are you signed in?')
    } finally {
      setBusy('')
    }
  }

  if (!mounted) return <Loading label="Loading topics" full />

  const btn = 'shrink-0 border px-3 py-1.5 font-mono text-[11px] min-h-[32px] inline-flex items-center justify-center'
  const input = 'border border-mathua-border bg-mathua-surface text-mathua-primary font-mono text-xs px-3 py-2 min-h-[36px] min-w-0'

  // Opened from a study plan, a topic reads in that plan's own texts first.
  const withCourse = (href: string) => (plan ? `${href}&course=${encodeURIComponent(plan.id)}` : href)

  let lastDomain = ''
  return (
    <>
      <Header />
      <main className="mx-auto w-full max-w-[920px] min-w-0 px-4 sm:px-6 py-8 sm:py-12 overflow-x-hidden">
        <SectionHeader label="Library" title="Topics" />
        <p className="font-mono text-[11px] text-mathua-secondary mb-6">
          Every topic, with its lesson one click away. <b>Read</b> opens the lesson and what it builds on;{' '}
          <b>Practise</b> starts questions. Reading earns nothing on its own; mastery comes from practice.
          Browse by book in the <Link href="/library" className="underline hover:text-mathua-blue">Library</Link>.
        </p>

        {signedIn && plans.length > 0 && (
          <details className="mb-5 border border-mathua-border bg-mathua-surface" open={following.length === 0}>
            <summary className="cursor-pointer px-4 py-3 font-mono text-xs text-mathua-primary">
              Plans you follow: {following.length === 0
                ? 'none'
                : following.map((id) => shortName(plans.find((c) => c.id === id)?.name ?? id)).join(', ')}
            </summary>
            <div className="px-4 pb-3">
              <p className="font-mono text-[11px] text-mathua-secondary mb-2">
                Tick every plan you are working on. Your recommendations take turns between them, earliest topics first.
              </p>
              <ul className="grid gap-1 sm:grid-cols-2">
                {plans.map((c) => (
                  <li key={c.id} className="min-w-0">
                    <label className="flex items-start gap-2 font-mono text-[11px] text-mathua-primary py-1 cursor-pointer">
                      <input
                        id={`follow-${c.id}`}
                        type="checkbox"
                        checked={following.includes(c.id)}
                        disabled={busy !== ''}
                        onChange={() => void toggleFollow(c.id)}
                        className="mt-0.5"
                      />
                      <span className="min-w-0 break-words [overflow-wrap:anywhere]">
                        {shortName(c.name)}
                        {c.progress && <span className="text-mathua-muted"> · {Math.round(c.progress.pct * 100)}%</span>}
                      </span>
                    </label>
                  </li>
                ))}
              </ul>
            </div>
          </details>
        )}
        {!plan && notice && <p role="status" className="mb-3 font-mono text-[11px] text-mathua-secondary">{notice}</p>}
        <div className="flex flex-wrap gap-2 mb-3 items-end">
          {plans.length > 0 && (
            <label className="flex flex-col gap-1 min-w-0">
              <span className="font-mono text-[10px] uppercase tracking-wider text-mathua-muted">Study plan</span>
              <select id="topic-plan" value={planId} onChange={(e) => { setPlanId(e.target.value); setDomain('') }} className={`${input} max-w-[260px]`}>
                <option value="">All topics</option>
                {plans.map((c) => (
                  <option key={c.id} value={c.id}>{shortName(c.name)}{following.includes(c.id) ? ' (following)' : ''}</option>
                ))}
              </select>
            </label>
          )}
          <label className="flex flex-col gap-1 flex-1 min-w-[180px]">
            <span className="font-mono text-[10px] uppercase tracking-wider text-mathua-muted">Search</span>
            <input id="topic-q" type="search" value={q} onChange={(e) => setQ(e.target.value)} placeholder="Lagrange, elasticity, groupby…" className={input} />
          </label>
          <label className="flex flex-col gap-1 min-w-0">
            <span className="font-mono text-[10px] uppercase tracking-wider text-mathua-muted">Domain</span>
            <select id="topic-domain" value={domain} onChange={(e) => setDomain(e.target.value)} className={`${input} max-w-[240px]`}>
              <option value="">All domains</option>
              {domains.map((d) => <option key={d} value={d}>{domainLabel(d)}</option>)}
            </select>
          </label>
        </div>

        {plan && (
          <div className="mb-3 border border-mathua-border bg-mathua-surface p-3 flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-4 min-w-0">
            <div className="min-w-0 flex-1">
              <div className="font-mono text-xs text-mathua-primary">{shortName(plan.name)}</div>
              <div className="font-mono text-[10px] text-mathua-secondary">
                {plan.plan?.length} plan topics
                {plan.progress && ` · ${plan.progress.mastered} of ${plan.progress.total} on its path mastered (${Math.round(plan.progress.pct * 100)}%)`}
              </div>
              {plan.progress && (
                <div className="mt-1.5 h-1 bg-mathua-border" aria-hidden>
                  <div className="h-1 bg-mathua-blue" style={{ width: `${Math.round(plan.progress.pct * 100)}%` }} />
                </div>
              )}
            </div>
            {signedIn && (
              <button
                type="button"
                aria-pressed={following.includes(plan.id)}
                disabled={busy === plan.id}
                onClick={() => void toggleFollow(plan.id)}
                className={`${btn} border-mathua-blue text-mathua-blue hover:bg-mathua-blue-faint disabled:opacity-60 ${following.includes(plan.id) ? 'bg-mathua-blue-faint' : ''}`}
              >
                {busy === plan.id ? 'Saving…' : following.includes(plan.id) ? '✓ Following' : 'Follow this plan'}
              </button>
            )}
          </div>
        )}
        {plan && notice && <p role="status" className="mb-3 font-mono text-[11px] text-mathua-secondary">{notice}</p>}
        <div className="flex flex-wrap items-center justify-between gap-2 mb-3">
          <p className="font-mono text-[11px] text-mathua-secondary" aria-live="polite">
            {plan ? `${shortName(plan.name)} in teaching order: ` : ''}
            {rows.length} topic{rows.length === 1 ? '' : 's'}
          </p>
          {plan && (
            <label className="font-mono text-[11px] text-mathua-secondary inline-flex items-center gap-2">
              <input id="topic-all" type="checkbox" checked={withPrereqs} onChange={(e) => setWithPrereqs(e.target.checked)} />
              Include prerequisites
            </label>
          )}
        </div>

        {rows.length === 0 ? (
          <p className="border border-mathua-border bg-mathua-surface p-4 font-mono text-xs text-mathua-secondary">
            No topics match. Clear the search or pick another domain.
          </p>
        ) : (
          <ol className="border border-mathua-border divide-y divide-mathua-border bg-mathua-surface">
            {rows.slice(0, limit).map(({ c, planned }, n) => {
              const st = masteryState(progress[c.id])
              const showDomainHead = !plan && c.domain !== lastDomain
              lastDomain = c.domain
              return (
                <li key={c.id} className="min-w-0">
                  {showDomainHead && (
                    <div className="px-4 pt-3 pb-1 font-mono text-[10px] uppercase tracking-wider text-mathua-muted bg-mathua-bg">
                      {domainLabel(c.domain)}
                    </div>
                  )}
                  <div className="px-4 py-2.5 flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-3 min-w-0">
                    <div className="min-w-0 flex-1 flex gap-3">
                      {plan && <span className="shrink-0 w-7 text-right font-mono text-[11px] text-mathua-muted tabular-nums">{n + 1}</span>}
                      <div className="min-w-0">
                        <Link
                          href={withCourse(studyConceptPath(c.id))}
                          className={`font-mono text-xs break-words [overflow-wrap:anywhere] hover:text-mathua-blue ${planned || !plan ? 'text-mathua-primary' : 'text-mathua-secondary'} ${planned ? 'font-semibold' : ''}`}
                        >
                          {c.label}
                        </Link>
                        <div className="font-mono text-[10px] text-mathua-muted">
                          {plan ? domainLabel(c.domain) : ''}
                          {plan && stateWord(st) ? ' · ' : ''}
                          {stateWord(st) && <span className={st === 'mastered' ? 'text-mathua-blue' : ''}>{stateWord(st)}</span>}
                        </div>
                      </div>
                    </div>
                    <div className="flex gap-2 sm:shrink-0">
                      <Link href={withCourse(studyConceptPath(c.id))} className={`${btn} flex-1 sm:flex-none border-mathua-border text-mathua-secondary hover:border-mathua-blue hover:text-mathua-blue`}>
                        Read
                      </Link>
                      <Link href={withCourse(learnConceptPath(c.id))} className={`${btn} flex-1 sm:flex-none border-mathua-blue text-mathua-blue hover:bg-mathua-blue-faint`}>
                        Practise
                      </Link>
                    </div>
                  </div>
                </li>
              )
            })}
          </ol>
        )}
        {rows.length > limit && (
          <button
            type="button"
            onClick={() => setLimit(limit + PAGE)}
            className={`${btn} mt-3 w-full border-mathua-border text-mathua-secondary hover:border-mathua-blue hover:text-mathua-blue`}
          >
            Show {Math.min(PAGE, rows.length - limit)} more of {rows.length - limit}
          </button>
        )}
      </main>
      <Footer />
      <BottomTabs />
    </>
  )
}

export default function TopicsPage() {
  return (
    <Suspense fallback={<Loading label="Loading topics" full />}>
      <TopicsContent />
    </Suspense>
  )
}
