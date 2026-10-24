'use client'

import { Suspense, useEffect, useMemo, useRef, useState } from 'react'
import Link from 'next/link'
import { useRouter, useSearchParams } from 'next/navigation'
import Header from '../../components/Header'
import BottomTabs from '../../components/BottomTabs'
import Footer from '../../components/Footer'
import Loading from '../../components/Loading'
import SectionHeader from '../../components/SectionHeader'
import PdfReader from '../../components/PdfReader'
import { useTheme } from '../../hooks/useTheme'
import { getLibrary, getProgress, type ConceptProgress, type LibraryBook, type LibrarySection } from '../../lib/api'
import { getUserInfo } from '../../lib/auth'
import { masteryState } from '../../lib/progress'
import { learnConceptPath, studyConceptPath } from '../../lib/learnReturn'
import { TRACKS } from '../../lib/track'

// Fork change: the library. Every book and source your topics are drawn from,
// shelved by track. Open a book for its table of contents: each section lists
// the topics it teaches (each a click from its lesson and practice), and a PDF
// opens beside it at that section's page.

function pages(s: LibrarySection): string {
  if (!s.page) return ''
  return s.end_page && s.end_page !== s.page ? `PDF pp. ${s.page}–${s.end_page}` : `PDF p. ${s.page}`
}

function BookView({ book, progress, onBack }: { book: LibraryBook; progress: Record<string, ConceptProgress>; onBack: () => void }) {
  const [current, setCurrent] = useState<number>(book.kind === 'pdf' && book.sections.length ? 0 : -1)
  const [filter, setFilter] = useState('')
  const readerRef = useRef<HTMLDivElement>(null)
  const sec = current >= 0 ? book.sections[current] : null
  const needle = filter.trim().toLowerCase()
  const sections = book.sections
    .map((s, i) => ({ s, i }))
    .filter(({ s }) => !needle || s.title.toLowerCase().includes(needle) || s.topics.some((t) => t.label.toLowerCase().includes(needle)))

  function open(i: number) {
    setCurrent(i)
    // On a phone the reader sits below the contents; bring it into view.
    if (window.innerWidth < 1024) setTimeout(() => readerRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' }), 50)
  }

  const btn = 'shrink-0 border px-2.5 py-1 font-mono text-[11px] min-h-[30px] inline-flex items-center justify-center'
  return (
    <>
      <button type="button" onClick={onBack} className="font-mono text-xs text-mathua-secondary hover:text-mathua-blue">← Library</button>
      <div className="mt-4 mb-5">
        <p className="font-mono text-[11px] uppercase tracking-[0.15em] text-mathua-muted">{book.kind === 'pdf' ? 'Your PDF' : 'Online / deck'}</p>
        <h1 className="font-serif text-2xl text-mathua-primary mt-1 break-words">{book.title}</h1>
        {book.author && <p className="font-mono text-xs text-mathua-secondary mt-1">{book.author}</p>}
        <p className="font-mono text-[11px] text-mathua-muted mt-1">
          {book.sections.length} section{book.sections.length === 1 ? '' : 's'} · {book.topic_count} topic{book.topic_count === 1 ? '' : 's'}
        </p>
        {book.kind === 'pdf' && !book.available && (
          <p className="mt-3 border border-mathua-border bg-mathua-surface p-3 font-mono text-[11px] text-mathua-secondary">
            This PDF isn&apos;t in its module&apos;s files folder on this computer yet, so it can&apos;t be opened. The contents still link to every topic.
          </p>
        )}
      </div>
      <div className={`grid gap-4 ${book.kind === 'pdf' && book.available ? 'lg:grid-cols-[minmax(0,340px)_minmax(0,1fr)]' : ''}`}>
        <nav aria-label="Contents" className="min-w-0">
          <input
            id="lib-filter"
            type="search"
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            placeholder="Filter sections and topics"
            className="w-full mb-2 border border-mathua-border bg-mathua-surface text-mathua-primary font-mono text-xs px-3 py-2 min-h-[36px]"
          />
          <ol className="border border-mathua-border divide-y divide-mathua-border bg-mathua-surface lg:max-h-[78vh] lg:overflow-auto">
            {sections.map(({ s, i }) => (
              <li key={i} className={`px-3 py-2.5 min-w-0 ${i === current ? 'bg-mathua-blue-faint' : ''}`}>
                <div className="flex items-start gap-2 justify-between min-w-0">
                  {book.kind === 'pdf' && book.available ? (
                    <button type="button" onClick={() => open(i)} className="text-left min-w-0 font-mono text-xs text-mathua-primary hover:text-mathua-blue break-words [overflow-wrap:anywhere]" aria-current={i === current}>
                      {s.title}
                    </button>
                  ) : s.url ? (
                    <a href={s.url} target="_blank" rel="noreferrer" className="min-w-0 font-mono text-xs text-mathua-primary hover:text-mathua-blue break-words [overflow-wrap:anywhere]">
                      {s.title} ↗
                    </a>
                  ) : (
                    <span className="min-w-0 font-mono text-xs text-mathua-primary break-words [overflow-wrap:anywhere]">{s.title}</span>
                  )}
                </div>
                {pages(s) && <div className="font-mono text-[10px] text-mathua-muted">{pages(s)}</div>}
                <ul className="mt-1.5 space-y-1">
                  {s.topics.map((t) => {
                    const st = masteryState(progress[t.id])
                    return (
                      <li key={t.id} className="flex items-center gap-2 min-w-0">
                        <Link href={studyConceptPath(t.id)} className="min-w-0 flex-1 font-mono text-[11px] text-mathua-secondary hover:text-mathua-blue truncate" title={t.label}>
                          {st === 'mastered' ? '✓ ' : '· '}{t.label}
                        </Link>
                        <Link href={learnConceptPath(t.id)} className={`${btn} border-mathua-blue text-mathua-blue hover:bg-mathua-blue-faint`}>Practise</Link>
                      </li>
                    )
                  })}
                </ul>
              </li>
            ))}
          </ol>
        </nav>
        {book.kind === 'pdf' && book.available && (
          <div ref={readerRef} className="min-w-0">
            {sec ? (
              <PdfReader reading={{ title: sec.title, url: book.url ?? '', page: sec.page, end_page: sec.end_page }} />
            ) : book.url ? (
              <PdfReader reading={{ title: book.title, url: book.url, page: 1 }} />
            ) : null}
          </div>
        )}
      </div>
    </>
  )
}

function LibraryContent() {
  const { mounted } = useTheme()
  const router = useRouter()
  const params = useSearchParams()
  const [books, setBooks] = useState<LibraryBook[] | null>(null)
  const [progress, setProgress] = useState<Record<string, ConceptProgress>>({})
  const bookId = params.get('book') ?? ''

  useEffect(() => {
    getLibrary().then(setBooks).catch(() => setBooks([]))
    const sid = getUserInfo()?.student_id
    if (sid) getProgress(sid).then(setProgress).catch(() => {})
  }, [])

  const shelves = useMemo(() => {
    const order = TRACKS.map((t) => t.id)
    const byTrack = new Map<string, LibraryBook[]>()
    for (const b of books ?? []) {
      const list = byTrack.get(b.track) ?? []
      list.push(b)
      byTrack.set(b.track, list)
    }
    return [...byTrack.entries()]
      .sort((a, b) => (order.indexOf(a[0]) + 1 || 99) - (order.indexOf(b[0]) + 1 || 99))
      .map(([track, list]) => ({
        track,
        label: TRACKS.find((t) => t.id === track)?.label ?? track,
        books: list.sort((a, b) => (a.kind === b.kind ? b.topic_count - a.topic_count : a.kind === 'pdf' ? -1 : 1)),
      }))
  }, [books])

  if (!mounted || books === null) return <Loading label="Loading library" full />
  const book = books.find((b) => b.id === bookId) ?? null

  return (
    <>
      <Header />
      <main className="mx-auto w-full max-w-[1100px] min-w-0 px-4 sm:px-6 py-8 sm:py-12 overflow-x-hidden">
        {book ? (
          <BookView book={book} progress={progress} onBack={() => router.push('/library')} />
        ) : (
          <>
            <SectionHeader label="Library" title="Books and sources" />
            <p className="font-mono text-[11px] text-mathua-secondary mb-6">
              Everything your topics are drawn from. Open a book for its contents: every section lists the topics it teaches, and
              your PDFs open right there at the section.
            </p>
            {books.length === 0 && (
              <p className="border border-mathua-border bg-mathua-surface p-4 font-mono text-xs text-mathua-secondary">
                No books yet. A module adds them with readings.json and books.json (see FORK.md).
              </p>
            )}
            <div className="space-y-8">
              {shelves.map((sh) => (
                <section key={sh.track} aria-labelledby={`shelf-${sh.track}`}>
                  <h2 id={`shelf-${sh.track}`} className="font-mono text-[11px] uppercase tracking-[0.15em] text-mathua-muted mb-2">{sh.label}</h2>
                  <ul className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
                    {sh.books.map((b) => (
                      <li key={b.id} className="min-w-0">
                        <Link
                          href={`/library?book=${encodeURIComponent(b.id)}`}
                          className="h-full flex flex-col gap-1 border border-mathua-border bg-mathua-surface p-3 hover:border-mathua-blue min-w-0"
                        >
                          <span className="font-mono text-[10px] uppercase tracking-wider text-mathua-muted">
                            {b.kind === 'pdf' ? (b.available ? 'Your PDF' : 'PDF, not on this computer') : 'Online / deck'}
                          </span>
                          <span className="font-mono text-xs text-mathua-primary break-words [overflow-wrap:anywhere]">{b.title}</span>
                          {b.author && <span className="font-mono text-[10px] text-mathua-secondary">{b.author}</span>}
                          <span className="mt-auto pt-1 font-mono text-[10px] text-mathua-muted">
                            {b.sections.length} section{b.sections.length === 1 ? '' : 's'} · {b.topic_count} topic{b.topic_count === 1 ? '' : 's'}
                          </span>
                        </Link>
                      </li>
                    ))}
                  </ul>
                </section>
              ))}
            </div>
          </>
        )}
      </main>
      <Footer />
      <BottomTabs />
    </>
  )
}

export default function LibraryPage() {
  return (
    <Suspense fallback={<Loading label="Loading library" full />}>
      <LibraryContent />
    </Suspense>
  )
}
