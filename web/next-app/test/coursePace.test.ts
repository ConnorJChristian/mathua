import { describe, it, expect } from 'vitest'
import { topicsPerWeek } from '../lib/dates'

// The Study page's "N topics/week to finish" paces the work due by the
// course's date. ECON-style plans list material taught after the final in
// modules marked after_due; counting those made the rate alarming and wrong.
describe('topicsPerWeek', () => {
  it('paces only the topics due by the date', () => {
    // 227 topics on the plan, 100 of them after the final, 7 mastered, 35 days left.
    const v = { total: 227, mastered: 7, due_total: 127, due_mastered: 7, days_left: 35 }
    expect(topicsPerWeek(v)).toBeCloseTo((120 / 35) * 7)
  })
  it('falls back to the whole plan without due counts', () => {
    expect(topicsPerWeek({ total: 20, mastered: 6, days_left: 14 })).toBeCloseTo(7)
  })
  it('is null with no date, no time or nothing left', () => {
    expect(topicsPerWeek({ total: 5, mastered: 1 })).toBeNull()
    expect(topicsPerWeek({ total: 5, mastered: 1, days_left: 0 })).toBeNull()
    expect(topicsPerWeek({ total: 9, mastered: 2, due_total: 4, due_mastered: 4, days_left: 10 })).toBeNull()
  })
})
