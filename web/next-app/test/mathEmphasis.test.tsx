import { describe, it, expect } from 'vitest'
import { render } from '@testing-library/react'
import KatexContent from '../components/KatexContent'

// Underscores and asterisks inside math spans are TeX, not Markdown. Before
// the fix, `\(\mathcal{L}_x\), \(\mathcal{L}_y\)` paired the two underscores
// as emphasis across the spans, and `\((x^*, y^*)\)` paired the asterisks, so
// KaTeX received half-expressions and the prompt rendered as `L \lambda` and
// `(x·y)`.
describe('KatexContent: math is opaque to Markdown emphasis', () => {
  // KaTeX renders HTML only (no MathML annotation), so a span that reached it
  // whole shows up as one .katex root with no raw TeX left in the prose.
  const roots = (c: HTMLElement) => c.querySelectorAll('.katex').length

  it('keeps subscripts in adjacent inline maths', () => {
    const { container } = render(
      <KatexContent>{'derivatives \\(\\mathcal{L}_x\\), \\(\\mathcal{L}_y\\), \\(\\mathcal{L}_\\lambda\\) and solve'}</KatexContent>,
    )
    expect(container.querySelector('em')).toBeNull()
    expect(roots(container)).toBe(3)
    expect(container.textContent).not.toContain('\\lambda')
  })

  it('keeps starred optima', () => {
    const { container } = render(<KatexContent>{'the optimal \\((x^*, y^*)\\) here'}</KatexContent>)
    expect(container.querySelector('em')).toBeNull()
    expect(roots(container)).toBe(1)
    expect(container.querySelector('.katex')?.textContent).toContain('∗')
  })

  it('still renders Markdown emphasis in prose', () => {
    const { container } = render(<KatexContent>{'a *stressed* word and $x_1$'}</KatexContent>)
    expect(container.querySelector('em')?.textContent).toBe('stressed')
  })
})
