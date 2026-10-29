import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import ReadingPanel from '../components/ReadingPanel'

// A topic read only in lecture notes (one HTML text, no PDF) used to render
// nothing, because the panel needed two texts or a PDF. The notes are the
// lesson's primary material, so they must show, embedded in place.
describe('ReadingPanel with lecture notes', () => {
  it('embeds a lone lecture-notes page', () => {
    render(
      <ReadingPanel
        texts={[{ key: 'notes:m', kind: 'file', book: 'Lecture notes', source: '5-Pie charts', url: '/api/modules/m/files/lectures/pie.html' }]}
      />,
    )
    const frame = screen.getByTitle('5-Pie charts')
    expect(frame.tagName).toBe('IFRAME')
    expect(frame.getAttribute('src')).toContain('/api/modules/m/files/lectures/pie.html')
    expect(screen.queryByRole('radiogroup')).toBeNull()
  })

  // The rendered notebook typesets its formulas with its own script
  // (MathJax), so the frame must allow scripts; and it must not share the
  // app's origin, or the page could read the login token from storage.
  it('lets the notes run their scripts without sharing the app origin', () => {
    render(
      <ReadingPanel
        texts={[{ key: 'notes:m', kind: 'file', book: 'Lecture notes', source: '5-Pie charts', url: '/api/modules/m/files/lectures/pie.html' }]}
      />,
    )
    const sandbox = screen.getByTitle('5-Pie charts').getAttribute('sandbox') ?? ''
    expect(sandbox.split(/\s+/)).toContain('allow-scripts')
    expect(sandbox).not.toContain('allow-same-origin')
  })
})
