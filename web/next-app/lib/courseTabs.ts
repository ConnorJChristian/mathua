import type { CourseView } from './api'

export type CourseTab = 'home' | 'modules' | 'readings' | 'videos' | 'grades'

const ALL: { id: CourseTab; label: string }[] = [
  { id: 'home', label: 'Home' },
  { id: 'modules', label: 'Modules' },
  { id: 'readings', label: 'Readings' },
  { id: 'videos', label: 'Lectures' },
  { id: 'grades', label: 'Grades' },
]

/** Whether a course has anything for its Lectures tab: lecture notes or videos. */
export function hasLectures(view: Pick<CourseView, 'notes' | 'videos' | 'modules'>): boolean {
  if (view.notes.length > 0 || view.videos.length > 0) return true
  return view.modules.some(m => m.topics.some(t => (t.videos?.length ?? 0) > 0))
}

/**
 * The course page's tabs. Lectures is left out for a course with no lecture
 * notes and no videos: its empty state used to be developer text about
 * videos.json and FORK.md.
 */
export function courseTabs(view: Pick<CourseView, 'notes' | 'videos' | 'modules'>): { id: CourseTab; label: string }[] {
  return hasLectures(view) ? ALL : ALL.filter(t => t.id !== 'videos')
}

/** What the Lectures tab says when a course has no lecture videos. */
export const NO_LECTURE_VIDEOS = 'No lecture videos for this course.'

/**
 * The course page's line for written tasks it cannot grade: proofs and writing
 * problems are left out of practice without an API key, and the page used to
 * say nothing about them. Empty when there are none.
 */
export function ungradedTasksLine(n?: number): string {
  if (!n || n <= 0) return ''
  return n === 1 ? '1 writing/proof task needs an API key to be graded.' : `${n} writing/proof tasks need an API key to be graded.`
}
