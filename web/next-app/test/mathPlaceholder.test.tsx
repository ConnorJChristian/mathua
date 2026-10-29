import { describe, it, expect } from 'vitest'
import { render, waitFor } from '@testing-library/react'
import { MathAnswerInput } from '../components/math/MathInput'
import { latexPlaceholder } from '../components/math/plainAnswer'
import { formatForGradingType } from '../lib/answerFormat'

// MathLive renders its placeholder as LaTeX: the hosts' "Your answer" came out
// as italic math with the space swallowed ("Youranswer").
describe('the math box placeholder', () => {
  it('is plain text inside the math editor', async () => {
    const { container } = render(<MathAnswerInput value="" onChange={() => {}} gradingType="numeric" conceptId="arith.add.single" placeholder="Your answer" />)
    await waitFor(() => expect(container.querySelector('math-field')).toBeTruthy())
    expect(container.querySelector('math-field')!.getAttribute('placeholder')).toBe('\\text{Your answer}')
  })

  it('escapes TeX specials and leaves LaTeX alone', () => {
    expect(latexPlaceholder('Your answer...')).toBe('\\text{Your answer...}')
    expect(latexPlaceholder('50% off {now}')).toBe('\\text{50\\% off \\{now\\}}')
    expect(latexPlaceholder('\\frac{a}{b}')).toBe('\\frac{a}{b}')
    expect(latexPlaceholder(undefined)).toBeUndefined()
  })
})

describe('interval and set hints', () => {
  it('say how to type ∞, ∪ and ∅', () => {
    for (const t of ['interval', 'set']) {
      expect(formatForGradingType(t).hint).toContain('type oo for ∞, U for ∪, {} for ∅')
    }
  })
})
