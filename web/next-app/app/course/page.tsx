'use client'

import { Suspense, useEffect, useMemo, useState } from 'react'
import Link from 'next/link'
import { useRouter, useSearchParams } from 'next/navigation'
import Header from '../../components/Header'
import BottomTabs from '../../components/BottomTabs'
import Footer from '../../components/Footer'
import Loading from '../../components/Loading'
import PdfReader from '../../components/PdfReader'
import VideoPlayer, { formatDuration } from '../../components/VideoPlayer'
import { useTheme } from '../../hooks/useTheme'
import { getCourseView, setActiveCourses, getCourseCatalog, getPins, setPins, type CourseNote, type CourseTopic, type CourseView, type CourseVideo } from '../../lib/api'
import { API_BASE } from '../../lib/apiBase'
import { getUserInfo } from '../../lib/auth'
import { formatDue } from '../../lib/dates'

// Fork change: a course's own page, laid out like a course site: Home (what
// to do next, progress), Modules (its topics in teaching order, each with its
// state, reading and videos), Readings (the course's texts), Videos (its
// lectures) and Grades (mastery per module). Every topic link carries
// ?course= so the lesson opens in this course's text.

type Tab = 'home' | 'modules' | 'readings' | 'videos' | 'grades'
const TABS: { id: Tab; label: string }[] = [
  { id: 'home', label: 'Home' },
  { id: 'modules', label: 'Modules' },
  { id: 'readings', label: 'Readings' },
  { id: 'videos', label: 'Lectures' },
  { id: 'grades', label: 'Grades' },
]

// A lecture page served by the module files endpoint is a bare path; a link
// stays as given.
function noteHref(n: CourseNote): string {
  return n.url.startsWith('/') ? `${API_BASE}${n.url}` : n.url
}

function statusWord(t: CourseTopic): string {
  if (t.status === 'mastered') return 'Mastered'
  if (t.due) return 'Review due'
  if (t.attempts > 0) return t.status === 'practicing' ? 'Practicing' : 'Learning'
  if (t.locked) return 'Locked'
  return 'Not started'
}

function statusClass(t: CourseTopic): string {
  if (t.status === 'mastered') return 'text-mathua-green'
  if (t.due) return 'text-amber-600 dark:text-amber-400'
  if (t.attempts > 0) return 'text-mathua-blue'
  return 'text-mathua-muted'
}

function TopicRow({ t, courseId, onRead, onWatch, pinned, onPin }: { t: CourseTopic; courseId: string; onRead: (t: CourseTopic) => void; onWatch: (v: CourseVideo[]) => void; pinned?: boolean; onPin?: (id: string, on: boolean) => void }) {
  const learn = `/learn?concept=${encodeURIComponent(t.id)}&course=${encodeURIComponent(courseId)}`
  const btn = 'shrink-0 border px-2.5 min-h-[30px] inline-flex items-center font-mono text-[11px]'
  return (
    <li className={`py-2 px-3 flex flex-wrap items-center gap-x-3 gap-y-1 min-w-0 ${pinned ? 'bg-mathua-blue-faint' : ''}`}>
      {onPin && t.status !== 'mastered' ? (
        <button type="button" onClick={() => onPin(t.id, !pinned)} aria-pressed={!!pinned} aria-label={pinned ? `Unpin ${t.label} from this week` : `Pin ${t.label} for this week`} title={pinned ? 'Pinned for this week — click to unpin' : 'Pin for this week: keep it in your Study queue'}
          className={`shrink-0 w-6 h-6 inline-flex items-center justify-center font-mono text-[13px] leading-none border ${pinned ? 'border-mathua-blue text-mathua-blue bg-mathua-surface' : 'border-transparent text-mathua-muted hover:text-mathua-blue hover:border-mathua-border'}`}>
          {pinned ? '●' : '○'}
        </button>
      ) : (
        <span className={`w-2 h-2 shrink-0 rounded-full ${t.status === 'mastered' ? 'bg-mathua-green' : t.attempts > 0 ? 'bg-mathua-blue' : t.locked ? 'bg-mathua-border' : 'bg-mathua-muted'}`} aria-hidden />
      )}
      <Link href={`/study?concept=${encodeURIComponent(t.id)}`} className="min-w-0 flex-1 basis-40 font-mono text-xs text-mathua-primary hover:text-mathua-blue break-words [overflow-wrap:anywhere]">{t.label}</Link>
      <span className={`font-mono text-[10px] ${statusClass(t)}`}>{statusWord(t)}</span>
      {t.cards ? <span className="font-mono text-[10px] text-mathua-muted" title="Your flashcards cover this topic">{t.cards} deck{t.cards === 1 ? '' : 's'}</span> : null}
      <span className="flex items-center gap-1.5 ml-auto">
        {t.reading && <button type="button" onClick={() => onRead(t)} className={`${btn} border-mathua-border text-mathua-secondary hover:border-mathua-blue hover:text-mathua-blue`}>Read</button>}
        {t.videos && t.videos.length > 0 && <button type="button" onClick={() => onWatch(t.videos!)} className={`${btn} border-mathua-border text-mathua-secondary hover:border-mathua-blue hover:text-mathua-blue`}>Watch</button>}
        {t.notes && t.notes.length > 0 && (
          <a href={noteHref(t.notes[0])} target="_blank" rel="noreferrer" title={t.notes.map((n) => n.title).join(' · ')} className={`${btn} border-mathua-border text-mathua-secondary hover:border-mathua-blue hover:text-mathua-blue`}>
            Notes{t.notes.length > 1 ? ` (${t.notes.length})` : ''}
          </a>
        )}
        <Link href={learn} className={`${btn} border-mathua-blue text-mathua-blue hover:bg-mathua-blue-faint`}>{t.due ? 'Review' : t.attempts > 0 ? 'Practice' : 'Start'}</Link>
      </span>
    </li>
  )
}

function CourseContent() {
  const { mounted } = useTheme()
  const router = useRouter()
  const params = useSearchParams()
  const id = params.get('id') ?? ''
  const tab = (params.get('tab') as Tab) || 'home'
  const [view, setView] = useState<CourseView | null | undefined>(undefined)
  const [reading, setReading] = useState<CourseTopic | null>(null)
  const [watching, setWatching] = useState<CourseVideo[] | null>(null)
  const [busy, setBusy] = useState(false)
  const [signedIn, setSignedIn] = useState(false)
  const [pins, setPinsState] = useState<Set<string>>(new Set())

  useEffect(() => {
    const sid = getUserInfo()?.student_id
    setSignedIn(!!sid)
    if (!id) { setView(null); return }
    getCourseView(id).then(setView).catch(() => setView(null))
    if (sid) getPins().then((ps) => setPinsState(new Set(ps.map((p) => p.concept)))).catch(() => {})
  }, [id])

  // Fork change: pin a topic for this week. The Study tab serves pinned
  // topics right after due reviews; the engine fills the rest of the queue.
  const togglePin = async (cid: string, on: boolean) => {
    const next = new Set(pins)
    if (on) next.add(cid)
    else next.delete(cid)
    setPinsState(next)
    try {
      const saved = await setPins([...next])
      setPinsState(new Set(saved.map((p) => p.concept)))
    } catch {
      setPinsState(pins)
    }
  }

  // Lecture notes grouped by unit, in the server's order (fork change).
  const noteUnits = useMemo(() => {
    const out: [string, CourseNote[]][] = []
    for (const n of view?.notes ?? []) {
      const last = out[out.length - 1]
      if (last && last[0] === (n.unit ?? '')) last[1].push(n)
      else out.push([n.unit ?? '', [n]])
    }
    return out
  }, [view])
  const topicLabel = useMemo(() => {
    const m = new Map<string, string>()
    for (const mod of view?.modules ?? []) for (const t of mod.topics) m.set(t.id, t.label)
    for (const t of view?.prerequisites ?? []) m.set(t.id, t.label)
    return m
  }, [view])
  const allVideos = useMemo(() => {
    if (!view) return []
    const out: { topic?: string; v: CourseVideo }[] = view.videos.map((v) => ({ v }))
    for (const m of view.modules) for (const t of m.topics) for (const v of t.videos ?? []) out.push({ topic: t.label, v })
    return out
  }, [view])

  function go(t: Tab) {
    router.replace(`/course?id=${encodeURIComponent(id)}${t === 'home' ? '' : `&tab=${t}`}`)
  }

  async function toggleFollow() {
    if (!view) return
    setBusy(true)
    try {
      const { activeIds } = await getCourseCatalog()
      const next = view.followed ? activeIds.filter((x) => x !== view.id) : [...activeIds, view.id]
      await setActiveCourses(next)
      setView({ ...view, followed: !view.followed })
    } finally {
      setBusy(false)
    }
  }

  if (!mounted || view === undefined) return <Loading label="Loading course" full />
  if (!view) {
    return (
      <>
        <Header />
        <main className="mx-auto w-full max-w-[1100px] px-4 sm:px-6 py-12">
          <p className="font-mono text-sm text-mathua-secondary">That course was not found. <Link href="/courses" className="underline">Back to your courses</Link>.</p>
        </main>
        <Footer />
        <BottomTabs />
      </>
    )
  }

  const pct = view.total ? Math.round((view.mastered / view.total) * 100) : 0
  const name = view.name.replace(/: study plan$/, '')
  const tabBtn = (t: Tab) => `px-3 min-h-[36px] inline-flex items-center font-mono text-xs border-b-2 ${tab === t ? 'border-mathua-blue text-mathua-blue' : 'border-transparent text-mathua-secondary hover:text-mathua-blue'}`

  return (
    <>
      <Header />
      <main className="mx-auto w-full max-w-[1100px] min-w-0 px-4 sm:px-6 py-8 sm:py-12 overflow-x-hidden">
        <Link href="/courses" className="font-mono text-xs text-mathua-secondary hover:text-mathua-blue">← Your courses</Link>
        <div className="mt-3 mb-4 flex flex-wrap items-start justify-between gap-3">
          <div className="min-w-0">
            {(view.term || view.due) && (
              <p className="font-mono text-[11px] uppercase tracking-[0.15em] text-mathua-muted">
                {view.term}{view.due ? `${view.term ? ' · ' : ''}due ${formatDue(view.due)}${view.days_left !== undefined ? ` (${view.days_left < 0 ? 'past' : `${view.days_left} days`})` : ''}` : ''}
              </p>
            )}
            <h1 className="font-serif text-2xl text-mathua-primary break-words [overflow-wrap:anywhere]">{name}</h1>
          </div>
          <button type="button" disabled={busy || !signedIn} onClick={() => void toggleFollow()} className="border border-mathua-border text-mathua-secondary hover:border-mathua-blue hover:text-mathua-blue px-3 min-h-[32px] font-mono text-[11px] disabled:opacity-40">
            {view.followed ? 'Following ✓' : 'Follow this course'}
          </button>
        </div>
        <nav aria-label="Course sections" className="flex gap-1 border-b border-mathua-border mb-6 overflow-x-auto">
          {TABS.map((t) => <button key={t.id} type="button" onClick={() => go(t.id)} className={tabBtn(t.id)} aria-current={tab === t.id}>{t.label}</button>)}
        </nav>

        {tab === 'home' && (
          <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_300px]">
            <div className="min-w-0 space-y-6">
              {view.needs_placement && (
                <section className="border border-mathua-blue bg-mathua-blue-faint p-4">
                  <p className="font-mono text-[10px] uppercase tracking-wider text-mathua-blue">Start here</p>
                  <p className="font-serif text-lg text-mathua-primary mt-1">Place yourself in this course</p>
                  <p className="font-mono text-[11px] text-mathua-secondary mt-1">
                    The plan rests on {view.prerequisites.length} prerequisite topics you have not shown yet. A short placement test finds where you stand so you start at your frontier instead of at the bottom.
                  </p>
                  <Link href={`/onboard?course=${encodeURIComponent(view.id)}`} className="mt-3 border border-mathua-blue text-mathua-blue bg-mathua-surface hover:bg-mathua-blue-faint px-4 min-h-[36px] inline-flex items-center font-mono text-xs">
                    Take the placement test →
                  </Link>
                </section>
              )}
              {view.next_topic ? (
                <section className="border border-mathua-blue bg-mathua-surface p-4">
                  <p className="font-mono text-[10px] uppercase tracking-wider text-mathua-blue">{view.needs_placement ? 'Or start from the bottom' : 'Up next in this course'}</p>
                  <p className="font-serif text-lg text-mathua-primary mt-1">{view.next_topic.label}</p>
                  {view.next_topic.reading && <p className="font-mono text-[11px] text-mathua-secondary mt-1">Read: {view.next_topic.reading.source}</p>}
                  <div className="mt-3 flex flex-wrap gap-2">
                    <Link href={`/learn?concept=${encodeURIComponent(view.next_topic.id)}&course=${encodeURIComponent(view.id)}`} className="border border-mathua-blue text-mathua-blue hover:bg-mathua-blue-faint px-4 min-h-[36px] inline-flex items-center font-mono text-xs">
                      {view.next_topic.attempts > 0 ? 'Continue →' : 'Start →'}
                    </Link>
                    {view.next_topic.reading && <button type="button" onClick={() => { setReading(view.next_topic!); go('readings') }} className="border border-mathua-border text-mathua-secondary hover:border-mathua-blue hover:text-mathua-blue px-4 min-h-[36px] inline-flex items-center font-mono text-xs">Read first</button>}
                  </div>
                </section>
              ) : (
                <section className="border border-mathua-border bg-mathua-surface p-4 font-mono text-xs text-mathua-secondary">Every open topic in this course is mastered. Reviews keep them that way.</section>
              )}
              {view.reviews_due > 0 && (
                <Link href="/review" className="block border border-mathua-border bg-mathua-surface p-4 font-mono text-xs text-mathua-primary hover:border-mathua-blue">
                  {view.reviews_due} topic{view.reviews_due === 1 ? '' : 's'} in this course {view.reviews_due === 1 ? 'is' : 'are'} due a review →
                </Link>
              )}
              <section>
                <h2 className="font-mono text-[11px] uppercase tracking-[0.15em] text-mathua-muted mb-2">About</h2>
                <p className="font-mono text-xs text-mathua-secondary leading-relaxed">{view.description}</p>
              </section>
              {view.videos.length > 0 && <VideoPlayer videos={view.videos} title="Course videos" />}
            </div>
            <aside className="space-y-4 min-w-0">
              <section className="border border-mathua-border bg-mathua-surface p-3">
                <p className="font-mono text-[10px] uppercase tracking-wider text-mathua-muted">Progress</p>
                <div className="mt-2 h-2 bg-mathua-code overflow-hidden"><div className="h-full bg-mathua-blue" style={{ width: `${pct}%` }} /></div>
                <p className="font-mono text-[11px] text-mathua-secondary mt-1">{view.mastered} of {view.total} mastered · {view.started} started</p>
              </section>
              <section className="border border-mathua-border bg-mathua-surface p-3">
                <p className="font-mono text-[10px] uppercase tracking-wider text-mathua-muted">Texts</p>
                <ul className="mt-1 space-y-1">
                  {view.texts.map((t) => (
                    <li key={t.key} className="font-mono text-[11px] min-w-0">
                      {t.library_id ? <Link href={`/library?book=${encodeURIComponent(t.library_id)}`} className="text-mathua-primary hover:text-mathua-blue break-words [overflow-wrap:anywhere]">{t.title}</Link> : <span className="text-mathua-secondary">{t.title}{t.kind === 'web' ? ' (online)' : ''}</span>}
                    </li>
                  ))}
                  {view.texts.length === 0 && <li className="font-mono text-[11px] text-mathua-muted">No texts set.</li>}
                </ul>
              </section>
              <section className="border border-mathua-border bg-mathua-surface p-3">
                <p className="font-mono text-[10px] uppercase tracking-wider text-mathua-muted">Modules</p>
                <ol className="mt-1 space-y-1">
                  {view.modules.map((m, i) => (
                    <li key={i} className="font-mono text-[11px] flex justify-between gap-2 min-w-0">
                      <button type="button" onClick={() => go('modules')} className="text-left text-mathua-primary hover:text-mathua-blue truncate">{m.title}</button>
                      <span className="text-mathua-muted shrink-0">{m.mastered}/{m.topics.length}</span>
                    </li>
                  ))}
                </ol>
              </section>
            </aside>
          </div>
        )}

        {tab === 'modules' && (
          <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,420px)]">
            <div className="min-w-0 space-y-4">
              {signedIn && (
                <p className="font-mono text-[11px] text-mathua-secondary">
                  Pin this week&apos;s topics with <span className="text-mathua-muted">○</span> → <span className="text-mathua-blue">●</span>. Pinned topics lead your <Link href="/today" className="underline hover:text-mathua-blue">Study</Link> queue after due reviews{pins.size ? ` · ${pins.size} pinned` : ''}.
                </p>
              )}
              {view.prerequisites.length > 0 && (
                <details className="border border-mathua-border bg-mathua-surface">
                  <summary className="cursor-pointer px-3 py-2.5 flex items-center justify-between gap-2 font-mono text-xs text-mathua-secondary">
                    <span>Prerequisites the plan needs ({view.prerequisites.length} not yet mastered)</span>
                    {view.needs_placement && <Link href={`/onboard?course=${encodeURIComponent(view.id)}`} className="shrink-0 text-mathua-blue hover:underline text-[11px]">Place me →</Link>}
                  </summary>
                  <ol className="border-t border-mathua-border divide-y divide-mathua-border max-h-[50vh] overflow-auto">
                    {view.prerequisites.map((t) => <TopicRow key={t.id} t={t} courseId={view.id} onRead={setReading} onWatch={setWatching} pinned={pins.has(t.id)} onPin={signedIn ? togglePin : undefined} />)}
                  </ol>
                </details>
              )}
              {view.modules.map((m, i) => (
                <details key={i} open={i === 0 || m.topics.some((t) => t.attempts > 0 && t.status !== 'mastered')} className="border border-mathua-border bg-mathua-surface">
                  <summary className="cursor-pointer px-3 py-2.5 flex items-center justify-between gap-2 font-mono text-xs text-mathua-primary">
                    <span className="min-w-0 break-words [overflow-wrap:anywhere]">{m.title}</span>
                    <span className="shrink-0 text-mathua-muted text-[11px]">{m.mastered}/{m.topics.length}</span>
                  </summary>
                  <ol className="border-t border-mathua-border divide-y divide-mathua-border">
                    {m.topics.map((t) => <TopicRow key={t.id} t={t} courseId={view.id} onRead={setReading} onWatch={setWatching} pinned={pins.has(t.id)} onPin={signedIn ? togglePin : undefined} />)}
                  </ol>
                </details>
              ))}
            </div>
            <div className="min-w-0 lg:sticky lg:top-4 self-start space-y-4">
              {reading?.reading && (reading.reading.kind === 'pdf' ? (
                <PdfReader reading={{ title: reading.reading.source, url: reading.reading.url ?? '', page: reading.reading.page, end_page: reading.reading.end_page }} />
              ) : (
                <div className="border border-mathua-border bg-mathua-bg p-3 font-mono text-xs">
                  <p className="text-mathua-muted text-[10px] uppercase tracking-wider">Read</p>
                  <p className="text-mathua-primary">{reading.reading.source}</p>
                  {reading.reading.url && <a href={reading.reading.url} target="_blank" rel="noreferrer" className="underline text-mathua-secondary">Open ↗</a>}
                </div>
              ))}
              {watching && <VideoPlayer videos={watching} />}
              {!reading && !watching && <p className="font-mono text-[11px] text-mathua-muted hidden lg:block">Pick Read or Watch on a topic to open it here.</p>}
            </div>
          </div>
        )}

        {tab === 'readings' && (
          <div className="grid gap-4 lg:grid-cols-[minmax(0,360px)_minmax(0,1fr)]">
            <div className="min-w-0 space-y-4">
              <section>
                <h2 className="font-mono text-[11px] uppercase tracking-[0.15em] text-mathua-muted mb-2">Course texts</h2>
                <ul className="border border-mathua-border divide-y divide-mathua-border bg-mathua-surface">
                  {view.texts.map((t) => (
                    <li key={t.key} className="px-3 py-2 font-mono text-xs min-w-0">
                      {t.library_id ? <Link href={`/library?book=${encodeURIComponent(t.library_id)}`} className="text-mathua-primary hover:text-mathua-blue break-words [overflow-wrap:anywhere]">{t.title} →</Link> : <span className="text-mathua-secondary">{t.title} (online)</span>}
                    </li>
                  ))}
                </ul>
              </section>
              <section>
                <h2 className="font-mono text-[11px] uppercase tracking-[0.15em] text-mathua-muted mb-2">Reading by topic</h2>
                <ol className="border border-mathua-border divide-y divide-mathua-border bg-mathua-surface lg:max-h-[70vh] lg:overflow-auto">
                  {view.modules.flatMap((m) => m.topics).filter((t) => t.reading).map((t) => (
                    <li key={t.id} className={`px-3 py-2 min-w-0 ${reading?.id === t.id ? 'bg-mathua-blue-faint' : ''}`}>
                      <button type="button" onClick={() => setReading(t)} className="text-left w-full min-w-0">
                        <span className="block font-mono text-xs text-mathua-primary break-words [overflow-wrap:anywhere]">{t.label}</span>
                        <span className="block font-mono text-[10px] text-mathua-muted break-words [overflow-wrap:anywhere]">{t.reading!.source}</span>
                      </button>
                    </li>
                  ))}
                </ol>
              </section>
            </div>
            <div className="min-w-0">
              {reading?.reading ? (reading.reading.kind === 'pdf' ? (
                <PdfReader reading={{ title: reading.reading.source, url: reading.reading.url ?? '', page: reading.reading.page, end_page: reading.reading.end_page }} />
              ) : (
                <div className="border border-mathua-border bg-mathua-bg p-3 font-mono text-xs">
                  <p className="text-mathua-primary">{reading.reading.source}</p>
                  {reading.reading.url && <a href={reading.reading.url} target="_blank" rel="noreferrer" className="underline text-mathua-secondary">Open ↗</a>}
                </div>
              )) : <p className="font-mono text-[11px] text-mathua-muted">Pick a topic to open its reading here.</p>}
            </div>
          </div>
        )}

        {tab === 'videos' && view.notes.length > 0 && (
          <section className="mb-6">
            <h2 className="font-mono text-[11px] uppercase tracking-[0.15em] text-mathua-muted mb-2">Lecture notes</h2>
            <div className="space-y-2">
              {noteUnits.map(([unit, notes]) => (
                <details key={unit} className="border border-mathua-border bg-mathua-surface">
                  <summary className="cursor-pointer px-3 py-2 flex items-center justify-between gap-2 font-mono text-xs text-mathua-primary">
                    <span className="min-w-0 break-words [overflow-wrap:anywhere]">{unit || 'Notes'}</span>
                    <span className="shrink-0 text-mathua-muted text-[11px]">{notes.length} page{notes.length === 1 ? '' : 's'}</span>
                  </summary>
                  <ol className="border-t border-mathua-border divide-y divide-mathua-border">
                    {notes.map((n, i) => {
                      const topic = n.topic ? topicLabel.get(n.topic) : undefined
                      return (
                        <li key={i} className="px-3 py-1.5 font-mono text-[11px] flex flex-wrap items-center gap-x-3 gap-y-1 min-w-0">
                          <a href={noteHref(n)} target="_blank" rel="noreferrer" className="min-w-0 flex-1 basis-48 text-mathua-primary hover:text-mathua-blue break-words [overflow-wrap:anywhere]">{n.seq ? `${n.seq}. ` : ''}{n.title} ↗</a>
                          {topic ? <Link href={`/learn?concept=${encodeURIComponent(n.topic!)}&course=${encodeURIComponent(view.id)}`} className="text-mathua-muted hover:text-mathua-blue truncate max-w-[50%]" title={topic}>{topic}</Link> : <span className="text-mathua-muted">whole course</span>}
                        </li>
                      )
                    })}
                  </ol>
                </details>
              ))}
            </div>
          </section>
        )}
        {tab === 'videos' && (
          allVideos.length === 0 ? (
            <div className="border border-mathua-border bg-mathua-surface p-4 font-mono text-xs text-mathua-secondary leading-relaxed">
              <p>No lecture videos yet.</p>
              <p className="mt-2 text-mathua-muted">When the course&apos;s module lists lectures (<code>videos.json</code>, see FORK.md), they play here.</p>
            </div>
          ) : (
            <div className="space-y-4">
              <VideoPlayer videos={allVideos.map((x) => x.v)} title="Lectures" />
              <ol className="border border-mathua-border divide-y divide-mathua-border bg-mathua-surface">
                {allVideos.map((x, i) => (
                  <li key={i} className="px-3 py-2 font-mono text-[11px] flex justify-between gap-2 min-w-0">
                    <span className="min-w-0 break-words [overflow-wrap:anywhere] text-mathua-primary">{x.v.title}{x.topic ? <span className="text-mathua-muted"> · {x.topic}</span> : null}</span>
                    <span className="text-mathua-muted shrink-0">{formatDuration(x.v.duration)}</span>
                  </li>
                ))}
              </ol>
            </div>
          )
        )}

        {tab === 'grades' && (
          <table className="w-full border border-mathua-border bg-mathua-surface font-mono text-xs">
            <thead>
              <tr className="text-left text-[10px] uppercase tracking-wider text-mathua-muted border-b border-mathua-border">
                <th className="px-3 py-2 font-normal">Module</th>
                <th className="px-3 py-2 font-normal text-right">Mastered</th>
                <th className="px-3 py-2 font-normal text-right">Started</th>
                <th className="px-3 py-2 font-normal text-right">Due</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-mathua-border">
              {view.modules.map((m, i) => (
                <tr key={i}>
                  <td className="px-3 py-2 text-mathua-primary break-words [overflow-wrap:anywhere]">{m.title}</td>
                  <td className="px-3 py-2 text-right tabular-nums">{m.mastered}/{m.topics.length}</td>
                  <td className="px-3 py-2 text-right tabular-nums text-mathua-secondary">{m.topics.filter((t) => t.attempts > 0).length}</td>
                  <td className="px-3 py-2 text-right tabular-nums text-mathua-secondary">{m.topics.filter((t) => t.due).length}</td>
                </tr>
              ))}
              <tr className="border-t border-mathua-border text-mathua-primary">
                <td className="px-3 py-2">Total</td>
                <td className="px-3 py-2 text-right tabular-nums">{view.mastered}/{view.total} ({pct}%)</td>
                <td className="px-3 py-2 text-right tabular-nums">{view.started}</td>
                <td className="px-3 py-2 text-right tabular-nums">{view.reviews_due}</td>
              </tr>
            </tbody>
          </table>
        )}
      </main>
      <Footer />
      <BottomTabs />
    </>
  )
}

export default function CoursePage() {
  return (
    <Suspense fallback={<Loading label="Loading course" full />}>
      <CourseContent />
    </Suspense>
  )
}
