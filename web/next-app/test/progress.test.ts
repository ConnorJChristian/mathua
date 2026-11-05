import { describe, it, expect } from 'vitest'
import { isMastered, isDueForReview, prereqMet, countByDomain, countOverall, type CatalogueConcept } from '../lib/progress'
import type { ConceptProgress } from '../lib/api'

const p = (status: string, completed?: boolean, due?: boolean): ConceptProgress => ({ status, streak: 0, completed, due })

// a ──► b ──► c, with d independent.
const catalogue: CatalogueConcept[] = [
  { id: 'a', domain: 'x' },
  { id: 'b', domain: 'x', prerequisites: ['a'] },
  { id: 'c', domain: 'x', prerequisites: ['b'] },
  { id: 'd', domain: 'y' },
]

describe('isMastered', () => {
  it('is case-insensitive, because the stored value is uppercase and older payloads are not', () => {
    expect(isMastered(p('MASTERED'))).toBe(true)
    expect(isMastered(p('mastered'))).toBe(true)
    expect(isMastered(p('Mastered'))).toBe(true)
  })

  it('does not treat decaying as mastered', () => {
    // DECAYING is what the server reports once a review goes stale. Counting it
    // as mastered is exactly the bug this shared helper exists to prevent.
    expect(isMastered(p('DECAYING'))).toBe(false)
  })

  it('is false for every other status and for nothing at all', () => {
    for (const s of ['UNSEEN', 'LEARNING', 'PRACTICING', '']) {
      expect(isMastered(p(s))).toBe(false)
    }
    expect(isMastered(undefined)).toBe(false)
  })
})

describe('prereqMet', () => {
  it('accepts a mastered prerequisite', () => {
    expect(prereqMet({ a: p('MASTERED') }, 'a')).toBe(true)
  })

  it('accepts a completed-but-unmastered prerequisite', () => {
    expect(prereqMet({ a: p('LEARNING', true) }, 'a')).toBe(true)
  })

  it('rejects an untouched one, and an absent one', () => {
    expect(prereqMet({ a: p('LEARNING') }, 'a')).toBe(false)
    expect(prereqMet({}, 'a')).toBe(false)
  })
})

describe('countByDomain', () => {
  it('buckets the catalogue and buckets a concept once', () => {
    // The buckets are mutually exclusive: a mastered concept is not also
    // counted as learning or locked. With b mastered and c practicing, a has
    // no prerequisites and c's are met, so nothing in x is locked.
    const got = countByDomain(catalogue, { b: p('MASTERED'), c: p('PRACTICING') })
    expect(got.get('x')).toEqual({ total: 3, mastered: 1, dueForReview: 0, fading: 0, learning: 1, completed: 0, locked: 0 })
    expect(got.get('y')).toEqual({ total: 1, mastered: 0, dueForReview: 0, fading: 0, learning: 0, completed: 0, locked: 0 })
  })

  it('locks a concept whose prerequisite is unmet', () => {
    // b needs a, so with nothing done b is locked.
    expect(countByDomain(catalogue, {}).get('x')?.locked).toBe(2)
    // a mastered unblocks b but not c.
    expect(countByDomain(catalogue, { a: p('MASTERED') }).get('x')?.locked).toBe(1)
    expect(countByDomain(catalogue, { a: p('MASTERED'), b: p('MASTERED') }).get('x')?.locked).toBe(0)
  })

  it('treats a completed prerequisite as met for the lock count', () => {
    expect(countByDomain(catalogue, { a: p('LEARNING', true) }).get('x')?.locked).toBe(1)
  })
})

describe('countOverall', () => {
  it('reports total, mastered and unlocked', () => {
    const got = countOverall(catalogue, { a: p('MASTERED'), b: p('PRACTICING') })
    expect(got.total).toBe(4)
    expect(got.mastered).toBe(1)
    expect(got.learning).toBe(1)
    // b and d are reachable; c is not, because b is only practicing.
    expect(got.locked).toBe(1)
    expect(got.unlocked).toBe(3)
  })

  it('reports unlocked as the complement of locked, for any input', () => {
    for (const progress of [
      {},
      { a: p('MASTERED') },
      { a: p('MASTERED'), b: p('MASTERED') },
      { c: p('LEARNING') },
      { d: p('DECAYING') },
    ]) {
      const got = countOverall(catalogue, progress)
      expect(got.unlocked).toBe(got.total - got.locked)
    }
  })

  it('never reports more unlocked than total', () => {
    const got = countOverall(catalogue, {})
    expect(got.unlocked).toBeLessThanOrEqual(got.total)
    expect(got.unlocked).toBeGreaterThanOrEqual(0)
  })
})

// Decay is computed at read time by the server, so a stale concept arrives as
// DECAYING rather than MASTERED. The two rules then have to disagree in a
// specific way: DECAYING is not mastery, but it does still unlock successors.
describe('decay', () => {
  it('is not mastery', () => {
    expect(isMastered(p('MASTERED'))).toBe(true)
    expect(isMastered(p('DECAYING'))).toBe(false)
  })

  it('still satisfies a prerequisite, matching scheduler.prereqsMet', () => {
    // If this were false, serving a decayed concept would relock everything
    // downstream of it and disagree with the scheduler that chose to serve it.
    expect(prereqMet({ a: p('DECAYING') }, 'a')).toBe(true)
    expect(prereqMet({ a: p('MASTERED') }, 'a')).toBe(true)
    expect(prereqMet({ a: p('LEARNING') }, 'a')).toBe(false)
  })

  it('does not change the locked count when a concept decays', () => {
    const fresh = countByDomain(catalogue, { a: p('MASTERED'), b: p('PRACTICING') })
    const stale = countByDomain(catalogue, { a: p('DECAYING'), b: p('PRACTICING') })
    expect(stale.get('x')?.locked).toBe(fresh.get('x')?.locked)
  })

  it('keeps the mastered count and reports the review debt separately', () => {
    // This used to assert the opposite: that DECAYING *lowered* `mastered`, "which is
    // the point of reporting it" — a deliberate choice not to over-claim.
    //
    // It over-claimed in the other direction instead. The graph gives a decayed concept
    // a full mastery bar, so a learner with 10 decayed and 32 fresh read "32 concepts
    // mastered" on the profile next to 42 full bars on the graph, with nothing to
    // explain the gap. Two numbers for one fact is the problem this file exists to
    // prevent.
    //
    // So `mastered` now means "competence demonstrated and not yet stale" — which is
    // what the word means — and decay is reported in its own right rather than by
    // silently shrinking it. The over-claiming the original was avoiding is prevented
    // by *showing* the review debt, not by making the headline number disagree with the
    // map.
    const fresh = countOverall(catalogue, { a: p('MASTERED') })
    const stale = countOverall(catalogue, { a: p('DECAYING') })

    expect(fresh.mastered).toBe(1)
    expect(fresh.fading).toBe(0)
    expect(stale.mastered).toBe(1)
    expect(stale.fading).toBe(1)

    // And the two still disagree where they should: `isMastered` is the strict
    // question, and it is false for a stale concept.
    expect(isMastered(p('DECAYING'))).toBe(false)
  })

  it('counts due for review from the server\'s rule, not from decay', () => {
    // DECAYING is the pacing notion (a fortnight since the last review). Whether a
    // review is *due* is the server's one rule (review.IsDue, exclusions applied),
    // carried as `due` on the row. A topic with a long interval reviewed over a
    // fortnight ago is decaying and not due; a topic reviewed yesterday whose date
    // has passed is due and not decaying. The profile used to count decay as
    // "due for review" and link it to /review (FSRS Phase 0 review, F1).
    const decayedNotDue = countOverall(catalogue, { a: p('DECAYING', undefined, false) })
    expect(decayedNotDue.dueForReview).toBe(0)
    expect(decayedNotDue.fading).toBe(1)

    const dueNotDecayed = countOverall(catalogue, { a: p('MASTERED', undefined, true) })
    expect(dueNotDecayed.dueForReview).toBe(1)
    expect(dueNotDecayed.fading).toBe(0)
    expect(dueNotDecayed.mastered).toBe(1)

    const both = countOverall(catalogue, { a: p('DECAYING', undefined, true) })
    expect(both.dueForReview).toBe(1)
    expect(both.fading).toBe(1)

    // A payload predating the field reads as not due, never as due.
    expect(isDueForReview(p('DECAYING'))).toBe(false)
    expect(countByDomain(catalogue, { a: p('DECAYING') }).get('x')?.dueForReview).toBe(0)
  })
})
