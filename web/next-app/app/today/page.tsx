'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import Header from '../../components/Header'
import BottomTabs from '../../components/BottomTabs'
import Footer from '../../components/Footer'
import Loading from '../../components/Loading'
import SectionHeader from '../../components/SectionHeader'
import { useTheme } from '../../hooks/useTheme'
import { getCourseCatalog, getCourseView, getNext, getScores, type CourseView, type Recommendation } from '../../lib/api'
import { getUserInfo } from '../../lib/auth'
import { hrefConceptId } from '../../lib/nextUp'

// Fork change: the Study tab. One queue across every course you follow, the
// way Math Academy opens on "today": the engine's next task first, then the
// alternatives, each tagged with the course it belongs to, with the mastery
// test and the review session surfaced when they are due. Nothing here is
// decided in the browser: the order is /api/next's, the counts are the server's.

type Row = Recommendation & { courses: string[] }

export default function TodayPage() {
  const { mounted } = useTheme()
  const [rows, setRows] = useState<Row[] | null>(null)
  const [views, setViews] = useState<CourseView[]>([])
  const [quizDue, setQuizDue] = useState(false)
  const [xp, setXp] = useState<{ since: number; gate: number } | null>(null)
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    const sid = getUserInfo()?.student_id
    ;(async () => {
      try {
        const [{ activeIds }, next] = await Promise.all([getCourseCatalog(), getNext()])
        const vs = (await Promise.all(activeIds.map((id) => getCourseView(id)))).filter((v): v is CourseView => !!v)
        setViews(vs)
        const byConcept = new Map<string, string[]>()
        for (const v of vs) {
          const name = v.name.replace(/: study plan$/, '')
          for (const m of v.modules) for (const t of m.topics) byConcept.set(t.id, [...(byConcept.get(t.id) ?? []), name])
        }
        const list = [next.primary, ...next.alternatives].filter((r): r is Recommendation => !!r)
        setRows(list.map((r) => ({ ...r, courses: byConcept.get(r.conceptId || hrefConceptId(r.action.href) || '') ?? [] })))
        if (sid) {
          const s = await getScores(sid).catch(() => null)
          if (s) {
            setQuizDue(!!s.quiz_due)
            if (s.quiz_gate_xp) setXp({ since: s.xp_since_quiz ?? 0, gate: s.quiz_gate_xp })
          }
        }
      } catch {
        setFailed(true)
        setRows([])
      }
    })()
  }, [])

  if (!mounted || rows === null) return <Loading label="Working out what to study" full />
  const reviewsDue = views.reduce((n, v) => n + v.reviews_due, 0)
  const kindLabel = (r: Recommendation) => r.kind === 'review' ? 'Review' : r.kind === 'practice' ? 'Practise' : r.kind === 'mastery_check' ? 'Mastery test' : r.reason === 'prerequisite' ? 'Prerequisite' : 'New topic'

  return (
    <>
      <Header />
      <main className="mx-auto w-full max-w-[900px] min-w-0 px-4 sm:px-6 py-8 sm:py-12 overflow-x-hidden">
        <SectionHeader label="Study" title="Today" />
        {views.length === 0 && (
          <p className="mb-4 border border-mathua-border bg-mathua-surface p-3 font-mono text-[11px] text-mathua-secondary">
            You follow no courses, so this queue draws from the whole track. <Link href="/courses" className="underline">Pick your courses</Link> to focus it.
          </p>
        )}
        {failed && <p className="mb-4 font-mono text-xs text-mathua-red">The queue could not be loaded. Check the server and try again.</p>}

        {(quizDue || reviewsDue > 0) && (
          <div className="mb-6 grid gap-2 sm:grid-cols-2">
            {quizDue && (
              <Link href="/goals?quiz=1" className="border border-mathua-blue bg-mathua-surface p-3 hover:bg-mathua-blue-faint">
                <p className="font-mono text-[10px] uppercase tracking-wider text-mathua-blue">Mastery test due</p>
                <p className="font-mono text-xs text-mathua-primary mt-1">Covers what your courses have taught you{xp ? ` · ${xp.since} XP since the last one` : ''}. Take it →</p>
              </Link>
            )}
            {reviewsDue > 0 && (
              <Link href="/review" className="border border-mathua-border bg-mathua-surface p-3 hover:border-mathua-blue">
                <p className="font-mono text-[10px] uppercase tracking-wider text-amber-600 dark:text-amber-400">Reviews due</p>
                <p className="font-mono text-xs text-mathua-primary mt-1">{reviewsDue} topic{reviewsDue === 1 ? '' : 's'} across your courses. Review →</p>
              </Link>
            )}
          </div>
        )}

        <ol className="space-y-2">
          {rows.map((r, i) => (
            <li key={r.id + i} className={`border bg-mathua-surface p-3 flex flex-wrap items-center gap-x-3 gap-y-2 min-w-0 ${i === 0 ? 'border-mathua-blue' : 'border-mathua-border'}`}>
              <div className="min-w-0 flex-1 basis-56">
                <p className="font-mono text-[10px] uppercase tracking-wider text-mathua-muted">
                  {i === 0 ? 'Up next' : kindLabel(r)}{r.courses.length ? ` · ${r.courses.join(', ')}` : ''}
                </p>
                <p className="font-serif text-base text-mathua-primary break-words [overflow-wrap:anywhere]">{r.conceptTitle || (r.id === 'diagnostic' ? 'Placement test' : r.id === 'paused' ? 'Paused placement test' : r.id)}</p>
                {r.detail && <p className="font-mono text-[11px] text-mathua-secondary mt-0.5 break-words [overflow-wrap:anywhere]">{r.detail}</p>}
              </div>
              <div className="flex items-center gap-2 ml-auto">
                {r.xp ? <span className="font-mono text-[10px] text-mathua-muted">+{r.xp} XP</span> : null}
                <Link href={r.action.href} className={`border px-3 min-h-[34px] inline-flex items-center font-mono text-[11px] ${i === 0 ? 'border-mathua-blue text-mathua-blue hover:bg-mathua-blue-faint' : 'border-mathua-border text-mathua-secondary hover:border-mathua-blue hover:text-mathua-blue'}`}>
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
                    {v.mastered}/{v.total} mastered{v.reviews_due ? ` · ${v.reviews_due} due` : ''}{v.next_topic ? ` · next: ${v.next_topic.label}` : ''}
                  </p>
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
