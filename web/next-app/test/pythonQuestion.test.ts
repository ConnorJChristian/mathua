import { describe, it, expect } from 'vitest'
import { displayAnswer, displayQuestion, plainAnswer, pythonKind, recallBack, starterCode, taskKey, typedEcho, typedFields, typedValues } from '../lib/pythonQuestion'
import { prepareLessonMath } from '../lib/lessonMath'
import { TRACKS, trackOfDomain, withTrack } from '../lib/track'

// The shapes internal/pytrack/track.go serves. Kept literal so a change to the
// server's format fails here rather than as a blank code box in the browser.
const CODE_Q = 'Make `as_int` from `raw`.\n\n```python starter task=py.np.dtypes#c2\nimport numpy as np\nas_int = ...\n```'
const MC_Q = 'What is the dtype of `np.array([3, 4.0])`?\n\nA) `int64`\n\nB) `float64`\n\nC) `object`'
const OUT_Q = 'What does this print?\n\n```python\nprint(1 + 1)\n```'

describe('python question shapes', () => {
  it('tells the three kinds apart from the question alone', () => {
    expect(pythonKind(CODE_Q)).toBe('code')
    expect(pythonKind(MC_Q)).toBe('mc')
    expect(pythonKind(OUT_Q)).toBe('output')
  })

  it('moves the starter into the code box and names the task for Run', () => {
    expect(starterCode(CODE_Q)).toBe('import numpy as np\nas_int = ...')
    expect(taskKey(CODE_Q)).toBe('code:py.np.dtypes#c2')
    expect(displayQuestion(CODE_Q)).toBe('Make `as_int` from `raw`.')
  })

  it('prints multiple-choice options once, as buttons', () => {
    expect(displayQuestion(MC_Q)).toBe('What is the dtype of `np.array([3, 4.0])`?')
    expect(displayQuestion(OUT_Q)).toBe(OUT_Q)
  })

  it('never shows a learner the answer key', () => {
    expect(displayAnswer('code:py.np.dtypes#c2')).toBe('')
    expect(displayAnswer('mc:`float64`')).toBe('`float64`')
    expect(displayAnswer('out:[1 2 3]')).toBe('[1 2 3]')
    expect(plainAnswer('`float64`')).toBe('float64')
  })
})

describe('recall cards', () => {
  const RECALL_Q = 'What does OLS minimise?\n\n<!--recall-back-->\n\nThe sum of squared residuals.'
  it('shows the front, keeps the back for the reveal, and hides the key', () => {
    expect(pythonKind(RECALL_Q)).toBe('recall')
    expect(displayQuestion(RECALL_Q)).toBe('What does OLS minimise?')
    expect(recallBack(RECALL_Q)).toBe('The sum of squared residuals.')
    expect(displayAnswer('recall:anki.woo.ch02#n1')).toBe('')
  })
})

describe('code in lesson Markdown', () => {
  it('leaves python fences untouched by the LaTeX passes', () => {
    const code = '```python\nprice = "$5"\nx = a[(b < 3) & (c > 1)]\ns = r"\\(x\\)"\n```'
    expect(prepareLessonMath('Look:\n\n' + code)).toBe('Look:\n\n' + code)
  })

  it('still renders math outside the fence', () => {
    expect(prepareLessonMath('$x^2$')).toContain('math-inline')
  })
})

describe('track parameter', () => {
  it('only marks non-math requests, so math requests are unchanged', () => {
    expect(withTrack('/api/next', 'math')).toBe('/api/next')
    expect(withTrack('/api/next', 'python')).toBe('/api/next?track=python')
    expect(withTrack('/api/next?exclude=a', 'python')).toBe('/api/next?exclude=a&track=python')
  })

  it('takes its tracks from the enabled modules', () => {
    expect(TRACKS[0].id).toBe('math')
    expect(TRACKS.some(t => t.id === 'python')).toBe(true)
    expect(trackOfDomain('python_numpy')).toBe('python')
    expect(trackOfDomain('calculus')).toBe('math')
  })
})

describe('math items (typed answers)', () => {
  const q = 'Find the stationary point.\n\n<!--typed-answer [{"label":"x_0","grading":"numeric"},{"label":"y_0","grading":"numeric"}]-->'

  it('is recognised before any other kind', () => {
    expect(pythonKind(q)).toBe('math')
  })

  it('reads the fields and hides the marker', () => {
    expect(typedFields(q)).toEqual([
      { label: 'x_0', grading: 'numeric' },
      { label: 'y_0', grading: 'numeric' },
    ])
    expect(displayQuestion(q)).toBe('Find the stationary point.')
    expect(typedFields('no marker here')).toEqual([])
    expect(typedFields('x\n\n<!--typed-answer not json-->')).toEqual([])
  })

  it('keeps one answer per field, one per line', () => {
    expect(typedValues('15\n7/2', 2)).toEqual(['15', '7/2'])
    expect(typedValues('15', 2)).toEqual(['15', ''])
    expect(typedEcho(q, '15\n7/2')).toBe('x_0 = 15, y_0 = 7/2')
  })

  it('never shows the served answer key', () => {
    expect(displayAnswer('math:anki.x#n1c0')).toBe('')
  })
})

describe('flashcards served on a generated topic', () => {
  it('take the bank inputs and hide the marker', async () => {
    const m = await import('../lib/pythonQuestion')
    const q = m.CARD_MARKER + '\nFront of the card\n\n<!--recall-back-->\n\nBack'
    expect(m.isBankServed('numeric', q)).toBe(true)
    expect(m.isBankServed('numeric', 'What is 2+2?')).toBe(false)
    expect(m.pythonKind(q)).toBe('recall')
    expect(m.displayQuestion(q)).toBe('Front of the card')
  })
})
