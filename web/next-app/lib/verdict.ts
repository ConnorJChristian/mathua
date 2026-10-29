// The words under a test's verdict line ("✓ Correct!" / "✗ Not quite").
//
// The tests used to fall back to "Correct!" or "Not quite." when the server
// sent no feedback, which printed the verdict twice: once as the verdict and
// once as its "feedback". The server sends only a grader's bare status token
// on an assessment ("Not equivalent", "Incorrect order"), or nothing; that is
// all that belongs under the verdict, and a token that only restates it is
// dropped.
const RESTATES_VERDICT = new Set(['correct', 'correct!', 'incorrect', 'incorrect.', 'not quite', 'not quite.', 'wrong'])

export function verdictDetail(feedback?: string | null): string {
  const f = (feedback ?? '').trim()
  return RESTATES_VERDICT.has(f.toLowerCase()) ? '' : f
}
