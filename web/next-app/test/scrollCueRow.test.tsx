import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import ScrollCueRow, { hasMoreRight } from '../components/ScrollCueRow'

// Q28: at 390px the course tab row cut "Grades" off with nothing to say the
// row scrolls. The cue shows while there is more to the right.
describe('ScrollCueRow', () => {
  it('knows when the row has more to the right', () => {
    expect(hasMoreRight({ scrollLeft: 0, scrollWidth: 374, clientWidth: 358 })).toBe(true)
    expect(hasMoreRight({ scrollLeft: 16, scrollWidth: 374, clientWidth: 358 })).toBe(false)
    expect(hasMoreRight({ scrollLeft: 0, scrollWidth: 358, clientWidth: 358 })).toBe(false)
  })
  it('keeps the tabs in a labelled, scrollable nav', () => {
    render(<ScrollCueRow label="Course sections"><button type="button">Home</button><button type="button">Grades</button></ScrollCueRow>)
    const nav = screen.getByRole('navigation', { name: 'Course sections' })
    expect(nav.className).toContain('overflow-x-auto')
    expect(nav.textContent).toBe('HomeGrades')
  })
})
