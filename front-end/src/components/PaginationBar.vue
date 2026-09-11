<template>
  <div v-if="total > 0" class="pager" role="navigation" aria-label="Pagination">
    <div class="pager__meta">
      <span>
        {{ from }}–{{ to }} of {{ total }}
      </span>
      <label class="pager__size">
        <span class="sr-only">Rows per page</span>
        <select :value="pageSize" @change="onSizeChange">
          <option v-for="size in pageSizeOptions" :key="size" :value="size">
            {{ size }} / page
          </option>
        </select>
      </label>
    </div>

    <div class="pager__controls">
      <button
        class="btn btn-icon"
        type="button"
        aria-label="First page"
        :disabled="page <= 1"
        @click="$emit('update:page', 1)"
      >
        <i class="ti ti-chevrons-left" aria-hidden="true" />
      </button>
      <button
        class="btn btn-icon"
        type="button"
        aria-label="Previous page"
        :disabled="page <= 1"
        @click="$emit('update:page', page - 1)"
      >
        <i class="ti ti-chevron-left" aria-hidden="true" />
      </button>
      <span class="pager__page">Page {{ page }} / {{ totalPages }}</span>
      <button
        class="btn btn-icon"
        type="button"
        aria-label="Next page"
        :disabled="page >= totalPages"
        @click="$emit('update:page', page + 1)"
      >
        <i class="ti ti-chevron-right" aria-hidden="true" />
      </button>
      <button
        class="btn btn-icon"
        type="button"
        aria-label="Last page"
        :disabled="page >= totalPages"
        @click="$emit('update:page', totalPages)"
      >
        <i class="ti ti-chevrons-right" aria-hidden="true" />
      </button>
    </div>
  </div>
</template>

<script setup>
defineProps({
  page: { type: Number, required: true },
  pageSize: { type: Number, required: true },
  total: { type: Number, required: true },
  totalPages: { type: Number, required: true },
  from: { type: Number, required: true },
  to: { type: Number, required: true },
  pageSizeOptions: {
    type: Array,
    default: () => [10, 20, 50, 100],
  },
})

const emit = defineEmits(['update:page', 'update:pageSize'])

function onSizeChange(event) {
  emit('update:pageSize', Number(event.target.value))
  emit('update:page', 1)
}
</script>

<style scoped>
.pager {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  padding: 10px 12px;
  border-top: 0.5px solid var(--hairline);
  background: var(--panel);
  font-size: 13px;
  color: var(--text-dim);
}

.pager__meta {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.pager__size select {
  width: auto;
  min-width: 108px;
  padding: 4px 8px;
  font-size: 12px;
}

.pager__controls {
  display: flex;
  align-items: center;
  gap: 4px;
}

.pager__page {
  font-family: var(--mono);
  font-size: 12px;
  min-width: 7.5rem;
  text-align: center;
  color: var(--text-faint);
}

.pager .btn-icon:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}
</style>
