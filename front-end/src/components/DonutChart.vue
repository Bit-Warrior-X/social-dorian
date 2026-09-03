<template>
  <div class="donut" :style="{ width: size + 'px', height: size + 'px' }">
    <svg :viewBox="`0 0 ${size} ${size}`" role="img" :aria-label="ariaLabel">
      <circle
        class="donut__track"
        :cx="center"
        :cy="center"
        :r="radius"
        :stroke-width="thickness"
        fill="none"
      />
      <circle
        v-for="arc in arcs"
        :key="arc.label"
        :cx="center"
        :cy="center"
        :r="radius"
        fill="none"
        :stroke="arc.color"
        :stroke-width="thickness"
        :stroke-dasharray="`${arc.length} ${circumference - arc.length}`"
        :stroke-dashoffset="-arc.offset"
        :transform="`rotate(-90 ${center} ${center})`"
      />
    </svg>
    <div class="donut__center">
      <span class="donut__value">{{ centerValue }}</span>
      <span class="donut__label">{{ centerLabel }}</span>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  // [{ label, value, color }] — zero-value slices are skipped.
  segments: { type: Array, default: () => [] },
  size: { type: Number, default: 132 },
  thickness: { type: Number, default: 14 },
  centerValue: { type: [String, Number], default: '' },
  centerLabel: { type: String, default: '' },
})

const center = computed(() => props.size / 2)
const radius = computed(() => (props.size - props.thickness) / 2)
const circumference = computed(() => 2 * Math.PI * radius.value)

const total = computed(() =>
  props.segments.reduce((sum, segment) => sum + Math.max(0, segment.value || 0), 0),
)

const arcs = computed(() => {
  if (total.value === 0) return []

  let offset = 0
  return props.segments
    .filter((segment) => (segment.value || 0) > 0)
    .map((segment) => {
      const length = (segment.value / total.value) * circumference.value
      const arc = { ...segment, length, offset }
      offset += length
      return arc
    })
})

const ariaLabel = computed(() =>
  props.segments
    .filter((segment) => (segment.value || 0) > 0)
    .map((segment) => `${segment.label}: ${segment.value}`)
    .join(', ') || 'No data',
)
</script>

<style scoped>
.donut {
  position: relative;
  flex-shrink: 0;
}

svg {
  display: block;
  width: 100%;
  height: 100%;
}

.donut__track {
  stroke: var(--panel-raised);
}

.donut__center {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 1px;
  pointer-events: none;
}

.donut__value {
  font-family: var(--mono);
  font-size: 22px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.donut__label {
  font-family: var(--mono);
  font-size: 10px;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--text-faint);
}
</style>
