// Python-track questions are Markdown the server renders from its question bank
// (internal/pytrack). Every question says what kind it is, so each host (Learn,
// review, quiz) can pick the right input from the question alone:
//
//   - a code task carries its starter code in a fence tagged `python starter`,
//     which also names the task: ```python starter task=py.np.views#c1
//   - multiple choice ends with "A) …" options, like every other MC question
//   - anything else asks what a snippet prints
//
// The served answer is prefixed the same way (`code:`, `mc:`, `out:`); it is a
// key, not something to show a learner as-is.

import { parseChoices } from './choices'

export type PythonKind = 'code' | 'mc' | 'output' | 'recall' | 'math'

// A math item (e.g. an Anki practice problem): the prompt, then this marker,
// a JSON list of answer fields and "-->". Each field is one math input, graded
// on the server by Mathua's own math graders. internal/pytrack/track.go
// TypedMarker.
export const TYPED_MARKER = '<!--typed-answer '

export interface TypedField {
  /** LaTeX shown before the input as "label =". Empty for a single unlabelled answer. */
  label: string
  /** The grader, which also picks the math editor's keyboard: "numeric" or "expression". */
  grading: string
}

/** The answer fields of a math item, or [] for any other question. */
export function typedFields(question: string): TypedField[] {
  const at = question.indexOf(TYPED_MARKER)
  if (at < 0) return []
  const end = question.indexOf('-->', at)
  try {
    const raw = JSON.parse(question.slice(at + TYPED_MARKER.length, end < 0 ? undefined : end))
    if (!Array.isArray(raw)) return []
    return raw.map((f: { label?: unknown; grading?: unknown }) => ({
      label: typeof f.label === 'string' ? f.label : '',
      grading: typeof f.grading === 'string' ? f.grading : 'expression',
    }))
  } catch {
    return []
  }
}

/** The answers of a math item, one per field: the submitted string holds one per line. */
export function typedValues(value: string, n: number): string[] {
  const parts = value.split('\n')
  return Array.from({ length: n }, (_, i) => parts[i] ?? '')
}

/** A math item's submitted answers as one line: "x_0 = 60, y_0 = 40". */
export function typedEcho(question: string, answer: string): string {
  const fields = typedFields(question)
  const vals = typedValues(answer, Math.max(fields.length, 1))
  return vals.map((v, i) => (fields[i]?.label ? `${fields[i].label} = ${v}` : v)).join(', ')
}

// A recall card (a flashcard, e.g. imported from Anki): front, then this
// marker, then the back. internal/pytrack/track.go RecallMarker.
export const RECALL_MARKER = '<!--recall-back-->'

const STARTER_RE = /\n*```python starter(?: task=([^\s`]+))?\n([\s\S]*?)\n```[ \t]*/

export function pythonKind(question: string): PythonKind {
  if (question.includes(TYPED_MARKER)) return 'math'
  if (question.includes(RECALL_MARKER)) return 'recall'
  if (STARTER_RE.test(question)) return 'code'
  if (parseChoices(question)) return 'mc'
  return 'output'
}

/** The starter code for a code task, or '' when the question has none. */
export function starterCode(question: string): string {
  const m = STARTER_RE.exec(question)
  return m ? m[2] : ''
}

/** The task key the Run button sends, so the server can load the task's fixture. */
export function taskKey(question: string): string {
  const m = STARTER_RE.exec(question)
  return m && m[1] ? `code:${m[1]}` : ''
}

// The "A) …" block at the end of a multiple-choice stem (same shape as lib/choices.ts).
const OPTIONS_RE = /\n\nA\)\s[\s\S]*$/

/**
 * The question as shown. The starter block moves into the code box and the
 * options into the choice buttons, so neither is printed twice.
 */
export function displayQuestion(question: string): string {
  question = question.replace(CARD_MARKER, '').replace(/^\n/, '')
  const typed = question.indexOf(TYPED_MARKER)
  if (typed >= 0) return question.slice(0, typed).trimEnd()
  const cut = question.indexOf(RECALL_MARKER)
  if (cut >= 0) return question.slice(0, cut).trimEnd()
  let q = question.replace(STARTER_RE, '')
  if (parseChoices(q)) q = q.replace(OPTIONS_RE, '')
  return q.trimEnd()
}

/** An answer as plain text for a one-line echo: inline-code backticks removed. */
export function plainAnswer(s: string): string {
  return s.replace(/`([^`]*)`/g, '$1')
}

/**
 * The served answer as a learner should read it after answering. Code tasks have
 * no single answer (the explanation shows one solution), so they return ''.
 */
/** A recall card's back, revealed on request. */
export function recallBack(question: string): string {
  const cut = question.indexOf(RECALL_MARKER)
  return cut >= 0 ? question.slice(cut + RECALL_MARKER.length).trim() : ''
}

export function displayAnswer(answer: string): string {
  if (answer.startsWith('code:') || answer.startsWith('recall:') || answer.startsWith('math:')) return ''
  if (answer.startsWith('mc:')) return answer.slice(3)
  if (answer.startsWith('out:')) return answer.slice(4)
  return answer
}

export function pythonHint(kind: PythonKind, timeLimit = 8): string {
  switch (kind) {
    case 'code':
      return `Run to try your code, then Check to grade it against hidden tests (${timeLimit}-second limit). Ctrl+Enter checks.`
    case 'recall':
      return 'Answer in your head, reveal the back, then say honestly whether you had it.'
    case 'math':
      return 'Type your answer. Any equivalent form counts.'
    case 'mc':
      return 'Pick one option, then Check.'
    default:
      return 'Type exactly what it prints. Spacing inside arrays does not matter; everything else does.'
  }
}

/**
 * Question-bank concepts (internal/pytrack): the Python track and any module's
 * banks, including imported recall cards. "python" is the type's original name.
 */
export const isPythonGrading = (gradingType?: string | null) => gradingType === 'bank' || gradingType === 'python'

/**
 * One of your flashcards served in a generated topic's practice
 * (internal/engine/cards.go). It takes the card's own input, whatever the
 * topic's grading type.
 */
export const CARD_MARKER = '<!--card-->'
export const isCardQuestion = (question?: string | null) => !!question && question.startsWith(CARD_MARKER)
/** Whether a served question takes the question-bank inputs. */
export const isBankServed = (gradingType?: string | null, question?: string | null) =>
  isPythonGrading(gradingType) || isCardQuestion(question)
