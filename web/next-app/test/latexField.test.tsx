import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import LatexField, { latexToPlain } from '../components/math/LatexField'

// Typed LaTeX is submitted as the plain answer the visual editor would send:
// MathLive's converter, then the same toPlainAnswer shim. Without the converter
// the raw LaTeX goes through, which the server's NormalizeTypedMath reads.
describe('latexToPlain', () => {
  it('runs the converter output through the plain-answer shim', () => {
    const convert = (l: string) => (l === '\\frac{1}{2}' ? '(1)/(2)' : l)
    expect(latexToPlain('\\frac{1}{2}', convert)).toBe('1/2')
  })
  it('sends raw LaTeX when the converter is unavailable or throws', () => {
    expect(latexToPlain('\\frac{1}{2}', null)).toBe('\\frac{1}{2}')
    expect(latexToPlain('x^{2}', () => { throw new Error('boom') })).toBe('x^{2}')
  })
  it('sends nothing for an empty field', () => {
    expect(latexToPlain('   ', null)).toBe('')
  })
})

describe('LatexField', () => {
  it('reports what is typed and previews it', () => {
    const onChange = vi.fn()
    render(<LatexField value="" onChange={onChange} />)
    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'x^{2}' } })
    expect(onChange).toHaveBeenLastCalledWith(expect.objectContaining({ latex: 'x^{2}' }))
    expect(screen.getByTestId('latex-preview').textContent).not.toContain('Preview appears here')
  })
  it('submits on Enter', () => {
    const onSubmit = vi.fn()
    render(<LatexField value="" onChange={() => {}} onSubmit={onSubmit} />)
    fireEvent.keyDown(screen.getByRole('textbox'), { key: 'Enter' })
    expect(onSubmit).toHaveBeenCalled()
  })
})
