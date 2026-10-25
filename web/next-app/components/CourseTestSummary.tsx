import Link from 'next/link'
import type { CourseTestSection } from '../lib/api'

// Fork change: what a course test found, by course section. Placement says
// what it placed (answered right, or below a right answer) and where to
// start; review says what held up and what was queued for review. Topics
// link to Learn, the one surface that teaches.

type Mode = 'placement' | 'review'

const WORDS: Record<Mode, { passed: string; missed: string; inferred: string }> = {
  placement: { passed: 'Placed', missed: 'Start here', inferred: 'Placed below a right answer' },
  review: { passed: 'Held up', missed: 'Needs review', inferred: '' },
}

function TopicList({ label, topics, courseId, tone }: { label: string; topics: { id: string; label: string }[]; courseId?: string; tone: string }) {
  if (topics.length === 0) return null
  return (
    <div className="mt-2 min-w-0">
      <p className={`font-mono text-[10px] uppercase tracking-wider ${tone}`}>{label} ({topics.length})</p>
      <ul className="mt-1 flex flex-wrap gap-x-3 gap-y-1">
        {topics.map((t) => (
          <li key={t.id} className="min-w-0">
            <Link
              href={`/learn?concept=${encodeURIComponent(t.id)}${courseId ? `&course=${encodeURIComponent(courseId)}` : ''}`}
              className="font-mono text-[11px] text-mathua-primary hover:text-mathua-blue break-words [overflow-wrap:anywhere]"
            >
              {t.label}
            </Link>
          </li>
        ))}
      </ul>
    </div>
  )
}

export default function CourseTestSummary({ sections, mode, courseId }: { sections: CourseTestSection[]; mode: Mode; courseId?: string }) {
  if (sections.length === 0) return null
  const w = WORDS[mode]
  return (
    <section aria-label="Results by section" className="border border-mathua-border bg-mathua-surface text-left">
      <p className="px-4 pt-3 font-mono text-[10px] uppercase tracking-wider text-mathua-muted">By section</p>
      <ol className="divide-y divide-mathua-border">
        {sections.map((s, i) => (
          <li key={`${s.title}-${i}`} className="px-4 py-3 min-w-0">
            <p className="font-mono text-xs text-mathua-primary break-words [overflow-wrap:anywhere]">
              {s.title}
              <span className="text-mathua-muted"> · {s.passed.length + (s.inferred?.length ?? 0)} {mode === 'placement' ? 'placed' : 'held'}, {s.missed.length} {mode === 'placement' ? 'to start' : 'to review'}</span>
            </p>
            <TopicList label={w.passed} topics={s.passed} courseId={courseId} tone="text-mathua-green" />
            {mode === 'placement' && <TopicList label={w.inferred} topics={s.inferred ?? []} courseId={courseId} tone="text-mathua-muted" />}
            <TopicList label={w.missed} topics={s.missed} courseId={courseId} tone="text-mathua-red" />
          </li>
        ))}
      </ol>
    </section>
  )
}
