const BASE = '/api/accounts'

async function request(path, options = {}) {
  const res = await fetch(path, {
    headers: { 'Content-Type': 'application/json', ...(options.headers || {}) },
    ...options,
  })

  if (res.status === 204) {
    return null
  }

  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(data.error || 'Request failed')
  }
  return data
}

export function listAccounts() {
  return request(BASE)
}

export function createAccount(payload) {
  return request(BASE, {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export function updateAccount(id, payload) {
  return request(`${BASE}/${id}`, {
    method: 'PUT',
    body: JSON.stringify(payload),
  })
}

export function deleteAccount(id) {
  return request(`${BASE}/${id}`, { method: 'DELETE' })
}

export function openAccountSession(id) {
  return request(`${BASE}/${id}/open`, { method: 'POST' })
}

export function closeAccountSession(sessionUrl) {
  if (!sessionUrl) return Promise.resolve(null)
  const match = String(sessionUrl).match(/\/api\/sessions\/([^/]+)/)
  if (!match) return Promise.resolve(null)
  return request(`/api/sessions/${match[1]}`, { method: 'DELETE' })
}
