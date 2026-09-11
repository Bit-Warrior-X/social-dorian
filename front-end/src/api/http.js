export class AuthError extends Error {
  constructor(message) {
    super(message)
    this.name = 'AuthError'
  }
}

export async function request(path, options = {}) {
  const headers = { ...(options.headers || {}) }
  const isForm = typeof FormData !== 'undefined' && options.body instanceof FormData
  if (options.body && !headers['Content-Type'] && !isForm) {
    headers['Content-Type'] = 'application/json'
  }
  if (isForm && headers['Content-Type']) {
    delete headers['Content-Type']
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
