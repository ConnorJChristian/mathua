'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { recommendationTitle } from '../../lib/recommendations'
import Header from '../../components/Header'
import BottomTabs from '../../components/BottomTabs'
import Footer from '../../components/Footer'
import Loading from '../../components/Loading'
import SectionHeader from '../../components/SectionHeader'
import { useTheme } from '../../hooks/useTheme'
import { getCourseCatalog, getCourseView, getDueReviews, getNext, getPins, getScores, setPins, type CourseView, type Pin, type Recommendation } from '../../lib/api'
import { getUserInfo } from '../../lib/auth'
import { formatDue, topicsPerWeek } from '../../lib/dates'
import { hrefConceptId } from '../../lib/nextUp'
import { quizXPLine } from '../../lib/quizGate'
import { track } from '../../lib/events'

// Fork change: the Study tab. One queue across every course you follow, the
// way Math Academy opens on "today": the engine's next task first, then the
// alternatives, each tagged with the course it belongs to, with the mastery
// test and the review session surfaced when they are due. Nothing here is
// decided in the browser: the order is /api/next's, the counts are the server's.
//
// "This week" is the learner's own hand on the queue: topics pinned on a
// course's Modules tab lead the queue after due reviews, and the engine
// (placement, sequencing, the courses' due dates) serves the rest.

type Row = Recommendation & { courses: string[] }

export default function TodayPage() {
  const { mounted } = useTheme()
  const [rows, setRows] = useState<Row[] | null>(null)
  const [views, setViews] = useState<CourseView[]>([])
  const [dueCount, setDueCount] = useState<number | null>(null)
  const [quizDue, setQuizDue] = useState(false)
  const [xp, setXp] = useState<{ since: number; gate: number; taken?: boolean } | null>(null)
  const [failed, setFailed] = useState(false)
  const [pins, setPinList] = useState<Pin[]>([])
  const [signedIn, setSignedIn] = useState(false)

  const unpin = async (cid: string) => {
    track({ kind: 'pin', concept: cid, data: { on: false, from: 'study' } })
    const next = pins.filter((p) => p.concept !== cid)
    setPinList(next)
    try {
      setPinList(await setPins(next.map((p) => p.concept)))
      setRows(await loadRows())
    } catch { /* keep the optimistic state */ }
  }

  const [byConcept, setByConcept] = useState<Map<string, string[]>>(new Map())
  const loadRows = async (map?: Map<string, string[]>) => {
    const next = await getNext()
    const m = map ?? byConcept
    const list = [next.primary, ...next.alternatives].filter((r): r is Recommendation => !!r)
    return list.map((r) => ({ ...r, courses: m.get(r.conceptId || hrefConceptId(r.action.href) || '') ?? [] }))
  }

  useEffect(() => {
    const sid = getUserInfo()?.student_id
    setSignedIn(!!sid)
    ;(async () => {
      try {
        const [{ activeIds }, ps] = await Promise.all([getCourseCatalog(), sid ? getPins().catch(() => []) : Promise.resolve([])])
        setPinList(ps)
        const vs = (await Promise.all(activeIds.map((id) => getCourseView(id)))).filter((v): v is CourseView => !!v)
        setViews(vs)
        const map = new Map<string, string[]>()
        for (const v of vs) {
          const name = v.name.replace(/: study plan$/, '')
          for (const m of v.modules) for (const t of m.topics) map.set(t.id, [...(map.get(t.id) ?? []), name])
        }
        setByConcept(map)
        setRows(await loadRows(map))
        if (sid) {
          getDueReviews().then((d) => setDueCount(d.count)).catch(() => {})
          const s = await getScores(sid).catch(() => null)
          if (s) {
            setQuizDue(!!s.quiz_due)
            if (s.quiz_gate_xp) setXp({ since: s.xp_since_quiz ?? 0, gate: s.quiz_gate_xp, taken: s.quiz_taken })
          }
        }
      } catch {
        setFailed(true)
        setRows([])
      }
    })()
  }, [])

  if (!mounted || rows === null) return <Loading label="Working out what to study" full />
  // The count is the review service's own, so it is the number /review will
  // serve: summing course pages missed mastered prerequisites no course lists
  // (Q38) and counted a topic on two courses twice (Q2). Course pages are the
  // fallback while it loads.
  const reviewsDue = dueCount ?? new Set(views.flatMap((v) => v.modules.flatMap((m) => m.topics.filter((t) => t.due).map((t) => t.id)))).size
  const kindLabel = (r: Recommendation) => r.pinned ? (r.pinnedFor ? `This week · for ${r.pinnedFor}` : 'This week') : r.kind === 'review' ? 'Review' : r.badge === 'New' ? 'New topic' : r.kind === 'practice' ? 'Practice' : r.kind === 'mastery_check' ? 'Mastery check' : r.reason === 'prerequisite' ? 'Prerequisite' : 'New topic'
  const pinCourses = (cid: string) => byConcept.get(cid) ?? []

  return (
    <>
      <Header />
      <main className="mx-auto w-full max-w-[900px] min-w-0 px-4 sm:px-6 py-8 sm:py-12 overflow-x-hidden">
        <SectionHeader label="Today" title="Study" />
        {views.length === 0 && (
          <p className="mb-4 border border-mathua-border bg-mathua-surface p-3 font-mono text-[11px] text-mathua-secondary">
            You follow no courses, so this queue draws from every topic. <Link href="/courses" className="underline">Pick your courses</Link> to focus it.
          </p>
        )}
        {failed && <p className="mb-4 font-mono text-xs text-mathua-red">The queue could not be loaded. Check the server and try again.</p>}

        {(quizDue || reviewsDue > 0) && (
          <div className="mb-6 grid gap-2 sm:grid-cols-2">
            {quizDue && (
              <Link href="/goals?quiz=1" className="border border-mathua-blue bg-mathua-surface p-3 hover:bg-mathua-blue-faint">
                <p className="font-mono text-[10px] uppercase tracking-wider text-mathua-blue">Mastery check due</p>
                <p className="font-mono text-xs text-mathua-primary mt-1">Covers what your courses have taught you{xp ? ` · ${quizXPLine(xp.since, xp.taken)}` : ''}. Take it →</p>
              </Link>
            )}
            {reviewsDue > 0 && (
              <Link href="/review" className="border border-mathua-border bg-mathua-surface p-3 hover:border-mathua-blue">
                <p className="font-mono text-[10px] uppercase tracking-wider text-mathua-amber">Reviews due</p>
                <p className="font-mono text-xs text-mathua-primary mt-1">{reviewsDue} topic{reviewsDue === 1 ? '' : 's'} across your courses. Review →</p>
              </Link>
            )}
          </div>
        )}

        {signedIn && (
          <section className="mb-6 border border-mathua-border bg-mathua-surface p-3">
            <div className="flex flex-wrap items-baseline justify-between gap-2">
              <h2 className="font-mono text-[11px] uppercase tracking-[0.15em] text-mathua-muted">This week</h2>
              <p className="font-mono text-[10px] text-mathua-muted">Your picks lead the queue after due reviews; the rest is the engine&apos;s.</p>
            </div>
            {pins.length === 0 ? (
              <p className="mt-2 font-mono text-[11px] text-mathua-secondary">
                Nothing pinned. Open a course&apos;s <span className="text-mathua-primary">Modules</span> tab and pin the topics you are covering this week{views.length ? <> — e.g. <Link href={`/course?id=${encodeURIComponent(views[0].id)}&tab=modules`} className="underline hover:text-mathua-blue">{views[0].name.replace(/: study plan$/, '')}</Link></> : null}.
              </p>
            ) : (
              <ul className="mt-2 flex flex-wrap gap-1.5">
                {pins.map((p) => {
                  const age = Math.floor((Date.now() - new Date(`${p.since}T12:00:00`).getTime()) / 86400000)
                  const stale = age >= 7
                  return (
                    <li key={p.concept} className={`inline-flex items-center gap-1 border px-2 min-h-[28px] font-mono text-[11px] ${p.done ? 'border-mathua-green text-mathua-secondary' : stale ? 'border-mathua-border text-mathua-secondary' : 'border-mathua-blue text-mathua-primary'}`} title={p.done ? 'Mastered — unpin it when you are done with it' : stale ? `Pinned ${age} days ago — still this week's?` : `Pinned ${p.since}`}>
                      <Link href={`/learn?concept=${encodeURIComponent(p.concept)}`} className="hover:text-mathua-blue break-words [overflow-wrap:anywhere]">{p.done ? '✓' : '●'} {p.label}</Link>
                      {pinCourses(p.concept).length ? <span className="text-mathua-muted">· {pinCourses(p.concept)[0]}</span> : null}
                      {stale && <span className="text-mathua-amber">· {age}d</span>}
                      <button type="button" onClick={() => void unpin(p.concept)} aria-label={`Unpin ${p.label}`} className="ml-0.5 text-mathua-muted hover:text-mathua-red">×</button>
                    </li>
                  )
                })}
              </ul>
            )}
          </section>
        )}

        <ol className="space-y-2">
          {rows.map((r, i) => (
            <li key={r.id + i} className={`border bg-mathua-surface p-3 flex flex-wrap items-center gap-x-3 gap-y-2 min-w-0 ${i === 0 ? 'border-mathua-blue' : 'border-mathua-border'}`}>
              <div className="min-w-0 flex-1 basis-56">
                <p className="font-mono text-[10px] uppercase tracking-wider text-mathua-muted">
                  {i === 0 ? 'Up next' : kindLabel(r)}{r.courses.length ? ` · ${r.courses.join(', ')}` : ''}
                </p>
                <p className="font-serif text-base text-mathua-primary break-words [overflow-wrap:anywhere]">{r.id === 'paused' && !r.conceptTitle ? 'Paused placement test' : recommendationTitle(r)}</p>
                {r.detail && <p className="font-mono text-[11px] text-mathua-secondary mt-0.5 break-words [overflow-wrap:anywhere]">{r.detail}</p>}
                {r.why && <p data-why className="font-mono text-[10px] text-mathua-muted mt-0.5 break-words [overflow-wrap:anywhere]">Why: {r.why}</p>}
              </div>
              <div className="flex items-center gap-2 ml-auto">
                {r.xp ? <span className="font-mono text-[10px] text-mathua-muted">+{r.xp} XP</span> : null}
                <Link href={r.action.href} onClick={() => track({ kind: 'queue_click', concept: r.conceptId, data: { rank: i, kind: r.kind, badge: r.badge, pinned: !!r.pinned, courses: r.courses } })} className={`border px-3 min-h-[34px] inline-flex items-center font-mono text-[11px] ${i === 0 ? 'border-mathua-blue text-mathua-blue hover:bg-mathua-blue-faint' : 'border-mathua-border text-mathua-secondary hover:border-mathua-blue hover:text-mathua-blue'}`}>
                  {r.cta || 'Go →'}
                </Link>
              </div>
            </li>
          ))}
          {rows.length === 0 && !failed && <li className="font-mono text-xs text-mathua-secondary">Nothing is due. Open a course to pick a topic.</li>}
        </ol>

        {views.length > 0 && (
          <section className="mt-8">
            <h2 className="font-mono text-[11px] uppercase tracking-[0.15em] text-mathua-muted mb-2">By course</h2>
            <ul className="grid gap-2 sm:grid-cols-2">
              {views.map((v) => (
                <li key={v.id} className="border border-mathua-border bg-mathua-surface p-3 min-w-0">
                  <Link href={`/course?id=${encodeURIComponent(v.id)}`} className="font-mono text-xs text-mathua-primary hover:text-mathua-blue break-words [overflow-wrap:anywhere]">{v.name.replace(/: study plan$/, '')}</Link>
                  <div className="mt-1.5 h-1 bg-mathua-code overflow-hidden"><div className="h-full bg-mathua-blue" style={{ width: `${v.total ? Math.round((v.mastered / v.total) * 100) : 0}%` }} /></div>
                  <p className="font-mono text-[10px] text-mathua-muted mt-1">
                    {v.mastered}/{v.total} mastered{v.reviews_due ? ` · ${v.reviews_due} due` : ''}{v.needs_placement ? ' · place yourself first' : v.next_topic ? ` · next: ${v.next_topic.label}` : ''}
                  </p>
                  {v.due && (
                    <p className={`font-mono text-[10px] mt-0.5 ${v.days_left !== undefined && v.days_left < 14 ? 'text-mathua-amber' : 'text-mathua-muted'}`}>
                      Due {formatDue(v.due)}{v.days_left !== undefined ? ` · ${v.days_left < 0 ? 'past due' : `${v.days_left} days`}` : ''}{topicsPerWeek(v) !== null ? ` · ${topicsPerWeek(v)!.toFixed(1)} topics/week to finish` : ''}
                    </p>
                  )}
                </li>
              ))}
            </ul>
          </section>
        )}
      </main>
      <Footer />
      <BottomTabs />
    </>
  )
}
