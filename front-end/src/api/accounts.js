import { request } from './http'

const BASE = '/api/accounts'

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

function sessionTokenFromUrl(sessionUrl) {
  const match = String(sessionUrl || '').match(/\/api\/sessions\/([^/]+)/)
  return match ? match[1] : ''
}

export function closeAccountSession(sessionUrl) {
  const token = sessionTokenFromUrl(sessionUrl)
  if (!token) return Promise.resolve(null)
  return request(`/api/sessions/${token}`, { method: 'DELETE' })
}

export function getAccountSessionStatus(sessionUrl) {
  const token = sessionTokenFromUrl(sessionUrl)
  if (!token) return Promise.resolve(null)
  return request(`/api/sessions/${token}/status`)
}

export function retryAccountLogin(sessionUrl) {
  const token = sessionTokenFromUrl(sessionUrl)
  if (!token) return Promise.resolve(null)
  return request(`/api/sessions/${token}/login`, { method: 'POST' })
}
