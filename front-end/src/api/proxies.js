import { request } from './http'

const BASE = '/api/proxies'

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
