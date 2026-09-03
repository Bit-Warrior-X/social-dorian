export const PLATFORMS = [
  { value: 'twitter', label: 'Twitter / X', icon: 'ti-brand-x' },
  { value: 'instagram', label: 'Instagram', icon: 'ti-brand-instagram' },
  { value: 'linkedin', label: 'LinkedIn', icon: 'ti-brand-linkedin' },
  { value: 'facebook', label: 'Facebook', icon: 'ti-brand-facebook' },
  { value: 'tiktok', label: 'TikTok', icon: 'ti-brand-tiktok' },
  { value: 'youtube', label: 'YouTube', icon: 'ti-brand-youtube' },
]

export const GENDERS = [
  { value: 'male', label: 'Male' },
  { value: 'female', label: 'Female' },
  { value: 'other', label: 'Other' },
  { value: 'prefer_not_to_say', label: 'Prefer not to say' },
]

export const STATUSES = [
  { value: 'active', label: 'Active' },
  { value: 'expired', label: 'Expired' },
  { value: 'error', label: 'Error' },
  { value: 'revoked', label: 'Revoked' },
]

export const PROXY_MODES = [
  { value: 'none', label: 'None' },
  { value: 'auto', label: 'Auto select', description: 'Select proxy automatically' },
  { value: 'manual', label: 'Manual', description: 'Select a registered proxy' },
]

export function platformMeta(value) {
  return PLATFORMS.find((p) => p.value === value) || { value, label: value, icon: 'ti-share' }
}

export function genderLabel(value) {
  return GENDERS.find((g) => g.value === value)?.label || value
}

export function statusLabel(value) {
  return STATUSES.find((s) => s.value === value)?.label || value
}

export function proxyModeLabel(value) {
  return PROXY_MODES.find((m) => m.value === value)?.label || value || 'None'
}

export function accountDisplayName(account) {
  if (!account) return 'this account'
  const name = [account.firstName, account.lastName].filter(Boolean).join(' ').trim()
  return name || account.email || 'this account'
}

export function accountProxyDetails(account, proxies = []) {
  if (!account || account.proxyMode === 'none') {
    return { mode: 'None', host: '', country: '', label: 'None' }
  }

  const proxy = proxies.find((p) => String(p.id) === String(account.proxyId))
  const mode = account.proxyMode === 'auto' ? 'Auto' : 'Manual'
  const host = proxy
    ? `${proxy.host}:${proxy.port}`
    : account.proxyId
      ? `Proxy #${account.proxyId}`
      : ''
  const country = proxy?.country || ''

  if (!host) {
    return {
      mode,
      host: '',
      country: '',
      label: account.proxyMode === 'auto' ? 'Auto select' : 'Manual',
    }
  }

  const label = country ? `${mode} · ${host} · ${country}` : `${mode} · ${host}`
  return { mode, host, country, label }
}

export function accountProxyLabel(account, proxies = []) {
  return accountProxyDetails(account, proxies).label
}
