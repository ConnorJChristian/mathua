import { hrefConceptId } from './nextUp'

/**
 * The return stack behind prerequisite navigation.
 *
 * A learner follows a chain — target -> prerequisite -> prerequisite's prerequisite — and must
 * be able to walk back up it without losing where they came from. That path is real URL state,
 * not component state, because a navigation unmounts the component that held it: the stack
 * travels in repeated `?return=` parameters, most-recent first.
 *
 * Each entry is a *path* (`/learn?concept=X` or `/study?concept=X`), not a bare concept id,
 * because a prerequisite can be opened either as reference (`/study`) or as the learning loop
 * (`/learn`), and returning must land back on the surface the learner actually left. An older
 * link that stored a bare concept id is read as a learning-loop target, which is what it meant.
 *
 * The loop guard is the load-bearing part. Pushing the current page drops anything already
 * pointing at the same concept, so A -> B -> A collapses instead of growing without bound and
 * no back-link can ever name the page you are already on.
 */

/** A path a learner can return to, e.g. `/learn?concept=x` or `/study?concept=y`. */
export type ReturnEntry = string

export function learnConceptPath(conceptId: string): string {
  return `/learn?concept=${encodeURIComponent(conceptId)}`
}

export function studyConceptPath(conceptId: string): string {
  return `/study?concept=${encodeURIComponent(conceptId)}`
}

/** A stored entry is a path; a bare value is an old concept-id link to the learning loop. */
export function normalizeReturnEntry(raw: string): ReturnEntry | null {
  const v = raw.trim()
  if (!v) return null
  if (v.startsWith('/')) return v
  return learnConceptPath(v)
}

/**
 * Read the stack from a query. Malformed or repeated entries are dropped rather than trusted: a
 * return target is attacker-reachable (it is just a URL) and a back-link is a redirect, so it
 * must never be able to reach a route we would not build ourselves.
 */
export function parseReturnStack(params: Pick<URLSearchParams, 'getAll'>): ReturnEntry[] {
  const out: ReturnEntry[] = []
  const seen = new Set<string>()
  for (const raw of params.getAll('return')) {
    const entry = normalizeReturnEntry(raw)
    if (!entry || seen.has(entry)) continue
    seen.add(entry)
    out.push(entry)
  }
  return out
}

/**
 * Push the current page onto the stack. The current page is dropped from the tail by concept id,
 * so a cycle through the DAG cannot make the stack grow and a back-link cannot point at itself.
 */
export function pushReturn(stack: ReturnEntry[], currentPath: string): ReturnEntry[] {
  const currentId = hrefConceptId(currentPath)
  const kept = stack.filter(
    (entry) => entry !== currentPath && (currentId === null || hrefConceptId(entry) !== currentId),
  )
  return [currentPath, ...kept]
}

/**
 * The page to go back to, if any: the most recent return entry, carrying the rest of the stack so
 * the chain survives one more hop. Null when the stack is empty, which is a first-visit target.
 */
export function backTarget(stack: ReturnEntry[]): { href: string; remaining: ReturnEntry[] } | null {
  if (stack.length === 0) return null
  return { href: withReturnStack(stack[0], stack.slice(1)), remaining: stack.slice(1) }
}

/** Serialize a stack onto a path as repeated `return` parameters. */
export function withReturnStack(path: string, stack: ReturnEntry[]): string {
  if (stack.length === 0) return path
  const sep = path.includes('?') ? '&' : '?'
  return path + stack.map((entry) => `${sep}return=${encodeURIComponent(entry)}`).join('')
}

/**
 * A course page as a return target (fork change): a topic opened from a course's Modules or
 * Home tab returns there, not to the profile. The tab is kept so "back" lands where the
 * learner was.
 */
export function courseReturnPath(courseId: string, tab?: string): ReturnEntry {
  const t = tab && tab !== 'home' ? `&tab=${encodeURIComponent(tab)}` : ''
  return `/course?id=${encodeURIComponent(courseId)}${t}`
}

/** The words a back-link reads for a course return target, or null for any other target. */
export function courseReturnLabel(href: string): string | null {
  if (!href.startsWith('/course?')) return null
  const tab = new URLSearchParams(href.slice('/course?'.length)).get('tab')
  return tab === 'modules' ? 'course modules' : tab ? `the course (${tab})` : 'the course'
}

/**
 * Where the "home" exit of a practice or review screen goes (Q64): the course the learner came
 * from when the return stack holds one, the profile otherwise. A topic opened from a course
 * offered "Back to Profile" on its locked view and at the end of practice, beside "← Back to the
 * course", and review end screens went to the profile only.
 */
export function homeTarget(stack: ReturnEntry[]): { href: string; label: string } {
  const course = stack.find((e) => e.startsWith('/course?'))
  return course ? { href: course, label: 'Back to the course' } : { href: '/profile', label: 'Back to Profile' }
}
