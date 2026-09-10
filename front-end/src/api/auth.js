import { ref } from 'vue'
import { request } from './http'

export const currentUser = ref(null)

let resolved = false

export function clearCurrentUser() {
  currentUser.value = null
  resolved = true
}

export async function getCurrentUser(force = false) {
  if (!force && resolved) {
    return currentUser.value
  }
  try {
    currentUser.value = await request('/api/auth/me')
  } catch {
    currentUser.value = null
  }
  resolved = true
  return currentUser.value
}

export async function login(email, password) {
  const user = await request('/api/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  })
  currentUser.value = user
  resolved = true
  return user
}

export async function logout() {
  try {
    await request('/api/auth/logout', { method: 'POST' })
  } finally {
    clearCurrentUser()
  }
}

export function safeNextPath(value) {
  const next = String(value || '').trim()
  if (!next.startsWith('/') || next.startsWith('//') || next === '/' || next.startsWith('/login')) {
    return '/dashboard'
  }
  return next
}
