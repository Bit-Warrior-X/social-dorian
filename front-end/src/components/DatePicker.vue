<template>
  <div class="datepicker" ref="rootEl">
    <button
      :id="inputId"
      ref="triggerEl"
      class="datepicker__trigger"
      type="button"
      :aria-expanded="open"
      aria-haspopup="dialog"
      @click="toggle"
    >
      <span :class="{ muted: !modelValue }">{{ displayValue }}</span>
      <i class="ti ti-calendar" aria-hidden="true" />
    </button>

    <Teleport to="body">
      <div
        v-if="open"
        class="datepicker__popover"
        role="dialog"
        :aria-label="label || 'Choose date'"
        :style="popoverStyle"
      >
        <div class="datepicker__nav">
          <button class="nav-btn" type="button" aria-label="Previous month" @click="shiftMonth(-1)">
            <i class="ti ti-chevron-left" aria-hidden="true" />
          </button>
          <div class="datepicker__selectors">
            <select
              class="datepicker__select datepicker__select--month"
              :value="viewMonth"
              aria-label="Month"
              @change="setMonth(Number($event.target.value))"
            >
              <option v-for="(name, index) in months" :key="name" :value="index">
                {{ name }}
              </option>
            </select>
            <select
              class="datepicker__select datepicker__select--year"
              :value="viewYear"
              aria-label="Year"
              @change="setYear(Number($event.target.value))"
            >
              <option v-for="year in years" :key="year" :value="year">
                {{ year }}
              </option>
            </select>
          </div>
          <button class="nav-btn" type="button" aria-label="Next month" @click="shiftMonth(1)">
            <i class="ti ti-chevron-right" aria-hidden="true" />
          </button>
        </div>

        <div class="datepicker__weekdays">
          <span v-for="day in weekdays" :key="day">{{ day }}</span>
        </div>

        <div class="datepicker__grid">
          <button
            v-for="(cell, index) in cells"
            :key="index"
            type="button"
            class="day"
            :class="{
              'is-outside': !cell.inMonth,
              'is-selected': cell.iso === modelValue,
              'is-today': cell.iso === todayIso,
            }"
            :disabled="!cell.inMonth"
            @click="selectDay(cell)"
          >
            {{ cell.day }}
          </button>
        </div>

        <div class="datepicker__footer">
          <button class="btn btn-icon" type="button" @click="selectToday">Today</button>
          <button class="btn btn-icon" type="button" @click="clear">Clear</button>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'

const props = defineProps({
  modelValue: { type: String, default: '' },
  inputId: { type: String, default: '' },
  label: { type: String, default: '' },
})

const emit = defineEmits(['update:modelValue'])

const open = ref(false)
const rootEl = ref(null)
const triggerEl = ref(null)
const popoverStyle = ref({})
const view = ref(startOfMonth(parseIso(props.modelValue) || new Date()))

const weekdays = ['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa']
const months = [
  'January',
  'February',
  'March',
  'April',
  'May',
  'June',
  'July',
  'August',
  'September',
  'October',
  'November',
  'December',
]
const todayIso = toIso(new Date())
const currentYear = new Date().getFullYear()

const viewMonth = computed(() => view.value.getMonth())
const viewYear = computed(() => view.value.getFullYear())

const years = computed(() => {
  const list = Array.from({ length: 121 }, (_, i) => currentYear - i)
  const y = viewYear.value
  if (y > currentYear) return [y, ...list]
  if (y < currentYear - 120) return [...list, y]
  return list
})

const displayValue = computed(() => {
  if (!props.modelValue) return 'Select date'
  const date = parseIso(props.modelValue)
  if (!date) return props.modelValue
  return date.toLocaleDateString(undefined, {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  })
})

const cells = computed(() => {
  const year = view.value.getFullYear()
  const month = view.value.getMonth()
  const first = new Date(year, month, 1)
  const startOffset = first.getDay()
  const daysInMonth = new Date(year, month + 1, 0).getDate()
  const prevDays = new Date(year, month, 0).getDate()

  const list = []
  for (let i = 0; i < 42; i++) {
    const dayNum = i - startOffset + 1
    let date
    let inMonth = true
    if (dayNum < 1) {
      date = new Date(year, month - 1, prevDays + dayNum)
      inMonth = false
    } else if (dayNum > daysInMonth) {
      date = new Date(year, month + 1, dayNum - daysInMonth)
      inMonth = false
    } else {
      date = new Date(year, month, dayNum)
    }
    list.push({
      day: date.getDate(),
      iso: toIso(date),
      inMonth,
    })
  }
  return list
})

watch(
  () => props.modelValue,
  (value) => {
    const parsed = parseIso(value)
    if (parsed) view.value = startOfMonth(parsed)
  },
)

async function toggle() {
  open.value = !open.value
  if (open.value) {
    const parsed = parseIso(props.modelValue)
    view.value = startOfMonth(parsed || new Date())
    await nextTick()
    positionPopover()
  }
}

function positionPopover() {
  if (!triggerEl.value) return
  const rect = triggerEl.value.getBoundingClientRect()
  const width = 280
  const left = Math.min(rect.left, window.innerWidth - width - 12)
  let top = rect.bottom + 6
  const estimatedHeight = 320
  if (top + estimatedHeight > window.innerHeight - 12) {
    top = Math.max(12, rect.top - estimatedHeight - 6)
  }
  popoverStyle.value = {
    top: `${top}px`,
    left: `${Math.max(12, left)}px`,
  }
}

function shiftMonth(delta) {
  view.value = new Date(view.value.getFullYear(), view.value.getMonth() + delta, 1)
}

function setMonth(month) {
  view.value = new Date(view.value.getFullYear(), month, 1)
}

function setYear(year) {
  view.value = new Date(year, view.value.getMonth(), 1)
}

function selectDay(cell) {
  if (!cell.inMonth) return
  emit('update:modelValue', cell.iso)
  open.value = false
}

function selectToday() {
  emit('update:modelValue', toIso(new Date()))
  view.value = startOfMonth(new Date())
  open.value = false
}

function clear() {
  emit('update:modelValue', '')
  open.value = false
}

function onDocumentClick(event) {
  if (!open.value) return
  const inTrigger = rootEl.value?.contains(event.target)
  const inPopover = event.target.closest?.('.datepicker__popover')
  if (!inTrigger && !inPopover) open.value = false
}

function onKeydown(event) {
  if (event.key === 'Escape') open.value = false
}

function onResize() {
  if (open.value) positionPopover()
}

function parseIso(value) {
  if (!value || !/^\d{4}-\d{2}-\d{2}$/.test(value)) return null
  const [y, m, d] = value.split('-').map(Number)
  const date = new Date(y, m - 1, d)
  if (Number.isNaN(date.getTime())) return null
  return date
}

function toIso(date) {
  const y = date.getFullYear()
  const m = String(date.getMonth() + 1).padStart(2, '0')
  const d = String(date.getDate()).padStart(2, '0')
  return `${y}-${m}-${d}`
}

function startOfMonth(date) {
  return new Date(date.getFullYear(), date.getMonth(), 1)
}

onMounted(() => {
  document.addEventListener('mousedown', onDocumentClick)
  document.addEventListener('keydown', onKeydown)
  window.addEventListener('resize', onResize)
  window.addEventListener('scroll', onResize, true)
})

onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onDocumentClick)
  document.removeEventListener('keydown', onKeydown)
  window.removeEventListener('resize', onResize)
  window.removeEventListener('scroll', onResize, true)
})
</script>

<style scoped>
.datepicker {
  position: relative;
}

.datepicker__trigger {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 10px;
  border: 0.5px solid var(--hairline-strong);
  border-radius: var(--radius);
  background: var(--panel);
  color: var(--text);
  font: inherit;
  font-size: 13px;
  text-align: left;
  cursor: pointer;
}

.datepicker__trigger:hover {
  border-color: var(--text-faint);
}

.datepicker__trigger:focus {
  outline: none;
  border-color: var(--viper-500);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--viper-500) 28%, transparent);
}

.datepicker__trigger .muted {
  color: var(--text-faint);
}

.datepicker__trigger .ti {
  color: var(--viper-400);
  font-size: 16px;
}
</style>

<style>
.datepicker__popover {
  position: fixed;
  z-index: 400;
  width: 280px;
  padding: 12px;
  border: 0.5px solid var(--hairline);
  border-radius: 12px;
  background: var(--panel);
  box-shadow: 0 16px 40px rgba(0, 0, 0, 0.45);
  color: var(--text);
  font-family: var(--sans);
}

.datepicker__nav {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
  margin-bottom: 10px;
}

.datepicker__selectors {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  flex: 1;
  justify-content: center;
}

.datepicker__select {
  appearance: none;
  border: 0.5px solid var(--hairline);
  border-radius: 6px;
  background: var(--panel-raised)
    url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%238B978F' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpolyline points='6 9 12 15 18 9'/%3E%3C/svg%3E")
    no-repeat right 6px center;
  color: var(--text);
  font-family: var(--mono);
  font-size: 12px;
  font-weight: 600;
  padding: 5px 22px 5px 8px;
  cursor: pointer;
}

.datepicker__select:focus {
  outline: none;
  border-color: var(--viper-500);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--viper-500) 28%, transparent);
}

.datepicker__select--month {
  min-width: 0;
  flex: 1.3;
}

.datepicker__select--year {
  width: 78px;
  flex: 0 0 auto;
}

.datepicker__select option {
  background: var(--panel);
  color: var(--text);
}

.datepicker__popover .nav-btn {
  width: 28px;
  height: 28px;
  display: grid;
  place-items: center;
  border: 0.5px solid var(--hairline);
  border-radius: 6px;
  background: var(--panel-raised);
  color: var(--text-dim);
  cursor: pointer;
  flex-shrink: 0;
}

.datepicker__popover .nav-btn:hover {
  color: var(--viper-400);
  border-color: var(--viper-700);
}

.datepicker__weekdays {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 2px;
  margin-bottom: 4px;
}

.datepicker__weekdays span {
  text-align: center;
  font-family: var(--mono);
  font-size: 10.5px;
  color: var(--text-faint);
  letter-spacing: 0.04em;
  text-transform: uppercase;
  padding: 4px 0;
}

.datepicker__grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 2px;
}

.datepicker__popover .day {
  aspect-ratio: 1;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--text);
  font-family: var(--mono);
  font-size: 12px;
  cursor: pointer;
}

.datepicker__popover .day:hover:not(:disabled) {
  background: var(--panel-raised);
}

.datepicker__popover .day.is-outside {
  color: var(--text-faint);
  opacity: 0.35;
  cursor: default;
}

.datepicker__popover .day.is-today:not(.is-selected) {
  color: var(--gold-500);
  box-shadow: inset 0 0 0 1px var(--gold-500);
}

.datepicker__popover .day.is-selected {
  background: var(--viper-500);
  color: #08120e;
  font-weight: 600;
}

.datepicker__popover .day.is-selected:hover {
  background: var(--viper-400);
}

.datepicker__footer {
  display: flex;
  justify-content: space-between;
  margin-top: 10px;
  padding-top: 8px;
  border-top: 0.5px solid var(--hairline);
}

.datepicker__footer .btn-icon {
  color: var(--viper-400);
  font-size: 12px;
}
</style>
