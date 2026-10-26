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

export type PythonKind = 'code' | 'mc' | 'output' | 'recall' | 'math' | 'proof'

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

// A written proof (internal/proof): the statement, then this marker. The
// answer is the learner's proof, Markdown with LaTeX, graded on the server
// against a marking scheme by a language-model judge. internal/proof/bank.go
// Marker.
export const PROOF_MARKER = '<!--proof-answer-->'

/** Whether a served question asks for a written proof. */
export const isProofQuestion = (question?: string | null) => !!question && question.includes(PROOF_MARKER)

// A writing problem (internal/proof, kind "writing"): served like a proof,
// with this marker after PROOF_MARKER. internal/proof/bank.go WritingMarker.
export const WRITING_MARKER = '<!--writing-answer-->'

/** Whether a served question asks for a piece of academic writing. */
export const isWritingQuestion = (question?: string | null) =>
  !!question && question.includes(PROOF_MARKER) && question.includes(WRITING_MARKER)

/** Words in a text, as a writer counts them: runs of non-space characters. */
export const wordCount = (text: string) => (text.trim() ? text.trim().split(/\s+/).length : 0)

/** The hint under a writing problem's box. */
export const WRITING_HINT =
  'Write your text in the box. It is graded against a rubric (genre, moves, register, accuracy, clarity), which takes up to a minute.'

// A flashcard that asks for a proof or derivation, served for a written
// answer (internal/pytrack/written.go): front, PROOF_MARKER, RECALL_MARKER,
// back. It takes the proof box and is graded like a proof, but the learner
// may instead reveal the back and mark it themselves, submitting "right" or
// "wrong" as for any flashcard.
export const isWrittenRecall = (question?: string | null) =>
  !!question && question.includes(PROOF_MARKER) && question.includes(RECALL_MARKER)

/** A written-answer flashcard the learner chose to mark themselves. */
export const isSelfMarked = (question: string | null | undefined, answer: string) =>
  isWrittenRecall(question) && (answer === 'right' || answer === 'wrong')

// A recall card (a flashcard, e.g. imported from Anki): front, then this
// marker, then the back. internal/pytrack/track.go RecallMarker.
export const RECALL_MARKER = '<!--recall-back-->'

const STARTER_RE = /\n*```python starter(?: task=([^\s`]+))?\n([\s\S]*?)\n```[ \t]*/

/**
 * The kind an answered question is reported as: a written-answer flashcard
 * the learner marked themselves is a recall card, not a graded proof.
 */
export function answeredKind(question: string, answer: string): PythonKind {
  return isSelfMarked(question, answer) ? 'recall' : pythonKind(question)
}

export function pythonKind(question: string): PythonKind {
  if (question.includes(PROOF_MARKER)) return 'proof'
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
  const proof = question.indexOf(PROOF_MARKER)
  if (proof >= 0) return question.slice(0, proof).trimEnd()
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
  if (answer.startsWith('code:') || answer.startsWith('recall:') || answer.startsWith('math:') || answer.startsWith('proof:')) return ''
  if (answer.startsWith('mc:')) return answer.slice(3)
  if (answer.startsWith('out:')) return answer.slice(4)
  return answer
}

/** The hint under a written-answer flashcard's box. */
export const WRITTEN_RECALL_HINT =
  'Write your answer: it is graded against a marking scheme, which takes up to a minute. Or reveal the answer and mark it yourself.'

export function pythonHint(kind: PythonKind, timeLimit = 8): string {
  switch (kind) {
    case 'code':
      return `Run to try your code, then Check to grade it against hidden tests (${timeLimit}-second limit). Ctrl+Enter checks.`
    case 'recall':
      return 'Answer in your head, reveal the back, then say honestly whether you had it.'
    case 'math':
      return 'Type your answer. Any equivalent form counts.'
    case 'proof':
      return 'Write a complete proof. LaTeX between \\( and \\) is rendered in the preview. Grading takes up to a minute.'
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
/** A bank answer carries its kind as a prefix (grader.IsBankAnswer), so a
 * served question can be recognised as bank-served even when the concept is
 * missing from the bundled concept list (a module the bundle was built
 * without), fork change. */
export const isBankAnswer = (answer?: string | null) =>
  !!answer && /^(mc|out|code|recall|math|proof):/.test(answer)

/** Whether a served question takes the question-bank inputs. */
export const isBankServed = (gradingType?: string | null, question?: string | null, answer?: string | null) =>
  isPythonGrading(gradingType) || isCardQuestion(question) || isBankAnswer(answer)
