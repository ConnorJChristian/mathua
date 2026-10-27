/**
 * Move the document to its top after an in-page page change.
 *
 * Router navigation already has `ScrollToTopOnNavigate`, but a page that
 * paginates with a query string or local state never changes the pathname, so
 * that guard does not fire. Pagination controls call this explicitly.
 */
export function scrollToPageTop(): void {
  if (typeof window === 'undefined') return
  window.scrollTo({ top: 0, left: 0, behavior: 'auto' })
}
