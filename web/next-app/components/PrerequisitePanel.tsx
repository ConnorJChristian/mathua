'use client'

import Link from 'next/link'
import type { EligibilityPrereq } from '../lib/api'
import { studyConceptPath, withReturnStack, type ReturnEntry } from '../lib/learnReturn'

/**
 * The prerequisite list, shared by the learning loop and the curriculum browser.
 *
 * Its one non-negotiable property: **every prerequisite is openable.** A prerequisite's state
 * describes mastery, and mastery is not a navigation lock — the material behind a locked or
 * upcoming prerequisite is reference, and reading it is not evidence, so there is no reason to
 * strand a learner on a dead "Locked" label. Before ADR-047's correction this list returned no
 * link for `locked`/`upcoming`, which is exactly the dead end that made a prerequisite
 * unreachable.
 *
 * Every row opens the prerequisite's reference page. Establishing mastery happens on the study
 * page's separate *Practise this concept* control, which goes through the normal `/learn` flow
 * and the backend's own eligibility gate — never through this component. Reading a prerequisite
 * must not unlock the target, and it must not be mistaken for practice.
 */

/** Every prerequisite opens its reference page; mastery is not an access rule. */
export function prereqHref(p: EligibilityPrereq, returnStack: ReturnEntry[] = []): string {
  return withReturnStack(studyConceptPath(p.conceptId), returnStack)
}

export function prereqCta(_p: EligibilityPrereq): string {
  return 'Study →'
}

/** The learner-facing words for a prerequisite's state, from the server's verdict. */
export function prereqStateWord(p: EligibilityPrereq): string {
  switch (p.state) {
    case 'mastered':
      return 'Mastered'
    case 'due_for_review':
      return 'Mastered · due a check'
    case 'fading':
      return 'Mastered · fading'
    case 'in_progress':
      return 'In progress'
    case 'unlocked':
    case 'upcoming':
      return 'Not started'
    case 'locked':
      return 'Locked'
    default:
      return p.met ? 'Mastered' : 'Not started'
  }
}

export default function PrerequisitePanel({
  topicTitle,
  prerequisites,
  heading,
  intro,
  returnStack = [],
  onDismiss,
}: {
  topicTitle?: string
  prerequisites: EligibilityPrereq[]
  heading?: string
  intro?: string
  returnStack?: ReturnEntry[]
  onDismiss?: () => void
}) {
  const title = heading ?? `Prerequisites (${prerequisites.length})`
  // The default copy is the "not ready yet" case: the panel is shown when a target is locked, or
  // as an advisory on an eligible one (which passes its own `intro`). Either way it must say that
  // the learner can *study* a prerequisite they cannot yet practise.
  const body = intro ?? "You're not ready to practise this topic yet. Study the prerequisites below, then return to check your readiness."
  return (
    <section
      aria-label="Prerequisites"
      role="note"
      className="mt-4 border border-mathua-border bg-mathua-surface p-4 min-w-0"
    >
      <h3 className="font-mono text-xs uppercase tracking-wider text-mathua-primary">
        {title}
      </h3>
      {topicTitle && (
        <p className="mt-1 font-mono text-[11px] text-mathua-muted break-words [overflow-wrap:anywhere]">
          {topicTitle}
        </p>
      )}
      {body && (
        <p className="mt-2 font-mono text-[11px] text-mathua-secondary break-words [overflow-wrap:anywhere]">
          {body}
        </p>
      )}
      <ul className="mt-3 space-y-2">
        {prerequisites.map((p) => (
          <li key={p.conceptId} className="flex flex-col sm:flex-row sm:items-center gap-2 min-w-0">
            <span className="min-w-0 flex-1">
              <span className="block font-mono text-[11px] text-mathua-primary break-words [overflow-wrap:anywhere]">
                {p.title}
              </span>
              <span className="block font-mono text-[10px] uppercase tracking-wider text-mathua-muted">
                {prereqStateWord(p)} · Required prerequisite
              </span>
            </span>
            <Link
              href={prereqHref(p, returnStack)}
              className="w-full sm:w-auto shrink-0 border border-mathua-blue text-mathua-blue hover:bg-mathua-blue-faint px-4 py-2 font-mono text-[11px] min-h-[36px] inline-flex items-center justify-center text-center whitespace-nowrap"
            >
              {prereqCta(p)}
            </Link>
          </li>
        ))}
      </ul>
      {onDismiss && (
        <button
          type="button"
          onClick={onDismiss}
          className="mt-3 font-mono text-[10px] uppercase tracking-wider text-mathua-muted hover:text-mathua-blue"
        >
          Close
        </button>
      )}
    </section>
  )
}
