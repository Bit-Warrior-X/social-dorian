export class AuthError extends Error {
  constructor(message) {
    super(message)
    this.name = 'AuthError'
  }
}

export async function request(path, options = {}) {
  const headers = { ...(options.headers || {}) }
  if (options.body && !headers['Content-Type']) {
    headers['Content-Type'] = 'application/json'
  }

  const res = await fetch(path, {
    credentials: 'include',
    ...options,
    headers,
  })

  if (res.status === 204) {
    return null
  }

  const data = await res.json().catch(() => ({}))
  if (res.status === 401 && !String(path).startsWith('/api/auth/')) {
    const { clearCurrentUser } = await import('./auth')
    clearCurrentUser()
    if (typeof window !== 'undefined' && window.location.pathname !== '/') {
      const next = encodeURIComponent(window.location.pathname + window.location.search)
      window.location.assign(`/?next=${next}`)
    }
    throw new AuthError(data.error || 'sign in required')
  }
  if (!res.ok) {
    throw new Error(data.error || 'Request failed')
  }
  return data
}
