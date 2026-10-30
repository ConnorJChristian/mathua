import type { LibraryBook } from './api'

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
