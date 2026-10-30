import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent, act } from '@testing-library/react'

// pdf.js stand-in: a 4-page document whose pages draw nothing and whose
// text layer writes one span per page, so the reader's own behaviour (which
// page opens, where the section ends, that the text is laid over the canvas)
// is what the test sees.
const textLayerRenders: string[] = []
const rendered: number[] = []
vi.mock('pdfjs-dist/legacy/build/pdf.mjs', () => {
  class TextLayer {
    private o: { container: HTMLElement; textContentSource: string }
    constructor(o: { container: HTMLElement; textContentSource: string }) {
      this.o = o
    }
    async render() {
      const span = document.createElement('span')
      span.textContent = this.o.textContentSource
      this.o.container.appendChild(span)
      textLayerRenders.push(this.o.textContentSource)
    }
    cancel() {}
  }
  const page = (n: number) => ({
    getViewport: ({ scale }: { scale: number }) => ({ width: 600 * scale, height: 800 * scale, scale }),
    render: () => {
      rendered.push(n)
      return { promise: Promise.resolve(), cancel: () => {} }
    },
    streamTextContent: () => `text of page ${n}`,
  })
  return {
    GlobalWorkerOptions: { workerSrc: '' },
    getDocument: () => ({ promise: Promise.resolve({ numPages: 4, getPage: async (n: number) => page(n), destroy: async () => {} }) }),
    TextLayer,
  }
})

import PdfReader from '../components/PdfReader'

beforeEach(() => {
  textLayerRenders.length = 0
  rendered.length = 0
  // jsdom has no layout: give the reader a width and a 2D context.
  Object.defineProperty(HTMLElement.prototype, 'clientWidth', { configurable: true, get: () => 640 })
  ;(globalThis as unknown as { ResizeObserver: unknown }).ResizeObserver = class {
    observe() {}
    disconnect() {}
  }
  HTMLCanvasElement.prototype.getContext = (() => ({})) as unknown as HTMLCanvasElement['getContext']
  HTMLElement.prototype.scrollTo = (() => {}) as unknown as HTMLElement['scrollTo']
})

describe('PdfReader', () => {
  it('opens at the section page, lays the text over it and marks the section end', async () => {
    render(<PdfReader reading={{ title: 'Stub 1.2', url: '/api/modules/m/files/stub.pdf', page: 2, end_page: 3 }} />)
    await waitFor(() => expect(screen.getByText((_, el) => el?.tagName === 'SPAN' && /^Page 2 \/ 4$/.test(el.textContent ?? ''))).toBeInTheDocument())
    await waitFor(() => expect(textLayerRenders).toContain('text of page 2'))
    expect(rendered).toContain(2)
    expect(screen.getByText('text of page 2').closest('.textLayer')).not.toBeNull()
    expect(screen.getByText(/page 1 of 2 in this section/)).toBeInTheDocument()

    await act(async () => fireEvent.click(screen.getByRole('button', { name: 'Next page' })))
    await act(async () => fireEvent.click(screen.getByRole('button', { name: 'Next page' })))
    await waitFor(() => expect(screen.getByText((_, el) => el?.tagName === 'SPAN' && /^Page 4 \/ 4$/.test(el.textContent ?? ''))).toBeInTheDocument())
    expect(screen.getByText(/Past the end of this section/)).toBeInTheDocument()
    await waitFor(() => expect(textLayerRenders).toContain('text of page 4'))
    // The previous page's text is gone: one page's text at a time.
    expect(screen.queryByText('text of page 2')).toBeNull()

    await act(async () => fireEvent.click(screen.getByRole('button', { name: 'Back to section start' })))
    await waitFor(() => expect(screen.getByText((_, el) => el?.tagName === 'SPAN' && /^Page 2 \/ 4$/.test(el.textContent ?? ''))).toBeInTheDocument())
  })
})
