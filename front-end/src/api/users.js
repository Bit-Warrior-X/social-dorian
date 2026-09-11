import { request } from './http'

export function listUsers() {
  return request('/api/users')
}

export function createUser(payload) {
  return request('/api/users', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export function updateUser(id, payload) {
  return request(`/api/users/${id}`, {
    method: 'PUT',
    body: JSON.stringify(payload),
  })
}

export function deleteUser(id) {
  return request(`/api/users/${id}`, { method: 'DELETE' })
}

export async function uploadMedia(file) {
  const body = new FormData()
  body.append('file', file)
  return request('/api/uploads', {
    method: 'POST',
    body,
  })
}
