'use client'

import { useEffect, useState, type ReactNode } from 'react'
import KatexContent from './KatexContent'
import LessonCredits from './LessonCredits'
import LessonDiagram from './LessonDiagram'
import ReportMenu from './ReportMenu'
import ReadingPanel from './ReadingPanel'
import { getLessonForConcept, getLessonKPs, type LessonKpsRes } from '../lib/api'
import { buildExampleBlocks, type ExampleBlock } from '../lib/lessonBlocks'
import { stripMathDelimiters } from '../lib/lessonMath'
import { concepts } from '../lib/conceptData'

// The reference material for one concept.
//
// This used to be `/study`: a browsable library of whole lessons, with its own search, its own
// domain index, and a link to it in every header, tab bar, sidebar and footer. ADR-047 closed
// that route and moved the material here, inside `/learn`, collapsed and link-free. ADR-054
// restored `/study` as a *context-bound* reference page — reachable for a prerequisite or a
// chosen topic, but not a catalogue, and not a route that awards anything — and this component
// is what both surfaces render.
//
// What is deliberately NOT here: a way to browse. There is no domain list, no lesson list, no
// search box, no sibling-concept chips. Navigation between concepts is the DAG neighbourhood the
// page around this component draws (prerequisites and dependents), and it is reference-only.
//
// Reading is also not evidence. There is no "mark as read", no completion, no "I studied this"
// — nothing below moves a mastery status, and the note at the foot says so explicitly, because a
// panel of authoritative-looking prose sitting next to the questions is exactly where a learner
// would expect one.

interface Props {
  conceptId: string
  /**
   * `panel` (default) is the collapsed reference drawn inside `/learn`. `page` is the expanded
   * reference `/study` renders, which also shows the full lesson article rather than only the
   * worked-example blocks.
   */
  variant?: 'panel' | 'page'
  /**
   * Actions to show under the missing-content message on the reference page, so a concept with no
   * material still offers its prerequisites and the practice hand-off rather than a dead end.
   */
  emptyActions?: ReactNode
}

// The server's honest placeholder for a step with no written explanation (`renderMinimalSlice`,
// internal/lessons/slice.go). The reference page replaces it with the full article rather than
// showing both, so the copy is recognised here as well as produced there.
const PLACEHOLDER_SLICE = /There isn't a written explanation/

export default function ConceptReference({ conceptId, variant = 'panel', emptyActions }: Props) {
  const [shard, setShard] = useState<LessonKpsRes | null>(null)
  const [lessonTitle, setLessonTitle] = useState<string | null>(null)
  const [body, setBody] = useState<string>('')
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    let cancelled = false
    setShard(null)
    setLessonTitle(null)
    setBody('')
    setFailed(false)
    getLessonKPs(conceptId)
      .then(res => { if (!cancelled) setShard(res) })
      .catch(() => { if (!cancelled) setFailed(true) })
    return () => { cancelled = true }
  }, [conceptId])

  const allBlocks: ExampleBlock[] = shard ? buildExampleBlocks(conceptId, shard) : []
  // `/learn` keeps every block, placeholder or not: they sit beside the questions and the panel is
  // an add-on. The reference page supersedes a placeholder with the full article, so it drops the
  // placeholder blocks rather than repeating "there is no explanation yet" beside the explanation.
  const blocks = variant === 'page'
    ? allBlocks.filter(b => !PLACEHOLDER_SLICE.test(b.body))
    : allBlocks
  // The panel shows the article only when there are no worked-example blocks, so it never
  // duplicates what `/learn` already put on screen. The reference page always wants the article.
  const wantsArticle = variant === 'page' || (shard !== null && allBlocks.length === 0)

  // The panel reaches for the article only for a concept whose shard carries no knowledge
  // points at all (rare: every concept in the corpus has a shard), so it stays off the common
  // path; the reference page always wants it. One request, by concept.
  useEffect(() => {
    if (!wantsArticle) return
    let cancelled = false
    getLessonForConcept(conceptId)
      .then((doc) => {
        if (cancelled || !doc) return
        setLessonTitle(doc.title)
        setBody(doc.body)
      })
      .catch(() => { if (!cancelled) setFailed(true) })
    return () => { cancelled = true }
  }, [wantsArticle, conceptId])

  const label = concepts.find(c => c.id === conceptId)?.label || conceptId
  const readingKp = shard?.kps.find(k => k.reading || k.texts)
  // Nothing to render: no shard content, no article. Say so honestly rather than showing a
  // titled empty panel, which reads as a finished lesson with nothing in it.
  const unavailable = shard !== null && !failed && blocks.length === 0 && body === ''

  const content = (
    <div className={variant === 'page' ? 'px-0' : 'px-4 pb-4 min-w-0 overflow-hidden'}>
      {failed && (
        <p role="alert" className="font-mono text-xs text-mathua-red">
          Couldn&apos;t load the reference for {label}. The questions below still work.
        </p>
      )}

      {unavailable && (
        <>
          <p role="note" className="font-mono text-xs text-mathua-secondary">
            Detailed study material is not available for this concept yet. You can still explore
            its prerequisites and related concepts, return to your original topic, or check
            whether practice is available.
          </p>
          {emptyActions && <div className="mt-4 flex flex-wrap gap-3">{emptyActions}</div>}
        </>
      )}

      {/* Fork change: a topic read from your own PDF opens it here too, at the section. */}
      {variant === 'page' && readingKp && <div className="mb-6"><ReadingPanel reading={readingKp.reading} texts={readingKp.texts} /></div>}

      {blocks.length > 0 && (
        <div data-testid="reference-blocks">
          <h3 className="font-mono text-[11px] text-mathua-muted mb-2 uppercase tracking-wider">
            {variant === 'page' && readingKp ? `Mathua's worked examples (${blocks.length})` : `Worked examples (${blocks.length})`}
          </h3>
          <div className="space-y-3">
            {blocks.map((b, i) => (
              <div key={b.key} className="border border-mathua-border p-4 min-w-0 overflow-hidden">
                <div className="flex items-start gap-2 min-w-0">
                  <div className="flex-1 min-w-0">
                    {b.body ? (
                      <details open={i === 0 && !(variant === 'page' && readingKp)}>
                        <summary className="font-mono text-xs text-mathua-primary cursor-pointer">
                          {i + 1}. {b.label ? stripMathDelimiters(b.label) : 'Worked example'}
                          {b.also.length > 0 && (
                            <span className="text-mathua-muted"> · also {b.also.map(stripMathDelimiters).join(', ')}</span>
                          )}
                        </summary>
                        {b.diagram && (
                          <div className="mt-3">
                            <LessonDiagram src={b.diagram} alt={`Worked diagram for ${label}`} />
                          </div>
                        )}
                        {b.subgoals.length > 0 && (
                          <ul className="mt-2 space-y-1">
                            {b.subgoals.map((sg, j) => (
                              <li key={j} className="font-mono text-[11px] text-mathua-secondary pl-3 relative before:content-['–'] before:absolute before:left-0">
                                {stripMathDelimiters(sg)}
                              </li>
                            ))}
                          </ul>
                        )}
                        <div className="mt-2 bg-mathua-code border border-mathua-border p-3 text-sm overflow-hidden">
                          <KatexContent>{b.body}</KatexContent>
                        </div>
                      </details>
                    ) : (
                      <>
                        <p className="font-mono text-xs text-mathua-primary">
                          {i + 1}. {b.label ? stripMathDelimiters(b.label) : 'Worked example'}
                          {b.also.length > 0 && (
                            <span className="text-mathua-muted"> · also {b.also.map(stripMathDelimiters).join(', ')}</span>
                          )}
                        </p>
                        {b.subgoals.length > 0 && (
                          <ul className="mt-2 space-y-1">
                            {b.subgoals.map((sg, j) => (
                              <li key={j} className="font-mono text-[11px] text-mathua-secondary pl-3 relative before:content-['–'] before:absolute before:left-0">
                                {stripMathDelimiters(sg)}
                              </li>
                            ))}
                          </ul>
                        )}
                      </>
                    )}
                  </div>
                  <ReportMenu
                    conceptId={conceptId}
                    lessonId={lessonTitle ?? undefined}
                    kind={b.diagram ? 'diagram' : 'worked_example'}
                    question={b.diagram ?? `${b.label}: ${b.body}`.slice(0, 2000)}
                    blockId={`${conceptId}/${b.index}`}
                  />
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {body && (
        <div className={blocks.length > 0 ? 'mt-4' : ''}>
          <h3 className="font-mono text-[11px] text-mathua-muted mb-2 uppercase tracking-wider">
            Full lesson{lessonTitle ? ` — ${lessonTitle}` : ''}
          </h3>
          <div className="border border-mathua-border bg-mathua-code p-3 text-sm overflow-hidden">
            <KatexContent>{body}</KatexContent>
          </div>
          <LessonCredits conceptId={conceptId} />
          <div className="mt-2 flex justify-end">
            <ReportMenu
              conceptId={conceptId}
              lessonId={lessonTitle ?? undefined}
              kind="lesson_body"
              question={body.slice(0, 2000)}
              blockId={`${conceptId}/body`}
            />
          </div>
        </div>
      )}

      <p className="mt-4 border-t border-mathua-border pt-3 font-mono text-[10px] text-mathua-muted">
        Reference explains. It does not count as progress — only the questions below move this
        concept.
      </p>
    </div>
  )

  if (variant === 'page') {
    // No outer card: the reference page already groups the material, and a box around a box is
    // the card-nesting the reading column should avoid.
    return <section className="min-w-0">{content}</section>
  }

  return (
    <details className="mb-4 border border-mathua-border bg-mathua-surface">
      <summary className="font-mono text-[11px] uppercase tracking-wider text-mathua-muted px-4 py-3 cursor-pointer hover:text-mathua-blue">
        Reference — the full lesson
      </summary>
      {content}
    </details>
  )
}
