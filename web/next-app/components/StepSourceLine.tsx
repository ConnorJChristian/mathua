'use client'

import type { StepPlace } from '../lib/api'

// Fork change (FORK.md, "The order of practice inside a topic"): where the
// learner is in the scaffold step, "Hoy → Schaum's → practice set", with the
// current source highlighted. The server sends the step's sources in order;
// a step with one source has no line.
export function StepSourceLine({ path, current, step }: { path?: StepPlace[]; current?: string; step?: number }) {
  if (!path || path.length < 2) return null
  const at = path.findIndex((p) => p.book === (current ?? ''))
  return (
    <span data-testid="step-sources" title={step ? `Scaffold step ${step}: where this question comes from` : undefined}>
      ·{' '}
      {path.map((p, i) => (
        <span key={`${p.book}|${i}`}>
          {i > 0 && <span aria-hidden="true"> → </span>}
          <span
            data-current={i === at ? 'true' : undefined}
            aria-current={i === at ? 'step' : undefined}
            className={i === at ? 'text-mathua-blue font-semibold' : undefined}
          >
            {p.label}
          </span>
        </span>
      ))}
    </span>
  )
}
