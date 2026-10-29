import { describe, it, expect } from 'vitest'
import { backTarget, courseReturnLabel, courseReturnPath, parseReturnStack, withReturnStack } from '../lib/learnReturn'

// A topic opened from a course page goes back to that page (and tab), not to the profile.
describe('course return targets', () => {
  it('round-trips a course page through the return stack', () => {
    const href = withReturnStack('/study?concept=py.mpl.pie', [courseReturnPath('plan.econ5283', 'modules')])
    const stack = parseReturnStack(new URL(href, 'http://x').searchParams)
    expect(stack).toEqual(['/course?id=plan.econ5283&tab=modules'])
    expect(backTarget(stack)?.href).toBe('/course?id=plan.econ5283&tab=modules')
  })

  it('labels course targets in words and leaves topic targets alone', () => {
    expect(courseReturnLabel('/course?id=plan.econ5283&tab=modules')).toBe('course modules')
    expect(courseReturnLabel('/course?id=plan.econ5283')).toBe('the course')
    expect(courseReturnLabel('/study?concept=calc.deriv.chain_rule')).toBeNull()
  })

  it('omits the tab for the home tab', () => {
    expect(courseReturnPath('plan.marx', 'home')).toBe('/course?id=plan.marx')
  })
})
