'use client'

import KatexContent from './KatexContent'
import CodeEcho from './python/CodeEcho'
import Walkthrough from './Walkthrough'
import type { PlacementQuestion } from '../lib/api'
import { displayQuestion, isBankAnswer } from '../lib/pythonQuestion'

// Fork change (FSRS_PORT_PLAN 0c): a placement test teaches nothing while it
// runs, but once it is over the results list every question asked, what was
// answered, and the walkthrough to the key. A miss shows its first step; a
// right answer keeps its steps behind the toggle.

export default function PlacementWalkthroughs({ questions }: { questions: PlacementQuestion[] }) {
  if (questions.length === 0) return null
  const missed = questions.filter((q) => !q.correct).length
  return (
    <section aria-label="Your answers" data-placement-walkthroughs className="border border-mathua-border bg-mathua-surface text-left">
      <p className="px-4 pt-3 font-mono text-[10px] uppercase tracking-wider text-mathua-muted">
        Your answers · {questions.length} asked, {missed} missed
      </p>
      <ol className="divide-y divide-mathua-border">
        {questions.map((q, i) => {
          const bank = isBankAnswer(q.walkthrough?.answer) || q.question.includes('<!--')
          return (
            <li key={`${q.concept_id}-${i}`} data-placement-question data-correct={q.correct} className="px-4 py-3 min-w-0">
              <p className={`font-mono text-xs ${q.correct ? 'text-mathua-green' : 'text-mathua-red'}`}>
                {q.correct ? '✓' : '✗'} <span className="text-mathua-primary">{q.concept_name}</span>
              </p>
              <KatexContent className="mt-1 text-sm text-mathua-primary font-mono whitespace-pre-wrap">{bank ? displayQuestion(q.question) : q.question}</KatexContent>
              <p className="mt-1 font-mono text-[11px] text-mathua-muted">
                {q.dont_know ? 'No answer.' : <>you answered <CodeEcho text={q.answer} inline /></>}
              </p>
              <Walkthrough walkthrough={q.walkthrough} correct={q.correct} dontKnow={q.dont_know} />
            </li>
          )
        })}
      </ol>
    </section>
  )
}
