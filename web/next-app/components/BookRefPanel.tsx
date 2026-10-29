'use client'

import PdfReader from './PdfReader'
import { bookRef, bookWhere } from '../lib/bookRef'

// Fork change: the page of the learner's book a question asks about, shown
// directly above the answer box (lib/bookRef.ts). With the PDF installed it
// opens in the embedded reader at the exercise; without it the label and the
// page numbers say where to look in a paper copy.
export default function BookRefPanel({ question }: { question?: string | null }) {
  const ref = bookRef(question)
  if (!ref) return null
  if (ref.available && ref.url) {
    return <PdfReader reading={{ title: ref.label, url: ref.url, page: ref.page, end_page: ref.end_page }} />
  }
  return (
    <section className="mt-4 border border-mathua-border bg-mathua-bg px-3 py-2.5 min-w-0" aria-label="Book exercise">
      <p className="font-mono text-[10px] uppercase tracking-wider text-mathua-muted">Read in your book</p>
      <p className="font-mono text-xs text-mathua-primary break-words [overflow-wrap:anywhere]">{ref.label}</p>
      <p className="font-mono text-[11px] text-mathua-secondary">
        {bookWhere(ref)}. The PDF ({ref.file}) is not in the {ref.module} module&apos;s files folder, so open your copy of the book at this page.
      </p>
    </section>
  )
}
