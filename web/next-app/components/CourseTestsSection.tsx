import Link from 'next/link'
import type { CourseView } from '../lib/api'

// Fork change: the course's tests. Each button shows only when the course can
// offer that test (CourseView.tests); the server's viability rule is in
// internal/engine/course_tests.go.
export default function CourseTestsSection({ view }: { view: CourseView }) {
  const t = view.tests
  if (!t) return null
  const q = (n: number) => `${n} question${n === 1 ? '' : 's'}`
  const btn = 'border px-4 min-h-[36px] inline-flex items-center font-mono text-xs'
  const id = encodeURIComponent(view.id)
  return (
    <section aria-labelledby="course-tests" className="border border-mathua-border bg-mathua-surface p-4">
      <h2 id="course-tests" className="font-mono text-[10px] uppercase tracking-wider text-mathua-muted">Tests</h2>
      <ul className="mt-2 space-y-3">
        {t.placement && (
          <li className="min-w-0">
            <Link href={`/onboard?course=${id}`} className={`${btn} border-mathua-blue text-mathua-blue hover:bg-mathua-blue-faint`}>Placement test →</Link>
            <p className="font-mono text-[11px] text-mathua-secondary mt-1">
              {t.placement_questions > 0
                ? `Up to ${q(t.placement_questions)} on this course and what it builds on. Topics you show you know open up, so you can skip ahead. Progress you already have is never lowered.`
                : 'You already have progress on every topic it could ask about.'}
            </p>
          </li>
        )}
        {t.review && (
          <li className="min-w-0">
            {t.review_topics > 0 ? (
              <span className="flex flex-wrap gap-2">
                <Link href={`/goals?quiz=1&course=${id}&scope=review`} className={`${btn} border-mathua-blue text-mathua-blue hover:bg-mathua-blue-faint`}>Review test →</Link>
                {t.full_review_questions > t.review_questions && (
                  <Link href={`/goals?quiz=1&course=${id}&scope=full`} className={`${btn} border-mathua-border text-mathua-secondary hover:border-mathua-blue hover:text-mathua-blue`}>Full course review</Link>
                )}
              </span>
            ) : (
              <span className={`${btn} border-mathua-border text-mathua-muted`} aria-disabled="true">Review test</span>
            )}
            <p className="font-mono text-[11px] text-mathua-secondary mt-1">
              {t.review_topics > 0
                ? `${q(t.review_questions)} on topics this course has taught you, weakest first, across its sections. Counts as mastery evidence; misses go back into review.${t.full_review_questions > t.review_questions ? ` The full review asks ${t.full_review_questions}.` : ''}`
                : 'Covers topics this course has taught you. Study a topic first.'}
            </p>
          </li>
        )}
      </ul>
    </section>
  )
}
