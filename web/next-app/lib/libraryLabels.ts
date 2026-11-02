import type { LibraryBook, LibrarySection } from './api'

// What a library card and a book page say a book is. A PDF that is not on
// this computer was headed "YOUR PDF" on its page and then said it could not
// be opened (Q63).
export function bookKindLabel(b: Pick<LibraryBook, 'kind' | 'available'>): string {
  if (b.kind !== 'pdf') return 'Online book'
  return b.available ? 'Your PDF' : 'PDF, not on this computer'
}

// "12 sections · 30 topics"; a book with no listed contents says so rather
// than "0 sections · 0 topics" (Q63: the Hoy Online Supplement).
export function bookCounts(b: Pick<LibraryBook, 'sections' | 'topic_count'>): string {
  const n = b.sections.length
  if (n === 0 && b.topic_count === 0) return 'Whole book: no contents listed'
  return `${n} section${n === 1 ? '' : 's'} · ${b.topic_count} topic${b.topic_count === 1 ? '' : 's'}`
}

// The PDF a contents row opens: its own file for a row of a companion book
// (the Hoy Online Supplement inside Hoy's page), else the book's. Null when
// that file is not on this computer.
export function sectionPdf(b: Pick<LibraryBook, 'kind' | 'available' | 'url'>, s: LibrarySection): string | null {
  if (b.kind !== 'pdf') return null
  if (s.pdf) return s.pdf_missing ? null : s.pdf
  return b.available && b.url ? b.url : null
}

// A book page has a reader when any of its rows can be opened: Hoy's own PDF
// may be missing while its supplement's is here.
export function bookHasReader(b: Pick<LibraryBook, 'kind' | 'available' | 'url' | 'sections'>): boolean {
  return b.kind === 'pdf' && (b.available || b.sections.some((s) => sectionPdf(b, s) !== null))
}
