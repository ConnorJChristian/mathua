'use client'

import { Suspense, useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { usePathname, useRouter, useSearchParams } from 'next/navigation'
import Header from '../../components/Header'
import BottomTabs from '../../components/BottomTabs'
import SectionHeader from '../../components/SectionHeader'
import Footer from '../../components/Footer'
import Loading from '../../components/Loading'
import GlossarySearch, { type GlossaryCategoryFilter, type GlossarySearchMode } from '../../components/GlossarySearch'
import GlossaryEntryCard from '../../components/GlossaryEntry'
import { useTheme } from '../../hooks/useTheme'
import { GLOSSARY_ENTRIES } from '../../lib/glossary/data'
import { categoryLabel } from '../../lib/glossary/categories'
import {
  filterEntries,
  GLOSSARY_PAGE_SIZE,
  paginateEntries,
  parsePageParam,
  totalPagesFor,
} from '../../lib/glossary/search'
import { mathSearchEntries } from '../../lib/glossary/mathQuery'
import { scrollToPageTop } from '../../lib/scroll'

/**
 * The mathematical glossary: a standalone reference for symbols, notation
 * and terminology. It reads the static bundle only — no API, no auth, no
 * XP, no progress — so looking anything up cannot affect learning state.
 *
 * Results render one page at a time (20 entries); the page lives in ?page=
 * so it is shareable and survives back/forward navigation.
 */

// Compact page window around the current page: 1 … p−2..p+2 … N.
function pageWindow(current: number, total: number): (number | '…')[] {
  if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1)
  const inner = new Set([1, 2, total - 1, total, current - 1, current, current + 1])
  const pages = [...inner].filter(p => p >= 1 && p <= total).sort((a, b) => a - b)
  const out: (number | '…')[] = []
  for (let i = 0; i < pages.length; i++) {
    if (i > 0 && pages[i] - pages[i - 1] > 1) out.push('…')
    out.push(pages[i])
  }
  return out
}

function PaginationControls({
  page,
  totalPages,
  onGo,
}: {
  page: number
  totalPages: number
  onGo: (p: number) => void
}) {
  if (totalPages <= 1) return null
  const btn =
    'border px-3 py-1 font-mono text-[11px] min-h-[44px] min-w-[44px] transition-colors disabled:opacity-40 disabled:cursor-default'
  const idle = 'border-mathua-border text-mathua-secondary hover:border-mathua-blue hover:text-mathua-blue'
  const current = 'border-mathua-blue text-mathua-blue'
  return (
    <nav aria-label="Glossary pages" className="flex flex-wrap items-center gap-2">
      <button
        type="button"
        onClick={() => onGo(page - 1)}
        disabled={page <= 1}
        aria-label="Previous page"
        className={`${btn} ${idle}`}
      >
        ←
      </button>
      {pageWindow(page, totalPages).map((p, i) =>
        p === '…' ? (
          <span key={`gap-${i}`} aria-hidden="true" className="font-mono text-[11px] text-mathua-muted">
            …
          </span>
        ) : (
          <button
            key={p}
            type="button"
            onClick={() => onGo(p)}
            disabled={p === page}
            aria-label={`Go to page ${p}`}
            aria-current={p === page ? 'page' : undefined}
            className={`${btn} ${p === page ? current : idle}`}
          >
            {p}
          </button>
        ),
      )}
      <button
        type="button"
        onClick={() => onGo(page + 1)}
        disabled={page >= totalPages}
        aria-label="Next page"
        className={`${btn} ${idle}`}
      >
        →
      </button>
    </nav>
  )
}

function GlossaryPageInner() {
  const { mounted } = useTheme()
  const router = useRouter()
  const pathname = usePathname()
  const searchParams = useSearchParams()
  const [query, setQuery] = useState('')
  const [category, setCategory] = useState<GlossaryCategoryFilter>('all')
  const [mode, setMode] = useState<GlossarySearchMode>('text')
  const [mathLatex, setMathLatex] = useState('')
  const [page, setPageState] = useState(1)
  const pendingScrollId = useRef<string | null>(null)

  const inCategory =
    category === 'all' ? GLOSSARY_ENTRIES : GLOSSARY_ENTRIES.filter(e => e.category === category)
  const results = useMemo(
    () => (mode === 'math' ? mathSearchEntries(inCategory, mathLatex) : filterEntries(GLOSSARY_ENTRIES, { query, category })),
    [inCategory, mathLatex, mode, query, category],
  )
  const totalPages = totalPagesFor(results.length)

  // The URL is the source of truth for the page: back/forward navigation and
  // shared links land on the right page. Anything else clamps into range.
  const urlPage = parsePageParam(searchParams.get('page'), totalPages)
  useEffect(() => {
    setPageState(urlPage)
  }, [urlPage])

  // A related-symbol jump may target another page: land there first, then
  // scroll to the card once it renders.
  useEffect(() => {
    if (!pendingScrollId.current) return
    const id = pendingScrollId.current
    pendingScrollId.current = null
    requestAnimationFrame(() => {
      document.getElementById(`glossary-${id}`)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
      document.getElementById(`glossary-${id}-name`)?.focus?.()
    })
  }, [page, results])

  function writePage(p: number, scroll: boolean, push = false) {
    const params = new URLSearchParams(searchParams.toString())
    if (p <= 1) params.delete('page')
    else params.set('page', String(p))
    const qs = params.toString()
    const url = `${pathname}${qs ? `?${qs}` : ''}`
    // Page turns push history entries so back/forward moves between pages;
    // filter typing replaces so each keystroke is not a history entry.
    if (push) router.push(url, { scroll: false })
    else router.replace(url, { scroll: false })
    if (scroll) {
      scrollToPageTop()
    }
  }

  function goToPage(p: number) {
    writePage(Math.min(Math.max(1, p), totalPages), true, true)
  }

  function handleQueryChange(q: string) {
    setQuery(q)
    if (mathLatex !== '') setMathLatex('')
    writePage(1, false)
  }

  function handleMathChange(latex: string) {
    setMathLatex(latex)
    if (query !== '') setQuery('')
    // Stay on the current page while typing so the list does not jump;
    // clearing to empty shows the full list again via the no-terms path.
  }

  function handleModeChange(m: GlossarySearchMode) {
    setMode(m)
    // One active query: switching modes clears the other input.
    if (m === 'math') setQuery('')
    else setMathLatex('')
    writePage(1, false)
  }

  function handleCategoryChange(c: GlossaryCategoryFilter) {
    setCategory(c)
    writePage(1, false)
  }

  function clear() {
    setQuery('')
    setCategory('all')
    setMathLatex('')
    writePage(1, false)
  }

  const selectRelated = useCallback(
    (id: string) => {
      // The target may live on another page of the current list: resolve its
      // page from the same ordering the list renders, then scroll on arrival.
      const idx = results.findIndex(e => e.id === id)
      pendingScrollId.current = id
      if (idx < 0) {
        clear()
        return
      }
      writePage(Math.floor(idx / GLOSSARY_PAGE_SIZE) + 1, false)
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [results],
  )

  if (!mounted) return <Loading label="Loading glossary" full />

  const pageEntries = paginateEntries(results, page)
  const rangeStart = results.length === 0 ? 0 : (page - 1) * GLOSSARY_PAGE_SIZE + 1
  const rangeEnd = Math.min(results.length, page * GLOSSARY_PAGE_SIZE)
  let lastCategory: string | null = null

  return (
    <>
      <Header />
      <main className="mx-auto w-full max-w-[820px] min-w-0 px-4 sm:px-6 py-8 sm:py-12 overflow-x-hidden">
        <SectionHeader label="Reference" title="Mathematical glossary" />

        <GlossarySearch
          query={query}
          category={category}
          resultCount={results.length}
          rangeStart={rangeStart}
          rangeEnd={rangeEnd}
          mode={mode}
          mathLatex={mathLatex}
          onQueryChange={handleQueryChange}
          onCategoryChange={handleCategoryChange}
          onModeChange={handleModeChange}
          onMathChange={handleMathChange}
          onClear={clear}
        />

        <div className="mt-6">
          {results.length === 0 ? (
            <div className="border border-mathua-border bg-mathua-surface px-4 py-8 text-center">
              <p className="font-mono text-xs text-mathua-primary">
                No entries match{query.trim() && <> “{query.trim()}”</>}.
              </p>
              <p className="mt-2 font-mono text-[11px] text-mathua-secondary">
                Try a symbol (∀, ∫, λ), a name (integral, lambda) or a word
                from a definition.
              </p>
              <button
                type="button"
                onClick={clear}
                className="mt-4 border border-mathua-blue px-4 py-2 font-mono text-[11px] text-mathua-blue hover:bg-mathua-blue-faint min-h-[44px]"
              >
                Clear search and filters
              </button>
            </div>
          ) : (
            <>
              <div className="space-y-4">
                {pageEntries.map(e => {
                  const showDivider = e.category !== lastCategory
                  lastCategory = e.category
                  return (
                    <div key={e.id}>
                      {showDivider && (
                        <h2 className="mb-3 font-mono text-[11px] uppercase tracking-[0.15em] text-mathua-muted">
                          {categoryLabel(e.category)}
                        </h2>
                      )}
                      <GlossaryEntryCard entry={e} onSelectRelated={selectRelated} />
                    </div>
                  )
                })}
              </div>
              <div className="mt-6">
                <PaginationControls page={page} totalPages={totalPages} onGo={goToPage} />
              </div>
            </>
          )}
        </div>
      </main>
      <Footer />
      <BottomTabs />
    </>
  )
}

// useSearchParams requires a Suspense boundary for static prerendering.
export default function GlossaryPage() {
  return (
    <Suspense fallback={null}>
      <GlossaryPageInner />
    </Suspense>
  )
}
