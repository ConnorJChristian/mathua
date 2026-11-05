// The one feedback contract a graded answer comes back with (fork change,
// FSRS_PORT_PLAN 0c; internal/walkthrough on the server): the key as shown,
// the worked steps, the one-line reason, where a wrong answer left the path
// and the topic's reading. Every surface renders it with components/Walkthrough.

import { API_BASE } from './apiBase'

export interface WalkthroughReading {
  label: string
  url: string
  page?: number
}

export interface WalkthroughDivergence {
  /** 1-based step the learner's answer matches; 0 when only the reason is known. */
  step: number
  reason: string
}

export interface Walkthrough {
  answer: string
  steps: string[]
  explain: string
  diverged_at?: WalkthroughDivergence
  reading?: WalkthroughReading
}

/** The markers a steps block travels under inside an explanation string. */
export const STEPS_MARKER = '<!--steps-->'
export const STEP_MARKER = '<!--step-->'

/**
 * An explanation string taken apart: the prose before the steps block and
 * the steps. A history row, or a server that predates the contract, carries
 * its steps this way.
 */
export function splitWalkthrough(explanation: string): { explain: string; steps: string[] } {
  const i = explanation.indexOf(STEPS_MARKER)
  if (i < 0) return { explain: explanation.trim(), steps: [] }
  const steps = explanation
    .slice(i + STEPS_MARKER.length)
    .split(STEP_MARKER)
    .map((s) => s.trim())
    .filter(Boolean)
  return { explain: explanation.slice(0, i).trim(), steps }
}

/**
 * The contract built on the client from an explanation and the shown key,
 * for a response without `walkthrough`.
 */
export function walkthroughFromExplanation(explanation: string, answer = ''): Walkthrough {
  const { explain, steps } = splitWalkthrough(explanation || '')
  return { answer, steps, explain }
}

/** The link a reading opens: a module's PDF at its page (through the API), or the web page. */
export function readingHref(r: WalkthroughReading): string {
  if (!r.url) return ''
  if (/^https?:\/\//.test(r.url)) return r.url
  return `${API_BASE}${r.url}${r.page ? `#page=${r.page}` : ''}`
}
