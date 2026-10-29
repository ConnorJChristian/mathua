'use client'

import { Suspense, useEffect, useMemo, useState } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import Link from 'next/link'
import Header from '../../components/Header'
import BottomTabs from '../../components/BottomTabs'
import Footer from '../../components/Footer'
import ConceptReference from '../../components/ConceptReference'
import RelationList, { type RelationItem } from '../../components/RelationList'
import { prereqStateWord } from '../../components/PrerequisitePanel'
import { getLessonReadiness, getProgress, type ConceptProgress, type ReadinessRes } from '../../lib/api'
import { getUserInfo } from '../../lib/auth'
import { concepts } from '../../lib/conceptData'
import { domainLabel } from '../../lib/domains'
import { masteryState } from '../../lib/progress'
import { backTarget, courseReturnLabel, learnConceptPath, parseReturnStack, pushReturn, studyConceptPath, withReturnStack } from '../../lib/learnReturn'
import { hrefConceptId } from '../../lib/nextUp'

// The reference and exploration surface.
//
// ADR-047 closed `/study` as a learner workflow because it was a *catalogue* — a browsable
// library of whole lessons with its own search and domain index, linked from every header, tab
// bar and sidebar. That let a learner read their way to a concept instead of being taught it.
//
// ADR-054 restores it, scoped: this page renders the reference for **one** concept named in the
// URL, in a reading column with its direct DAG neighbourhood beside it. It is not a catalogue —
// no domain index, no lesson list, no search — and it awards nothing. Discovery stays structured
// in `/domains`; practice stays gated in `/learn`. The only onward navigation is the concept's
// own prerequisites and dependents, plus a separate *Practise this concept* control.
//
// `Locked` never means "cannot read": every neighbour row is a link, whatever its state.
//
// Without `?concept=` there is nothing to show, so it forwards to `/learn` rather than presenting
// an index. Static export cannot use next.config redirects, so this is a client forward, the
// same pattern as the deprecated stubs.
function ConceptNotFound() {
  const router = useRouter()
  useEffect(() => { router.replace('/learn') }, [router])
  return null
}

function stateLabel(state: string | undefined): string {
  switch (state) {
    case 'mastered':
      return 'Mastered'
    case 'due_for_review':
      return 'Due a check'
    case 'in_progress':
      return 'In progress'
    case 'unlocked':
    case 'upcoming':
      return 'Not started'
    case 'locked':
      return 'Locked'
    default:
      return ''
  }
}

/** A dependent's state from the learner's own history — the only source the client has for it. */
function dependentStateWord(p: ConceptProgress | undefined): string {
  switch (masteryState(p)) {
    case 'mastered':
      return 'Mastered'
    case 'decaying':
      return 'Due a check'
    case 'practicing':
    case 'learning':
      return 'In progress'
    default:
      return 'Not started'
  }
}

function StudyContent() {
  const searchParams = useSearchParams()
  const conceptId = searchParams.get('concept') || ''
  const returnStack = useMemo(() => parseReturnStack(searchParams), [searchParams])

  const [readiness, setReadiness] = useState<ReadinessRes | null>(null)
  const [progress, setProgress] = useState<Record<string, ConceptProgress>>({})
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    if (!conceptId) return
    let cancelled = false
    setReadiness(null)
    setFailed(false)
    getLessonReadiness(conceptId)
      .then(res => { if (!cancelled) setReadiness(res) })
      .catch(() => { if (!cancelled) setFailed(true) })
    const sid = getUserInfo()?.student_id
    if (sid) {
      getProgress(sid)
        .then(p => { if (!cancelled) setProgress(p) })
        .catch(() => {})
    }
    return () => { cancelled = true }
  }, [conceptId])

  if (!conceptId) return <ConceptNotFound />

  const concept = concepts.find(c => c.id === conceptId)
  const label = concept?.label || conceptId
  const domain = concept?.domain
  const state = stateLabel(readiness?.state)
  const prereqItems: RelationItem[] = (readiness?.prerequisites ?? []).map(p => ({
    id: p.conceptId,
    title: p.title,
    state: prereqStateWord(p),
  }))
  // Dependents are presentation over the corpus the app already bundles, the same way the Learn
  // done card derives "what this unlocks". It is a display of the DAG, never a second rule.
  const dependents = concepts.filter(c => (c.prerequisites ?? []).includes(conceptId))
  const dependentItems: RelationItem[] = dependents.map(c => ({
    id: c.id,
    title: c.label,
    state: dependentStateWord(progress[c.id]),
  }))

  const back = backTarget(returnStack)
  const backLabel = back ? (courseReturnLabel(back.href) ?? concepts.find(c => c.id === hrefConceptId(back.href))?.label ?? '') : ''
  const mastered = readiness?.state === 'mastered' || readiness?.state === 'due_for_review'
  // A mastered concept is owed retrieval, not another tutorial (ADR-037); everything else enters
  // the one assessed loop, which enforces its own gate if the topic is locked.
  const practiseHref = withReturnStack(mastered ? '/review' : learnConceptPath(conceptId), returnStack)
  const neighbourStack = pushReturn(returnStack, studyConceptPath(conceptId))

  const emptyActions = (
    <>
      {prereqItems.length > 0 && (
        <Link
          href="#depends-on"
          className="border border-mathua-border px-4 py-2 font-mono text-[11px] text-mathua-secondary hover:border-mathua-blue hover:text-mathua-blue min-h-[36px] inline-flex items-center"
        >
          Explore prerequisites
        </Link>
      )}
      <Link
        href={practiseHref}
        className="border border-mathua-blue text-mathua-blue hover:bg-mathua-blue-faint px-4 py-2 font-mono text-[11px] min-h-[36px] inline-flex items-center"
      >
        Practise this concept →
      </Link>
    </>
  )

  return (
    <div className="mt-6 mb-16 px-4">
      <div className="max-w-4xl mx-auto">
        <div className="mb-4">
          {back ? (
            <Link href={back.href} className="font-mono text-xs text-mathua-secondary hover:text-mathua-blue">
              ← Back to {backLabel || 'previous topic'}
            </Link>
          ) : (
            <Link href="/profile" className="font-mono text-xs text-mathua-secondary hover:text-mathua-blue">
              ← Profile
            </Link>
          )}
        </div>

        <h1 className="font-mono text-base text-mathua-primary break-words [overflow-wrap:anywhere]">
          {label}
        </h1>
        <p className="mt-1 font-mono text-[10px] uppercase tracking-wider text-mathua-muted">
          Reference{domain ? ` · ${domainLabel(domain)}` : ''}{state ? ` · ${state}` : ''}
        </p>

        {failed && (
          <p role="alert" className="mt-3 font-mono text-xs text-mathua-red">
            Couldn&apos;t refresh this concept&apos;s status. You can still read the material
            below.
          </p>
        )}

        <div className="mt-6 grid gap-6 lg:grid-cols-[minmax(0,1fr)_240px]">
          <div className="min-w-0">
            <ConceptReference conceptId={conceptId} variant="page" emptyActions={emptyActions} />
          </div>
          <aside className="min-w-0 space-y-6">
            <RelationList
              id="depends-on"
              heading="Depends on"
              items={prereqItems}
              emptyText="No prerequisites listed"
              loading={readiness === null && !failed}
              returnStack={neighbourStack}
            />
            <RelationList
              heading="Unlocks"
              items={dependentItems}
              emptyText="No dependent concepts listed"
              returnStack={neighbourStack}
            />
          </aside>
        </div>

        <div className="mt-8 border-t border-mathua-border pt-5">
          <p className="font-mono text-xs text-mathua-primary">Ready to practise?</p>
          <p className="mt-1 font-mono text-[11px] text-mathua-secondary">
            Work toward mastery in the assessed learning flow. Reading here does not count as
            progress.
          </p>
          <div className="mt-3 flex flex-wrap gap-3">
            <Link
              href={practiseHref}
              className="border border-mathua-blue text-mathua-blue hover:bg-mathua-blue-faint px-5 py-2 font-mono text-xs min-h-[40px] inline-flex items-center"
            >
              Practise this concept →
            </Link>
            {back && (
              <Link
                href={back.href}
                className="border border-mathua-border px-5 py-2 font-mono text-xs text-mathua-secondary hover:border-mathua-blue hover:text-mathua-blue min-h-[40px] inline-flex items-center"
              >
                ← Back to {backLabel || 'previous topic'}
              </Link>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

export default function StudyPage() {
  return (
    <>
      <Header />
      <div className="pt-[var(--chrome-top)] lg:pt-0">
        <Suspense fallback={<div className="p-6 font-mono text-xs text-mathua-muted">Loading…</div>}>
          <StudyContent />
        </Suspense>
      </div>
      <BottomTabs />
      <Footer />
    </>
  )
}
