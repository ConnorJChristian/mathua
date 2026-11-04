'use client'

import { useState } from 'react'
import KatexContent from './KatexContent'
import CodeEcho from './python/CodeEcho'
import { readingHref, type Walkthrough as WalkthroughData } from '../lib/walkthrough'

// The feedback a graded answer gets, on every surface (fork change,
// FSRS_PORT_PLAN 0c): the key, then the worked steps under "Show the steps",
// then the one-line reason and the reading.
//
// After a miss, or "I don't know", the first step is shown and the rest wait
// behind the toggle, so the learner sees where the path starts without being
// handed the whole solution to skim. After a right answer every step waits
// behind the toggle. A wrong answer that matches a step's value is marked on
// that step; a diagnosis without a step is a line under the steps.
//
// Light and dark mode: every colour is a theme token, and the open state is
// React state rather than a native <details>, so the toggle reads the same on
// every platform and the test can drive it.

interface Props {
  walkthrough: WalkthroughData
  correct: boolean
  dontKnow?: boolean
  /** The key is already shown by the host, or there is none to show (a proof). */
  hideAnswer?: boolean
  /** The key is printed output: echoed raw, never through Markdown or KaTeX. */
  outputAnswer?: boolean
  /** Links in the steps and the reason open in a new tab (bank items). */
  externalLinks?: boolean
}

export default function Walkthrough({ walkthrough: w, correct, dontKnow = false, hideAnswer = false, outputAnswer = false, externalLinks = false }: Props) {
  const [open, setOpen] = useState(false)
  const wrong = !correct || dontKnow
  const steps = w.steps ?? []
  // After a miss the first step is visible; the toggle shows the rest.
  const shown = wrong && !open ? steps.slice(0, 1) : open ? steps : []
  const hidden = steps.length - shown.length
  const diverged = wrong ? w.diverged_at : undefined
  const markStep = diverged && diverged.step > 0 ? diverged.step : 0
  const textClass = 'text-xs font-mono text-mathua-secondary whitespace-pre-wrap'
  return (
    <div data-walkthrough data-walkthrough-state={wrong ? 'wrong' : 'right'} className="mt-1.5 min-w-0">
      {!hideAnswer && w.answer && (
        <p className="font-mono text-xs text-mathua-primary">
          Answer: {outputAnswer ? <CodeEcho text={w.answer} inline /> : <KatexContent className="inline-block align-baseline [&_p]:inline [&_p]:m-0">{w.answer}</KatexContent>}
        </p>
      )}
      {steps.length > 0 && (
        <div className="mt-1.5">
          {shown.length > 0 && (
            <ol data-walkthrough-steps className="list-none space-y-1.5">
              {shown.map((step, i) => (
                <li key={i} data-walkthrough-step={i + 1} className={`flex gap-2 min-w-0 ${markStep === i + 1 ? 'border-l-2 border-mathua-red pl-2' : ''}`}>
                  <span className="shrink-0 font-mono text-[11px] text-mathua-muted tabular-nums">{i + 1}.</span>
                  <div className="min-w-0">
                    <KatexContent externalLinks={externalLinks} className={textClass}>{step}</KatexContent>
                    {markStep === i + 1 && (
                      <p data-diagnosis className="mt-1 font-mono text-[11px] text-mathua-red">Your answer stops here. {diverged?.reason}</p>
                    )}
                  </div>
                </li>
              ))}
            </ol>
          )}
          <button
            type="button"
            onClick={() => setOpen((o) => !o)}
            aria-expanded={open}
            className={`${shown.length > 0 ? 'mt-1.5' : ''} font-mono text-[11px] text-mathua-muted hover:text-mathua-blue underline underline-offset-2`}
          >
            {open ? 'Hide the steps' : hidden === steps.length ? `Show the steps (${steps.length})` : `Show the rest of the steps (${hidden})`}
          </button>
        </div>
      )}
      {diverged && markStep === 0 && diverged.reason && (
        <p data-diagnosis className="mt-2 font-mono text-[11px] text-mathua-muted">{diverged.reason}</p>
      )}
      {w.explain && (
        <div className="mt-1.5">
          <p className="font-mono text-[11px] uppercase tracking-wider text-mathua-muted">{wrong ? 'How' : 'Why'}</p>
          <KatexContent externalLinks={externalLinks} className={`mt-1 ${textClass}`}>{w.explain}</KatexContent>
        </div>
      )}
      {w.reading && (w.reading.url ? (
        <p className="mt-1.5 font-mono text-[11px]">
          <a href={readingHref(w.reading)} target="_blank" rel="noopener noreferrer" className="text-mathua-blue hover:underline underline-offset-2">
            Read: {w.reading.label || 'the section'} →
          </a>
        </p>
      ) : w.reading.label ? (
        <p className="mt-1.5 font-mono text-[11px] text-mathua-muted">Source: {w.reading.label}</p>
      ) : null)}
    </div>
  )
}
