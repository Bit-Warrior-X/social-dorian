import { request } from './http'

const BASE = '/api/dashboard'

export function getDashboard() {
  return request(BASE)
}
