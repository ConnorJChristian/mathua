import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { StepSourceLine } from '../components/StepSourceLine'

const path = [
  { book: 'hoy', label: 'Hoy' },
  { book: 'schaums-calc-econ', label: "Schaum's Calculus for Business" },
  { book: '', label: 'practice set' },
]

describe('StepSourceLine', () => {
  it('names the step sources in order and highlights the current one', () => {
    render(<StepSourceLine path={path} current="schaums-calc-econ" step={1} />)
    const line = screen.getByTestId('step-sources')
    expect(line.textContent).toBe("· Hoy → Schaum's Calculus for Business → practice set")
    const cur = line.querySelector('[aria-current="step"]')
    expect(cur?.textContent).toBe("Schaum's Calculus for Business")
    expect(line.querySelectorAll('[aria-current]').length).toBe(1)
  })

  it('highlights the practice set for a generated question', () => {
    render(<StepSourceLine path={path} current="" step={2} />)
    expect(screen.getByTestId('step-sources').querySelector('[aria-current="step"]')?.textContent).toBe('practice set')
  })

  it('shows nothing for a step with one source', () => {
    const { container } = render(<StepSourceLine path={[{ book: '', label: 'practice set' }]} current="" />)
    expect(container.textContent).toBe('')
    const { container: none } = render(<StepSourceLine />)
    expect(none.textContent).toBe('')
  })
})
