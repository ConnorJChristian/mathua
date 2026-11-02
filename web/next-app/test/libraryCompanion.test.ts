import { describe, expect, it } from 'vitest'
import { bookHasReader, sectionPdf } from '../lib/libraryLabels'
import type { LibraryBook } from '../lib/api'

// The Hoy page lists the Online Supplement's chapters after Hoy's: those rows
// open the supplement's PDF, Hoy's rows open Hoy's, and a missing file is
// never stood in for by the other book's PDF.
const hoy: LibraryBook = {
  id: 'ma-program-hoy-3e-pdf', kind: 'pdf', track: 'math', title: 'Hoy', url: '/api/modules/ma-program/files/hoy-3e.pdf',
  available: false, topic_count: 2,
  sections: [
    { title: '2.1 Sets', page: 25, topics: [] },
    { title: 'Chapter S2 Review of Fundamentals', page: 1, pdf: '/api/modules/ma-program/files/hoy-supplement.pdf', book: 'Supplement', topics: [] },
  ],
}

describe('library companion rows', () => {
  it('opens a companion row in its own PDF', () => {
    expect(sectionPdf(hoy, hoy.sections[1])).toBe('/api/modules/ma-program/files/hoy-supplement.pdf')
  })
  it('does not open a row of a book whose PDF is missing', () => {
    expect(sectionPdf(hoy, hoy.sections[0])).toBeNull()
    expect(sectionPdf({ ...hoy, available: true }, hoy.sections[0])).toBe(hoy.url)
  })
  it('does not open a missing companion PDF', () => {
    expect(sectionPdf({ ...hoy, available: true }, { ...hoy.sections[1], pdf_missing: true })).toBeNull()
  })
  it('has a reader when any row can open', () => {
    expect(bookHasReader(hoy)).toBe(true)
    expect(bookHasReader({ ...hoy, sections: [hoy.sections[0]] })).toBe(false)
    expect(bookHasReader({ ...hoy, kind: 'web', available: true })).toBe(false)
  })
})
