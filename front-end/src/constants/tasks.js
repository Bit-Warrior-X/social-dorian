export const TASK_TYPES = [
  {
    value: 'report',
    label: 'Report post',
    icon: 'ti-flag',
    description: 'Report a post URL with selected accounts',
    needsUrl: true,
  },
  {
    value: 'post',
    label: 'New post',
    icon: 'ti-pencil-plus',
    description: 'Publish or prepare a post for selected accounts',
    needsUrl: true,
  },
  {
    value: 'browse',
    label: 'Browse feed',
    icon: 'ti-player-play',
    description: 'Simulate human browsing activity',
    needsUrl: false,
  },
  {
    value: 'login_test',
    label: 'Login test',
    icon: 'ti-shield-check',
    description: 'Verify saved session / credentials',
    needsUrl: false,
  },
]

export const TASK_STATUSES = [
  { value: 'queued', label: 'Queued' },
  { value: 'running', label: 'Running' },
  { value: 'completed', label: 'Completed' },
  { value: 'failed', label: 'Failed' },
  { value: 'cancelled', label: 'Cancelled' },
]

export function taskTypeMeta(value) {
  return (
    TASK_TYPES.find((item) => item.value === value) || {
      value,
      label: value,
      icon: 'ti-list-check',
      description: '',
      needsUrl: false,
    }
  )
}

export function taskStatusLabel(value) {
  return TASK_STATUSES.find((item) => item.value === value)?.label || value
}

export function taskProgress(task) {
  const total = Number(task?.accountCount || 0)
  const done = Number(task?.doneCount || 0)
  if (!total) return 0
  return Math.round((done / total) * 100)
}
