import { MATH_DOMAIN_ORDER } from './graphDomains'

/**
 * The two corpus facts the marketing copy states, in one place.
 *
 * `/how-it-works` said "1250 concepts spanning 17 domains" while the corpus had
 * grown to 21 domains, and nothing failed: a number in JSX cannot. The domain
 * count is derived here rather than written down, and the concept count is
 * written once, in the module the copy imports, so the next corpus change edits
 * one constant instead of hunting a sentence.
 *
 * `test/curriculumFacts.test.ts` re-derives both from `data/concepts` and fails
 * if they disagree, which is the half that makes this more than tidier markup.
 */
// The copy describes the mathematics curriculum; the Python track is a separate
// side of the site and is not counted here.
export const DOMAIN_COUNT = MATH_DOMAIN_ORDER.length

/** Sum of `data/concepts/*.json` for the domains in `MATH_DOMAIN_ORDER`. */
export const CONCEPT_COUNT = 1250
