export const GMAIL_STATUSES = [
  { value: 'active', label: 'Active', description: 'Ready for account use' },
  { value: 'inactive', label: 'Inactive', description: 'Temporarily unavailable' },
  { value: 'error', label: 'Error', description: 'Needs attention' },
]

export function gmailStatusLabel(value) {
  return GMAIL_STATUSES.find((s) => s.value === value)?.label || value
}

export function gmailDisplayName(gmail) {
  if (!gmail) return 'this mailbox'
  return gmail.label || gmail.email || 'this mailbox'
}
