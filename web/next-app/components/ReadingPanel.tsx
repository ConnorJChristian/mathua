'use client'

import { useEffect, useState } from 'react'
import PdfReader, { type ReadingPdf } from './PdfReader'
import type { ReadingText } from '../lib/api'
import { API_BASE } from '../lib/apiBase'

// Fork change: what to read for a topic, in which text. A topic can be read in
// several texts (Hoy and Schaum's, the Python Data Science Handbook and
// ISLP...). The server picks the default from the course you reached the
// topic through (its authority texts); the others are one click away here.
// PDFs open in the embedded reader; online books open in a new tab.

function absolute(url: string): string {
  return /^https?:\/\//.test(url) ? url : `${API_BASE}${url}`
}

// `onUnavailable`: the PDF shown could not be opened here (see PdfReader).
export default function ReadingPanel({ reading, texts, onUnavailable }: { reading?: ReadingPdf; texts?: ReadingText[]; onUnavailable?: () => void }) {
  const initial = texts?.find((t) => t.selected)?.key ?? texts?.[0]?.key ?? ''
  const [key, setKey] = useState(initial)
  useEffect(() => setKey(initial), [initial])

  if (!texts || texts.length === 0) return reading ? <PdfReader reading={reading} onUnavailable={onUnavailable} /> : null
  // One PDF text: the embedded reader alone, with no picker. A single text of any other
  // kind (lecture notes, an online book) falls through, so it is still shown.
  if (texts.length === 1 && texts[0].kind === 'pdf' && reading) return <PdfReader reading={reading} onUnavailable={onUnavailable} />

  const chosen = texts.find((t) => t.key === key) ?? texts[0]
  return (
    <div className="mt-4">
      {texts.length > 1 && (
      <div role="radiogroup" aria-label="Text" className="flex flex-wrap items-center gap-1.5">
        <span className="font-mono text-[10px] uppercase tracking-wider text-mathua-muted mr-1">Text</span>
        {texts.map((t) => (
          <button
            key={t.key}
            type="button"
            role="radio"
            aria-checked={t.key === chosen.key}
            onClick={() => setKey(t.key)}
            className={`border px-2.5 min-h-[32px] font-mono text-[11px] text-left ${
              t.key === chosen.key
                ? 'border-mathua-blue text-mathua-blue bg-mathua-blue-faint'
                : 'border-mathua-border text-mathua-secondary hover:border-mathua-blue hover:text-mathua-blue'
            }`}
          >
            {t.book}
            {t.selected && t.key !== chosen.key ? ' (course text)' : ''}
          </button>
        ))}
      </div>
      )}
      {chosen.kind === 'pdf' ? (
        <PdfReader reading={{ title: chosen.source, url: chosen.url ?? '', page: chosen.page, end_page: chosen.end_page }} onUnavailable={onUnavailable} />
      ) : chosen.kind === 'file' && chosen.url && /\.html?($|[?#])/i.test(chosen.url) ? (
        // Lecture notes (a rendered notebook) read in place, like a PDF section.
        // The page runs its own scripts (MathJax typesets the notebook's
        // formulas; with scripts blocked every lecture showed raw TeX and
        // logged a blocked-script error), but as an opaque origin: it cannot
        // read the app's storage or token. Links in the notes open in a tab.
        <div className="mt-3 border border-mathua-border bg-mathua-bg">
          <p className="px-3 pt-2.5 font-mono text-[10px] uppercase tracking-wider text-mathua-muted">Read · {chosen.source}</p>
          <iframe title={chosen.source} src={absolute(chosen.url)} sandbox="allow-scripts allow-popups allow-popups-to-escape-sandbox" className="mt-2 w-full h-[70vh] bg-white" />
          <a
            href={absolute(chosen.url)}
            target="_blank"
            rel="noreferrer"
            className="m-3 inline-flex border border-mathua-border px-2.5 min-h-[32px] items-center font-mono text-[11px] text-mathua-secondary hover:border-mathua-blue hover:text-mathua-blue"
          >
            Open in a new tab
          </a>
        </div>
      ) : (
        <div className="mt-3 border border-mathua-border bg-mathua-bg px-3 py-2.5">
          <p className="font-mono text-[10px] uppercase tracking-wider text-mathua-muted">Read</p>
          <p className="font-mono text-xs text-mathua-primary">{chosen.source}{chosen.kind === 'web' ? `, in ${chosen.book}` : ''}</p>
          {chosen.url && (
            <a
              href={chosen.kind === 'web' || chosen.kind === 'link' ? chosen.url : `${absolute(chosen.url)}${chosen.page ? `#page=${chosen.page}` : ''}`}
              target="_blank"
              rel="noreferrer"
              className="mt-2 inline-flex border border-mathua-border px-2.5 min-h-[32px] items-center font-mono text-[11px] text-mathua-secondary hover:border-mathua-blue hover:text-mathua-blue"
            >
              Open section in a new tab
            </a>
          )}
        </div>
      )}
    </div>
  )
}
