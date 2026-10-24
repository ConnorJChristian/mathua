'use client'

import { useState, useEffect } from 'react'
import { isLoggedIn, getUserInfo, fetchMe, type UserInfo } from '../lib/auth'

// The first render is always logged-out, and the stored session is read after
// mount. The pages are a static export, so the HTML every visitor receives was
// rendered with no session; reading localStorage during the first client render
// produced a different header and React threw away the server markup (errors
// #418/#423 on every page, and a flash of the logged-out header besides).
// `ready` says the stored session has been read, for callers that must not act
// on the logged-out first render (redirects).
export function useAuthState() {
  const [loggedIn, setLoggedIn] = useState(false)
  const [user, setUser] = useState<UserInfo | null>(null)
  const [ready, setReady] = useState(false)

  useEffect(() => {
    const sync = () => {
      setLoggedIn(isLoggedIn())
      setUser(getUserInfo())
      setReady(true)
    }
    sync()
    window.addEventListener('auth-changed', sync)
    window.addEventListener('storage', sync)
    return () => {
      window.removeEventListener('auth-changed', sync)
      window.removeEventListener('storage', sync)
    }
  }, [])

  // The role is re-read from the server, once per page load.
  //
  // `mathua_user` is a cache of `/api/auth/me`, and a cache of a privilege is only as fresh as
  // its last write: a phone that signed in before Admin V2 carries no role, and one that signed
  // in after it had `moderator`/`owner` dropped by a comparison that listed only two of the four
  // roles. Either way `isStaffRole(user.role)` answered false for good, so the Admin entry never
  // appeared on that device while a re-authenticated desktop showed it.
  //
  // `fetchMe` repairs the stored session and broadcasts `auth-changed`, so this resolves through
  // the same listener the sign-in path uses rather than being a parallel source of truth. It runs
  // only when a role is missing — a learner is fetched once and then left alone, because
  // "the server said not staff" is worth caching and "we could not ask" is not.
  useEffect(() => {
    if (typeof window === 'undefined' || !isLoggedIn()) return
    if (getUserInfo()?.role) return
    let cancelled = false
    fetchMe().then((me) => {
      if (!cancelled && me?.role) setUser(me)
    })
    return () => { cancelled = true }
  }, [])

  return { loggedIn, user, ready }
}