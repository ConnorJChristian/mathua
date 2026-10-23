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

export type PythonKind = 'code' | 'mc' | 'output'

const STARTER_RE = /\n*```python starter(?: task=([^\s`]+))?\n([\s\S]*?)\n```[ \t]*/

export function pythonKind(question: string): PythonKind {
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
export function displayAnswer(answer: string): string {
  if (answer.startsWith('code:')) return ''
  if (answer.startsWith('mc:')) return answer.slice(3)
  if (answer.startsWith('out:')) return answer.slice(4)
  return answer
}

export function pythonHint(kind: PythonKind, timeLimit = 8): string {
  switch (kind) {
    case 'code':
      return `Run to try your code, then Check to grade it against hidden tests (${timeLimit}-second limit). Ctrl+Enter checks.`
    case 'mc':
      return 'Pick one option, then Check.'
    default:
      return 'Type exactly what it prints. Spacing inside arrays does not matter; everything else does.'
  }
}

export const isPythonGrading = (gradingType?: string | null) => gradingType === 'python'
