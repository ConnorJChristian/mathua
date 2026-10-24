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

export default function ReadingPanel({ reading, texts }: { reading?: ReadingPdf; texts?: ReadingText[] }) {
  const initial = texts?.find((t) => t.selected)?.key ?? texts?.[0]?.key ?? ''
  const [key, setKey] = useState(initial)
  useEffect(() => setKey(initial), [initial])

  if (!texts || texts.length < 2) return reading ? <PdfReader reading={reading} /> : null

  const chosen = texts.find((t) => t.key === key) ?? texts[0]
  return (
    <div className="mt-4">
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
      {chosen.kind === 'pdf' ? (
        <PdfReader reading={{ title: chosen.source, url: chosen.url ?? '', page: chosen.page, end_page: chosen.end_page }} />
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
