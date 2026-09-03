const BASE = '/api/proxies'

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

export function listProxies() {
  return request(BASE)
}

export function createProxy(payload) {
  return request(BASE, {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export function updateProxy(id, payload) {
  return request(`${BASE}/${id}`, {
    method: 'PUT',
    body: JSON.stringify(payload),
  })
}

export function deleteProxy(id) {
  return request(`${BASE}/${id}`, { method: 'DELETE' })
}

export function checkProxy(id) {
  return request(`${BASE}/${id}/check`, { method: 'POST' })
}

export function checkAllProxies() {
  return request(`${BASE}/check-all`, { method: 'POST' })
}

export function lookupCountryFromHost(host) {
  return request('/api/geo/country', {
    method: 'POST',
    body: JSON.stringify({ host }),
  })
}
