import { describe, expect, it } from 'vitest'
import { quizXPLine } from '../lib/quizGate'

// Q59: the Study banner said "XP since the last one" before any check.
describe('quizXPLine', () => {
  it('has no "last one" before the first check', () => {
    expect(quizXPLine(59, false)).toBe('59 XP so far')
  })
  it('counts from the last check once there is one', () => {
    expect(quizXPLine(54, true)).toBe('54 XP since the last one')
  })
  it('keeps the old wording for a server that does not say', () => {
    expect(quizXPLine(50, undefined)).toBe('50 XP since the last one')
  })
})
