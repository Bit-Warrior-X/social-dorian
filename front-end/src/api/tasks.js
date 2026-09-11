import { request } from './http'

const BASE = '/api/tasks'

export function listTasks(status = '') {
  const query = status ? `?status=${encodeURIComponent(status)}` : ''
  return request(`${BASE}${query}`)
}

export function getTask(id) {
  return request(`${BASE}/${id}`)
}

export function createTask(payload) {
  return request(BASE, {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export function cancelTask(id) {
  return request(`${BASE}/${id}/cancel`, { method: 'POST' })
}

export function listTaskLogs(id) {
  return request(`${BASE}/${id}/logs`)
}

export function getCredits() {
  return request('/api/credits')
}

export function getBusyAccounts() {
  return request('/api/busy-accounts')
}

export function getMonitorFeed({ after = 0, limit = 80 } = {}) {
  const params = new URLSearchParams()
  if (after) params.set('after', String(after))
  if (limit) params.set('limit', String(limit))
  const query = params.toString()
  return request(`/api/monitor/feed${query ? `?${query}` : ''}`)
}
