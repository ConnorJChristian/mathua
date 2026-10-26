'use client'

// The grade of a written proof (internal/proof): the score out of 7, the
// marking scheme's milestones as a checklist, the first error the judge found
// and its feedback. Shown by every answer host after a proof is graded.

import type { ProofReport as Report } from '../lib/api'
import KatexContent from './KatexContent'

const MET_LABEL: Record<string, string> = { yes: 'Met', partial: 'Partly', no: 'Missing' }

// flashcard marks a written answer to a flashcard (internal/recallproof): its
// grade is recorded as the learner's own mark would be.
export default function ProofReport({ report, flashcard = false }: { report: Report; flashcard?: boolean }) {
  const runs = report.runs ?? []
  return (
    <div data-proof-report className="mt-3 border border-mathua-border bg-mathua-bg px-4 py-3">
      <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
        <p className="font-mono text-lg text-mathua-primary">
          {report.score}/{report.max}
        </p>
        <p className={`font-mono text-xs ${report.fully_correct ? 'text-green-400' : 'text-mathua-muted'}`}>
          {report.fully_correct ? 'Fully correct' : 'Not yet a complete proof'}
        </p>
        {runs.length > 1 && (
          <p className="font-mono text-[11px] text-mathua-muted">median of {runs.length} gradings: {runs.join(', ')}</p>
        )}
        {report.cached && <p className="font-mono text-[11px] text-mathua-muted">same proof as before: earlier grade</p>}
      </div>
      {report.milestones?.length > 0 && (
        <ul aria-label="Marking scheme" className="mt-3 flex flex-col gap-1.5">
          {report.milestones.map(m => (
            <li key={m.id} className="flex gap-2 text-xs">
              <span
                className={`shrink-0 w-16 font-mono text-[11px] ${
                  m.met === 'yes' ? 'text-green-400' : m.met === 'partial' ? 'text-yellow-400' : 'text-red-400'
                }`}
              >
                {MET_LABEL[m.met] ?? m.met}
              </span>
              <span className="shrink-0 w-10 font-mono text-[11px] text-mathua-muted">{m.points} pt</span>
              <span className="min-w-0">
                <KatexContent className="text-mathua-secondary">{m.desc}</KatexContent>
                {m.note ? <span className="block font-mono text-[11px] text-mathua-muted">{m.note}</span> : null}
              </span>
            </li>
          ))}
        </ul>
      )}
      {report.first_error && (
        <div className="mt-3">
          <p className="font-mono text-[11px] uppercase tracking-wider text-mathua-muted">First error</p>
          <KatexContent className="mt-0.5 text-xs text-red-400">{report.first_error}</KatexContent>
        </div>
      )}
      {report.feedback && (
        <div className="mt-3">
          <p className="font-mono text-[11px] uppercase tracking-wider text-mathua-muted">Feedback</p>
          <KatexContent className="mt-0.5 text-xs text-mathua-secondary whitespace-pre-wrap">{report.feedback}</KatexContent>
        </div>
      )}
      {flashcard ? (
        <p className="mt-3 font-mono text-[11px] text-mathua-muted">
          Recorded as if you had marked the card yourself: practice, not mastery evidence. The card&apos;s answer is below.
        </p>
      ) : !report.counts_for_mastery && (
        <p className="mt-3 font-mono text-[11px] text-mathua-muted">
          Feedback only: proof grades do not count toward mastery until the grader is calibrated.
        </p>
      )}
    </div>
  )
}
