'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import Header from '../../components/Header'
import BottomTabs from '../../components/BottomTabs'
import Footer from '../../components/Footer'
import Loading from '../../components/Loading'
import SectionHeader from '../../components/SectionHeader'
import { useTheme } from '../../hooks/useTheme'
import { getCourseCatalog, getCourseView, setActiveCourses, type CourseStatus, type CourseView } from '../../lib/api'
import { getUserInfo } from '../../lib/auth'
import { formatDue } from '../../lib/dates'

// Fork change: the dashboard. Your courses as cards, the way a course site
// shows them: each with its progress, what is due and the next topic, one
// click from the course's own page. The Study tab (/today) is the queue across
// all of them; this page is where you pick a course to work in.

const ACCENTS = ['bg-mathua-blue', 'bg-mathua-green', 'bg-amber-500', 'bg-purple-500', 'bg-rose-500', 'bg-teal-500']

function CourseCard({ course, view, accent, onUnfollow }: { course: CourseStatus; view: CourseView | null; accent: string; onUnfollow: () => void }) {
  const pct = view && view.total ? Math.round((view.mastered / view.total) * 100) : 0
  return (
    <li className="min-w-0 border border-mathua-border bg-mathua-surface flex flex-col hover:border-mathua-blue">
      <Link href={`/course?id=${encodeURIComponent(course.id)}`} className={`${accent} h-20 block relative`}>
        <span className="absolute left-3 bottom-2 font-mono text-[10px] uppercase tracking-wider text-white/90">{view?.term || course.grade}</span>
      </Link>
      <div className="p-3 flex-1 flex flex-col gap-2 min-w-0">
        <Link href={`/course?id=${encodeURIComponent(course.id)}`} className="font-serif text-base text-mathua-primary hover:text-mathua-blue break-words [overflow-wrap:anywhere] leading-snug">
          {course.name.replace(/: study plan$/, '')}
        </Link>
        {view ? (
          <>
            <div className="h-1.5 bg-mathua-code overflow-hidden" aria-label={`${pct}% mastered`}>
              <div className="h-full bg-mathua-blue" style={{ width: `${pct}%` }} />
            </div>
            <p className="font-mono text-[11px] text-mathua-secondary">
              {view.mastered} of {view.total} topics mastered{view.started > view.mastered ? ` · ${view.started - view.mastered} in progress` : ''}
            </p>
            {view.reviews_due > 0 && <p className="font-mono text-[11px] text-amber-600 dark:text-amber-400">{view.reviews_due} review{view.reviews_due === 1 ? '' : 's'} due</p>}
            {view.due && (
              <p className={`font-mono text-[11px] ${view.days_left !== undefined && view.days_left < 14 ? 'text-amber-600 dark:text-amber-400' : 'text-mathua-muted'}`}>
                Due {formatDue(view.due)}{view.days_left !== undefined ? ` · ${view.days_left < 0 ? 'past due' : `${view.days_left} days left`}` : ''}
              </p>
            )}
            {view.next_topic && (
              <p className="font-mono text-[11px] text-mathua-muted truncate" title={view.next_topic.label}>Next: {view.next_topic.label}</p>
            )}
          </>
        ) : (
          <p className="font-mono text-[11px] text-mathua-muted">{course.plan?.length ?? course.targets.length} topics</p>
        )}
        <div className="mt-auto pt-2 flex items-center gap-2 flex-wrap">
          {view?.next_topic && (
            <Link href={`/learn?concept=${encodeURIComponent(view.next_topic.id)}&course=${encodeURIComponent(course.id)}`} className="border border-mathua-blue text-mathua-blue hover:bg-mathua-blue-faint px-3 min-h-[32px] inline-flex items-center font-mono text-[11px]">
              Continue →
            </Link>
          )}
          <Link href={`/course?id=${encodeURIComponent(course.id)}`} className="border border-mathua-border text-mathua-secondary hover:border-mathua-blue hover:text-mathua-blue px-3 min-h-[32px] inline-flex items-center font-mono text-[11px]">
            Open
          </Link>
          <button type="button" onClick={onUnfollow} className="ml-auto font-mono text-[10px] text-mathua-muted hover:text-mathua-red">Unfollow</button>
        </div>
      </div>
    </li>
  )
}

export default function CoursesPage() {
  const { mounted } = useTheme()
  const [courses, setCourses] = useState<CourseStatus[] | null>(null)
  const [following, setFollowing] = useState<string[]>([])
  const [views, setViews] = useState<Record<string, CourseView | null>>({})
  const [showAll, setShowAll] = useState(false)
  const [busy, setBusy] = useState(false)
  const [signedIn, setSignedIn] = useState(false)

  useEffect(() => {
    setSignedIn(!!getUserInfo()?.student_id)
    getCourseCatalog().then(async ({ courses, activeIds }) => {
      setCourses(courses)
      setFollowing(activeIds)
      const vs = await Promise.all(activeIds.map((id) => getCourseView(id)))
      const map: Record<string, CourseView | null> = {}
      activeIds.forEach((id, i) => { map[id] = vs[i] })
      setViews(map)
    }).catch(() => setCourses([]))
  }, [])

  async function setFollow(ids: string[]) {
    setBusy(true)
    try {
      const saved = await setActiveCourses(ids)
      setFollowing(saved)
      const missing = saved.filter((id) => !(id in views))
      if (missing.length) {
        const vs = await Promise.all(missing.map((id) => getCourseView(id)))
        setViews((cur) => { const n = { ...cur }; missing.forEach((id, i) => { n[id] = vs[i] }); return n })
      }
    } finally {
      setBusy(false)
    }
  }

  if (!mounted || courses === null) return <Loading label="Loading your courses" full />
  const followed = following.map((id) => courses.find((c) => c.id === id)).filter((c): c is CourseStatus => !!c)
  const others = courses.filter((c) => !following.includes(c.id) && (c.plan?.length ?? 0) > 0)
  const grade = courses.filter((c) => !following.includes(c.id) && !(c.plan?.length ?? 0))

  return (
    <>
      <Header />
      <main className="mx-auto w-full max-w-[1100px] min-w-0 px-4 sm:px-6 py-8 sm:py-12 overflow-x-hidden">
        <SectionHeader label="Dashboard" title="Your courses" />
        {!signedIn && (
          <p className="mb-4 border border-mathua-border bg-mathua-surface p-3 font-mono text-[11px] text-mathua-secondary">
            Sign in to follow courses; progress on a guest account is not kept.
          </p>
        )}
        {followed.length === 0 ? (
          <p className="font-mono text-xs text-mathua-secondary mb-6">You follow no courses yet. Pick the ones you are taking below; the Study tab then works through all of them.</p>
        ) : (
          <>
            <ul className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 mb-8">
              {followed.map((c, i) => (
                <CourseCard key={c.id} course={c} view={views[c.id] ?? null} accent={ACCENTS[i % ACCENTS.length]} onUnfollow={() => void setFollow(following.filter((x) => x !== c.id))} />
              ))}
            </ul>
            <div className="mb-8 flex flex-wrap gap-2">
              <Link href="/today" className="border border-mathua-blue text-mathua-blue hover:bg-mathua-blue-faint px-4 min-h-[36px] inline-flex items-center font-mono text-xs">Study: what&apos;s next across your courses →</Link>
              <Link href="/library" className="border border-mathua-border text-mathua-secondary hover:border-mathua-blue hover:text-mathua-blue px-4 min-h-[36px] inline-flex items-center font-mono text-xs">Library</Link>
            </div>
          </>
        )}

        <h2 className="font-mono text-[11px] uppercase tracking-[0.15em] text-mathua-muted mb-2">Study plans you can follow</h2>
        <ul className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3 mb-6">
          {others.map((c) => (
            <li key={c.id} className="min-w-0 border border-mathua-border bg-mathua-surface p-3 flex flex-col gap-1">
              <Link href={`/course?id=${encodeURIComponent(c.id)}`} className="font-mono text-xs text-mathua-primary hover:text-mathua-blue break-words [overflow-wrap:anywhere]">{c.name.replace(/: study plan$/, '')}</Link>
              <p className="font-mono text-[10px] text-mathua-muted line-clamp-2">{c.description}</p>
              <button type="button" disabled={busy || !signedIn} onClick={() => void setFollow([...following, c.id])} className="mt-auto self-start border border-mathua-border text-mathua-secondary hover:border-mathua-blue hover:text-mathua-blue px-3 min-h-[30px] font-mono text-[11px] disabled:opacity-40">
                Follow
              </button>
            </li>
          ))}
        </ul>
        {grade.length > 0 && (
          <details open={showAll} onToggle={(e) => setShowAll((e.target as HTMLDetailsElement).open)}>
            <summary className="font-mono text-[11px] uppercase tracking-[0.15em] text-mathua-muted cursor-pointer mb-2">Grade-level math courses ({grade.length})</summary>
            <ul className="flex flex-wrap gap-2">
              {grade.map((c) => (
                <li key={c.id}>
                  <button type="button" disabled={busy || !signedIn} onClick={() => void setFollow([...following, c.id])} className="border border-mathua-border text-mathua-secondary hover:border-mathua-blue hover:text-mathua-blue px-3 min-h-[30px] font-mono text-[11px] disabled:opacity-40">
                    {c.name}
                  </button>
                </li>
              ))}
            </ul>
          </details>
        )}
      </main>
      <Footer />
      <BottomTabs />
    </>
  )
}
