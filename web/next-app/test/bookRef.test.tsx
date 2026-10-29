import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { BOOK_MARKER, bookRef, stripBookMarker } from '../lib/bookRef'
import { CARD_MARKER, TYPED_MARKER, displayQuestion, pythonKind } from '../lib/pythonQuestion'

// The embedded reader draws with pdf.js; here it only has to receive the page.
vi.mock('../components/PdfReader', () => ({
  default: ({ reading }: { reading: { title: string; url: string; page?: number } }) => (
    <div data-testid="reader">{`${reading.title}|${reading.url}|${reading.page}`}</div>
  ),
}))
import BookRefPanel from '../components/BookRefPanel'

// The shape internal/generator/bookref.go serves: marker, JSON, "-->", the
// visible line, then the item's own prompt and answer marker. Literal, so a
// change to the server's format fails here.
const LINE = '**Hoy Exercise 5.4, #3** (book page 173, PDF page 187).'
const WITH_PDF = `${BOOK_MARKER}{"file":"hoy-3e.pdf","module":"ma-program","page":187,"printed_page":173,"label":"Hoy Exercise 5.4, #3","available":true,"url":"/api/modules/ma-program/files/hoy-3e.pdf"}-->\n\n${LINE}\n\nEnter each derivative.\n\n${TYPED_MARKER}[{"label":"(a)","grading":"expression"}]-->`
const NO_PDF = `${CARD_MARKER}\n${BOOK_MARKER}{"file":"hoy-3e.pdf","module":"ma-program","page":187,"printed_page":173,"label":"Hoy Exercise 5.4, #3","available":false}-->\n\n${LINE} The PDF is not installed, so work from your copy of the book.\n\nEnter each derivative.\n\n${TYPED_MARKER}[{"label":"(a)","grading":"expression"}]-->`

describe('book-reference questions', () => {
  it('parse the page, with or without the card marker', () => {
    const a = bookRef(WITH_PDF)
    expect(a).toMatchObject({ page: 187, printed_page: 173, label: 'Hoy Exercise 5.4, #3', available: true, url: '/api/modules/ma-program/files/hoy-3e.pdf' })
    const b = bookRef(NO_PDF)
    expect(b).toMatchObject({ page: 187, available: false })
    expect(b?.url).toBeUndefined()
    expect(bookRef('plain question')).toBeNull()
    expect(bookRef(`${BOOK_MARKER}not json-->`)).toBeNull()
  })

  it('show the visible line and prompt without the marker, and keep their input', () => {
    expect(displayQuestion(WITH_PDF)).toBe(`${LINE}\n\nEnter each derivative.`)
    expect(displayQuestion(NO_PDF)).toContain('Enter each derivative.')
    expect(displayQuestion(NO_PDF)).not.toContain(BOOK_MARKER)
    expect(pythonKind(WITH_PDF)).toBe('math')
    expect(stripBookMarker(NO_PDF).startsWith(CARD_MARKER)).toBe(true)
  })

  it('open the PDF at the exercise when it is installed', () => {
    render(<BookRefPanel question={WITH_PDF} />)
    expect(screen.getByTestId('reader').textContent).toBe('Hoy Exercise 5.4, #3|/api/modules/ma-program/files/hoy-3e.pdf|187')
  })

  it('fall back to the label and page numbers when the PDF is missing', () => {
    render(<BookRefPanel question={NO_PDF} />)
    expect(screen.queryByTestId('reader')).toBeNull()
    expect(screen.getByText('Hoy Exercise 5.4, #3')).toBeTruthy()
    expect(screen.getByText(/book page 173 \(PDF page 187\)/)).toBeTruthy()
  })

  it('render nothing for an ordinary question', () => {
    const { container } = render(<BookRefPanel question="What is 2 + 2?" />)
    expect(container.innerHTML).toBe('')
  })
})
