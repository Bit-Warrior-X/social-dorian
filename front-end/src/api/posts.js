import { request } from './http'

export function listTrackedPosts(status = '') {
  const query = status ? `?status=${encodeURIComponent(status)}` : ''
  return request(`/api/posts${query}`)
}

export function getTrackedPost(id) {
  return request(`/api/posts/${id}`)
}

export function createTrackedPost(payload) {
  return request('/api/posts', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export function updateTrackedPost(id, payload) {
  return request(`/api/posts/${id}`, {
    method: 'PUT',
    body: JSON.stringify(payload),
  })
}

export function deleteTrackedPost(id) {
  return request(`/api/posts/${id}`, { method: 'DELETE' })
}

export function analyzeTrackedPost(id) {
  return request(`/api/posts/${id}/analyze`, { method: 'POST' })
}

export function suggestReplies(payload) {
  return request('/api/posts/suggest', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export function listEngagements({ accountId = 0, kind = '', limit = 150 } = {}) {
  const params = new URLSearchParams()
  if (accountId) params.set('accountId', String(accountId))
  if (kind) params.set('kind', kind)
  if (limit) params.set('limit', String(limit))
  const query = params.toString()
  return request(`/api/engagements${query ? `?${query}` : ''}`)
}
