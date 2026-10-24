'use client'

import { useEffect, useRef, useState } from 'react'
import { API_BASE } from '../lib/apiBase'

// Fork change: an embedded reader for a lesson whose material comes from your
// own PDF (a module's readings.json). It opens the book at the section's first
// page, says which pages to read, and pages through them in place, so the
// reading happens inside the lesson instead of in another tab.
//
// Pages are drawn with pdf.js rather than the browser's PDF plugin: phones
// (Android Chrome, iOS Safari) either download an embedded PDF or show only its
// first page, and a canvas renders the same everywhere. The worker is copied
// to /pdfjs at build time (scripts/copy-pdf-worker.mjs). Only the pages viewed
// are fetched, using range requests, so a 900-page textbook opens quickly.

export interface ReadingPdf {
  title: string
  url: string
  page?: number
  end_page?: number
}

type PdfDoc = {
  numPages: number
  getPage: (n: number) => Promise<PdfPage>
  destroy: () => Promise<void>
}
type PdfPage = {
  getViewport: (o: { scale: number }) => { width: number; height: number }
  render: (o: {
    canvasContext: CanvasRenderingContext2D
    viewport: { width: number; height: number }
    transform?: number[]
  }) => { promise: Promise<void>; cancel: () => void }
}

function absolute(url: string): string {
  if (/^https?:\/\//.test(url)) return url
  return `${API_BASE}${url}`
}

export default function PdfReader({ reading }: { reading: ReadingPdf }) {
  const start = Math.max(1, reading.page || 1)
  const end = reading.end_page && reading.end_page >= start ? reading.end_page : undefined
  const [page, setPage] = useState(start)
  const [numPages, setNumPages] = useState(0)
  const [zoom, setZoom] = useState(1)
  const [width, setWidth] = useState(0)
  const [open, setOpen] = useState(true)
  const [status, setStatus] = useState<'loading' | 'ready' | 'error'>('loading')
  const docRef = useRef<PdfDoc | null>(null)
  const boxRef = useRef<HTMLDivElement>(null)
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const taskRef = useRef<{ cancel: () => void } | null>(null)

  useEffect(() => setPage(start), [start, reading.url])

  // Load the document once per file.
  useEffect(() => {
    let cancelled = false
    setStatus('loading')
    ;(async () => {
      try {
        const pdfjs = await import('pdfjs-dist/legacy/build/pdf.mjs')
        pdfjs.GlobalWorkerOptions.workerSrc = '/pdfjs/pdf.worker.min.mjs'
        const doc = (await pdfjs.getDocument({
          url: absolute(reading.url),
          disableAutoFetch: true,
          rangeChunkSize: 262144,
        }).promise) as unknown as PdfDoc
        if (cancelled) {
          void doc.destroy()
          return
        }
        docRef.current = doc
        setNumPages(doc.numPages)
        setStatus('ready')
      } catch {
        if (!cancelled) setStatus('error')
      }
    })()
    return () => {
      cancelled = true
      taskRef.current?.cancel()
      void docRef.current?.destroy()
      docRef.current = null
    }
  }, [reading.url])

  // Fit the page to the reader's width.
  useEffect(() => {
    const el = boxRef.current
    if (!el) return
    const ro = new ResizeObserver(() => setWidth(el.clientWidth))
    ro.observe(el)
    setWidth(el.clientWidth)
    return () => ro.disconnect()
  }, [open])

  // Draw the current page.
  useEffect(() => {
    const doc = docRef.current
    if (!doc || status !== 'ready' || !open || !width) return
    let cancelled = false
    ;(async () => {
      try {
        const p = await doc.getPage(Math.min(Math.max(page, 1), doc.numPages))
        if (cancelled) return
        const base = p.getViewport({ scale: 1 })
        const scale = ((width - 2) / base.width) * zoom
        const vp = p.getViewport({ scale })
        const dpr = Math.min(window.devicePixelRatio || 1, 2)
        const canvas = canvasRef.current
        if (!canvas) return
        canvas.width = Math.floor(vp.width * dpr)
        canvas.height = Math.floor(vp.height * dpr)
        canvas.style.width = `${Math.floor(vp.width)}px`
        canvas.style.height = `${Math.floor(vp.height)}px`
        const ctx = canvas.getContext('2d')
        if (!ctx) return
        taskRef.current?.cancel()
        const task = p.render({ canvasContext: ctx, viewport: vp, transform: dpr !== 1 ? [dpr, 0, 0, dpr, 0, 0] : undefined })
        taskRef.current = task
        await task.promise
      } catch {
        // A cancelled render (the learner turned the page mid-draw) is expected.
      }
    })()
    boxRef.current?.scrollTo({ top: 0 })
    return () => {
      cancelled = true
    }
  }, [page, zoom, width, open, status])

  const fileLink = `${absolute(reading.url)}#page=${page}`
  const inSection = end ? page >= start && page <= end : page >= start
  const sectionPos = end && inSection ? `page ${page - start + 1} of ${end - start + 1} in this section` : ''
  const btn = 'border border-mathua-border px-2.5 min-h-[32px] font-mono text-[11px] text-mathua-secondary hover:border-mathua-blue hover:text-mathua-blue disabled:opacity-40 inline-flex items-center justify-center'

  return (
    <section className="mt-4 border border-mathua-border bg-mathua-bg min-w-0" aria-label="Reading">
      <div className="px-3 py-2 border-b border-mathua-border flex flex-wrap items-center gap-2 justify-between">
        <div className="min-w-0">
          <p className="font-mono text-[10px] uppercase tracking-wider text-mathua-muted">Read</p>
          <p className="font-mono text-xs text-mathua-primary break-words [overflow-wrap:anywhere]">{reading.title}</p>
          {end && (
            <p className="font-mono text-[10px] text-mathua-secondary">
              PDF pages {start}–{end}{sectionPos ? ` · ${sectionPos}` : ''}
            </p>
          )}
        </div>
        <button type="button" onClick={() => setOpen(!open)} className={btn} aria-expanded={open}>
          {open ? 'Hide reader' : 'Show reader'}
        </button>
      </div>
      {open && (
        <>
          <div className="px-3 py-2 flex flex-wrap items-center gap-1.5 border-b border-mathua-border">
            <button type="button" className={btn} onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1} aria-label="Previous page">‹ Prev</button>
            <span className="font-mono text-[11px] text-mathua-secondary tabular-nums px-1">
              Page {page}{numPages ? ` / ${numPages}` : ''}
            </span>
            <button type="button" className={btn} onClick={() => setPage((p) => Math.min(numPages || p + 1, p + 1))} disabled={!!numPages && page >= numPages} aria-label="Next page">Next ›</button>
            <span className="w-2" aria-hidden />
            <button type="button" className={btn} onClick={() => setZoom((z) => Math.max(0.6, +(z - 0.2).toFixed(1)))} aria-label="Zoom out">−</button>
            <button type="button" className={btn} onClick={() => setZoom((z) => Math.min(2.4, +(z + 0.2).toFixed(1)))} aria-label="Zoom in">+</button>
            {page !== start && (
              <button type="button" className={btn} onClick={() => setPage(start)}>Back to section start</button>
            )}
            <a href={fileLink} target="_blank" rel="noreferrer" className={`${btn} ml-auto`}>Open in new tab</a>
          </div>
          {!inSection && (
            <p className="px-3 py-1.5 font-mono text-[11px] text-mathua-secondary border-b border-mathua-border">
              {page < start ? 'Before' : 'Past the end of'} this section.
            </p>
          )}
          <div ref={boxRef} className="max-h-[75vh] overflow-auto bg-mathua-surface">
            {status === 'loading' && <p className="p-4 font-mono text-xs text-mathua-muted">Opening the PDF…</p>}
            {status === 'error' && (
              <p className="p-4 font-mono text-xs text-mathua-secondary">
                The PDF could not be opened here. Check that the file is in the module&apos;s files folder, or{' '}
                <a href={fileLink} target="_blank" rel="noreferrer" className="underline">open it in a new tab</a>.
              </p>
            )}
            <canvas ref={canvasRef} className={status === 'ready' ? 'block mx-auto bg-white' : 'hidden'} />
          </div>
        </>
      )}
    </section>
  )
}
