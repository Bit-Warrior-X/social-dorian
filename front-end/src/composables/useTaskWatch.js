const PENDING_WATCH_TASK_KEY = 'dorian-pending-watch-task'

export function liveViewHref(message) {
  const match = String(message || '').match(/Watch live:\s*(\/api\/sessions\/[^\s]+)/)
  return match ? match[1] : ''
}

export function findTaskWatchUrl(task) {
  const logs = Array.isArray(task?.logs) ? task.logs : []
  for (let i = logs.length - 1; i >= 0; i -= 1) {
    const href = liveViewHref(logs[i]?.message)
    if (href) return href
  }
  return ''
}

export function requestTaskWatch(taskId) {
  const id = Number(taskId)
  if (!id) return
  try {
    sessionStorage.setItem(PENDING_WATCH_TASK_KEY, String(id))
  } catch (_) {
    // Ignore private-mode / storage failures.
  }
}

export function peekPendingWatchTaskId() {
  try {
    return Number(sessionStorage.getItem(PENDING_WATCH_TASK_KEY) || 0) || 0
  } catch (_) {
    return 0
  }
}

export function clearPendingWatchTask(taskId = 0) {
  try {
    const pending = peekPendingWatchTaskId()
    if (!taskId || pending === Number(taskId)) {
      sessionStorage.removeItem(PENDING_WATCH_TASK_KEY)
    }
  } catch (_) {
    // Ignore.
  }
}
