// The Study banner's XP line for a due mastery check. Before the first check
// (or a course test standing in for one) there is no "last one" to count
// from, so it says "so far" (Q59). An older server that does not send
// quiz_taken keeps the old wording.
export function quizXPLine(since: number, taken: boolean | undefined): string {
  return `${since} XP ${taken === false ? 'so far' : 'since the last one'}`
}
