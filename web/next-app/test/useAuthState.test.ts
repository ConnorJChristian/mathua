import { describe, it, expect, beforeEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useAuthState } from '../hooks/useAuthState'

describe('useAuthState', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  // Q16: the static export renders the header logged-out, so the first client
  // render must match it even when a session is stored, or React discards the
  // server markup (#418/#423).
  it('renders logged-out first, then reads the stored session', () => {
    localStorage.setItem('mathua_token', 'tok')
    localStorage.setItem('mathua_user', JSON.stringify({ student_id: 's1', name: 'A', username: 'a', concepts_mastered: 0, current_streak: 0, level: 'l', diagnostic_completed: false }))
    const renders: { loggedIn: boolean; ready: boolean }[] = []
    const { result } = renderHook(() => {
      const s = useAuthState()
      renders.push({ loggedIn: s.loggedIn, ready: s.ready })
      return s
    })
    expect(renders[0]).toEqual({ loggedIn: false, ready: false })
    expect(result.current.loggedIn).toBe(true)
    expect(result.current.ready).toBe(true)
    expect(result.current.user?.student_id).toBe('s1')
  })

  it('reports logged-out by default', () => {
    const { result } = renderHook(() => useAuthState())
    expect(result.current.loggedIn).toBe(false)
    expect(result.current.user).toBe(null)
  })

  it('becomes reactive when auth-changed fires after login', async () => {
    const { result } = renderHook(() => useAuthState())
    act(() => {
      localStorage.setItem('mathua_token', 'tok')
      localStorage.setItem('mathua_user', JSON.stringify({ student_id: 's1', name: 'A', username: 'a', concepts_mastered: 0, current_streak: 0, level: 'l', diagnostic_completed: false }))
      window.dispatchEvent(new Event('auth-changed'))
    })
    expect(result.current.loggedIn).toBe(true)
    expect(result.current.user?.student_id).toBe('s1')
  })

  it('returns to logged-out when auth-changed fires after logout', async () => {
    localStorage.setItem('mathua_token', 'tok')
    const { result } = renderHook(() => useAuthState())
    act(() => {
      localStorage.removeItem('mathua_token')
      localStorage.removeItem('mathua_user')
      window.dispatchEvent(new Event('auth-changed'))
    })
    expect(result.current.loggedIn).toBe(false)
    expect(result.current.user).toBe(null)
  })
})
