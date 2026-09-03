import { reactive } from 'vue'

const state = reactive({
  items: [],
})

let nextId = 1

function push(type, message, options = {}) {
  const text = String(message || '').trim()
  if (!text) return null

  const id = nextId++
  const duration = options.duration ?? (type === 'error' ? 7000 : 4500)
  const item = {
    id,
    type,
    message: text,
    createdAt: Date.now(),
  }

  state.items.push(item)

  if (duration > 0) {
    window.setTimeout(() => dismiss(id), duration)
  }

  return id
}

function dismiss(id) {
  const index = state.items.findIndex((item) => item.id === id)
  if (index >= 0) state.items.splice(index, 1)
}

function clear() {
  state.items.splice(0, state.items.length)
}

export function useNotify() {
  return {
    notifications: state.items,
    notifySuccess(message, options) {
      return push('success', message, options)
    },
    notifyError(message, options) {
      return push('error', message, options)
    },
    notifyInfo(message, options) {
      return push('info', message, options)
    },
    dismiss,
    clear,
  }
}
