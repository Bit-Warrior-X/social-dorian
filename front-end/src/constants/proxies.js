export const PROXY_PROTOCOLS = [
  { value: 'http', label: 'HTTP', icon: 'ti-world', description: 'Standard web proxy' },
  { value: 'https', label: 'HTTPS', icon: 'ti-lock', description: 'TLS-secured proxy' },
  { value: 'socks5', label: 'SOCKS5', icon: 'ti-route', description: 'TCP tunnel proxy' },
]

export const PROXY_STATUSES = [
  { value: 'active', label: 'Active', description: 'Available for accounts' },
  { value: 'inactive', label: 'Inactive', description: 'Temporarily unavailable' },
  { value: 'error', label: 'Error', description: 'Needs attention' },
]

export function protocolLabel(value) {
  return PROXY_PROTOCOLS.find((p) => p.value === value)?.label || value
}

export function proxyStatusLabel(value) {
  return PROXY_STATUSES.find((s) => s.value === value)?.label || value
}

export function proxyAddress(proxy) {
  if (!proxy) return ''
  return `${proxy.protocol}://${proxy.host}:${proxy.port}`
}
