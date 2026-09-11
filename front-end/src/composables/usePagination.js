import { computed, ref, unref, watch } from 'vue'

const DEFAULT_PAGE_SIZE = 20
const PAGE_SIZE_OPTIONS = [10, 20, 50, 100]

/**
 * Client-side pagination over a reactive list (usually a filtered computed).
 * Resets to page 1 when the source length or page size changes in a way that
 * makes the current page empty.
 */
export function usePagination(source, { pageSize: initialSize = DEFAULT_PAGE_SIZE } = {}) {
  const page = ref(1)
  const pageSize = ref(initialSize)

  const items = computed(() => {
    const list = unref(source)
    return Array.isArray(list) ? list : []
  })

  const total = computed(() => items.value.length)

  const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value) || 1))

  const pageItems = computed(() => {
    const start = (page.value - 1) * pageSize.value
    return items.value.slice(start, start + pageSize.value)
  })

  const from = computed(() => (total.value === 0 ? 0 : (page.value - 1) * pageSize.value + 1))
  const to = computed(() => Math.min(page.value * pageSize.value, total.value))

  watch([total, pageSize], () => {
    if (page.value > totalPages.value) {
      page.value = totalPages.value
    }
    if (page.value < 1) page.value = 1
  })

  function setPage(next) {
    const n = Number(next)
    if (!Number.isFinite(n)) return
    page.value = Math.min(totalPages.value, Math.max(1, Math.floor(n)))
  }

  function nextPage() {
    setPage(page.value + 1)
  }

  function prevPage() {
    setPage(page.value - 1)
  }

  function reset() {
    page.value = 1
  }

  return {
    page,
    pageSize,
    pageItems,
    total,
    totalPages,
    from,
    to,
    setPage,
    nextPage,
    prevPage,
    reset,
    pageSizeOptions: PAGE_SIZE_OPTIONS,
  }
}

export { DEFAULT_PAGE_SIZE, PAGE_SIZE_OPTIONS }
