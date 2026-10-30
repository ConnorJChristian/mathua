import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import KatexContent from '../components/KatexContent'

// Regression for the division reference page: an array block with
// \text{\_\_\_} reached KaTeX as \text{\\\} because remark consumed the
// underscores as emphasis before rehype-raw decoded the span. The page then
// showed a raw TeX fallback instead of the long-division layout.
const divisionArray = String.raw`$$
\begin{array}{r}
{3\ } \\
\underset{\text{\_\_\_}}{\times 8} \\
{24\ } \\
\underset{\text{\_\_\_}}{+4} \\
28
\end{array}
$$`

describe('KatexContent markdown safety', () => {
  it('keeps underscores in text commands through the markdown pipeline', () => {
    const { container } = render(<KatexContent>{divisionArray}</KatexContent>)
    expect(container.querySelector('.math-fallback')).toBeNull()
    expect(container.querySelector('.katex-display')).not.toBeNull()
    expect(container.textContent).not.toContain('\\begin{array}')
    expect(container.textContent).toContain('28')
  })

  it('keeps subscripts and text underscores in inline math', () => {
    const { container } = render(<KatexContent>{String.raw`The term $x_1 + x_2$ and $\text{rate\_limit}$.`}</KatexContent>)
    expect(container.querySelector('.math-fallback')).toBeNull()
    expect(container.querySelectorAll('.katex').length).toBe(2)
  })
})
