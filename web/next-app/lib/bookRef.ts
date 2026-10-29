// Fork change: a question whose prompt is a page of the learner's own book
// (internal/generator/bookref.go). The served question opens with
// BOOK_MARKER, a JSON description of the page and "-->", then a visible line
// naming the exercise and its pages. The bank stores no exercise text: the
// learner reads the exercise in the embedded PDF reader above the answer box,
// or in their paper copy when the PDF is not installed.

export const BOOK_MARKER = '<!--book-ref '

export interface BookRef {
  file: string
  module: string
  page: number
  end_page?: number
  printed_page?: number
  label: string
  /** The module's files/ folder has the PDF. */
  available: boolean
  /** Where the server serves the PDF; absent when it is not installed. */
  url?: string
}

/** The question without the card marker the engine puts before a filed card. */
function body(question: string): string {
  return question.replace(/^<!--card-->\n?/, '')
}

/** The book reference a served question carries, or null. */
export function bookRef(question?: string | null): BookRef | null {
  if (!question) return null
  const q = body(question)
  if (!q.startsWith(BOOK_MARKER)) return null
  const end = q.indexOf('-->')
  if (end < 0) return null
  try {
    const raw = JSON.parse(q.slice(BOOK_MARKER.length, end)) as Partial<BookRef>
    if (!raw || typeof raw.label !== 'string' || typeof raw.page !== 'number') return null
    return {
      file: String(raw.file ?? ''),
      module: String(raw.module ?? ''),
      page: raw.page,
      end_page: typeof raw.end_page === 'number' ? raw.end_page : undefined,
      printed_page: typeof raw.printed_page === 'number' ? raw.printed_page : undefined,
      label: raw.label,
      available: raw.available === true && typeof raw.url === 'string' && raw.url !== '',
      url: typeof raw.url === 'string' && raw.url ? raw.url : undefined,
    }
  } catch {
    return null
  }
}

/** The question with the book marker and its JSON removed; the visible line stays. */
export function stripBookMarker(question: string): string {
  const card = question.startsWith('<!--card-->')
  const q = body(question)
  if (!q.startsWith(BOOK_MARKER)) return question
  const end = q.indexOf('-->')
  if (end < 0) return question
  const rest = q.slice(end + 3).replace(/^\n+/, '')
  return card ? '<!--card-->\n' + rest : rest
}

/** Where to find the exercise, for a learner without the PDF. */
export function bookWhere(ref: BookRef): string {
  return ref.printed_page ? `book page ${ref.printed_page} (PDF page ${ref.page})` : `PDF page ${ref.page}`
}
