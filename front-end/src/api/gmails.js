import { request } from './http'

const BASE = '/api/gmails'

export function listGmails() {
  return request(BASE)
}

export function createGmail(payload) {
  return request(BASE, {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export function updateGmail(id, payload) {
  return request(`${BASE}/${id}`, {
    method: 'PUT',
    body: JSON.stringify(payload),
  })
}

export function deleteGmail(id) {
  return request(`${BASE}/${id}`, { method: 'DELETE' })
}

export function getGmailPinCode(id) {
  return request(`${BASE}/${id}/pin-code`, { method: 'POST' })
}
