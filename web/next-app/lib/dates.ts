// Fork change: course due dates ("2026-12-06") shown as "Dec 6".
export function formatDue(iso: string): string {
  const d = new Date(`${iso}T12:00:00`)
  if (isNaN(d.getTime())) return iso
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
}

/**
 * Topics a week needed to finish a course's work by its date, or null when
 * there is no date, no time left or nothing left. Counts only the topics due
 * by the date (`due_total`): a module marked after_due, such as material
 * taught after the final, is not work the date asks for. Falls back to the
 * whole plan for a server that does not send the due counts.
 */
export function topicsPerWeek(v: { total: number; mastered: number; due_total?: number; due_mastered?: number; days_left?: number }): number | null {
  if (v.days_left === undefined || v.days_left <= 0) return null
  const left = v.due_total !== undefined ? v.due_total - (v.due_mastered ?? 0) : v.total - v.mastered
  if (left <= 0) return null
  return (left / Math.max(v.days_left, 1)) * 7
}
