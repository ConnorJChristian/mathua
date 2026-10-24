// Fork change: course due dates ("2026-12-06") shown as "Dec 6".
export function formatDue(iso: string): string {
  const d = new Date(`${iso}T12:00:00`)
  if (isNaN(d.getTime())) return iso
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
}
