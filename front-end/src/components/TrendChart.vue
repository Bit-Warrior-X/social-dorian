<template>
  <div class="trend" @mouseleave="activeIndex = -1">
    <svg class="trend__svg" :viewBox="`0 0 ${WIDTH} ${HEIGHT}`" role="img" :aria-label="ariaLabel">
      <defs>
        <linearGradient :id="gradientId" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stop-color="var(--viper-400)" stop-opacity="0.32" />
          <stop offset="100%" stop-color="var(--viper-400)" stop-opacity="0" />
        </linearGradient>
      </defs>

      <line
        v-for="y in gridLines"
        :key="'grid-' + y"
        class="trend__grid"
        :x1="PAD_X"
        :x2="WIDTH - PAD_X"
        :y1="y"
        :y2="y"
      />

      <path class="trend__area" :d="areaPath" :fill="`url(#${gradientId})`" />
      <path class="trend__line" :d="linePath" />

      <circle
        v-for="(point, index) in coords"
        :key="'dot-' + point.month"
        class="trend__dot"
        :class="{ 'is-active': index === activeIndex }"
        :cx="point.x"
        :cy="point.y"
        :r="index === activeIndex ? 4.5 : 2.5"
      />

      <text
        v-for="(point, index) in coords"
        :key="'label-' + point.month"
        class="trend__tick"
        :class="{ 'is-active': index === activeIndex }"
        :x="point.x"
        :y="HEIGHT - 8"
        text-anchor="middle"
      >
        {{ showTick(index) ? point.short : '' }}
      </text>

      <rect
        v-for="(point, index) in coords"
        :key="'hit-' + point.month"
        class="trend__hit"
        :x="point.x - hitWidth / 2"
        y="0"
        :width="hitWidth"
        :height="HEIGHT"
        @mouseenter="activeIndex = index"
      />
    </svg>

    <div v-if="activePoint" class="trend__tip" :style="tipStyle">
      <strong>{{ activePoint.count }}</strong>
      <span>{{ activePoint.long }}</span>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'

const props = defineProps({
  points: { type: Array, default: () => [] },
  unit: { type: String, default: 'connections' },
})

const WIDTH = 640
const HEIGHT = 172
const PAD_X = 14
const PAD_TOP = 16
const PAD_BOTTOM = 32

let uid = 0
const gradientId = `trend-fill-${(uid += 1)}-${Math.random().toString(36).slice(2, 7)}`

const activeIndex = ref(-1)

const plotHeight = HEIGHT - PAD_TOP - PAD_BOTTOM
const baseline = HEIGHT - PAD_BOTTOM

const gridLines = [PAD_TOP, PAD_TOP + plotHeight / 2, baseline]

const maxCount = computed(() => Math.max(1, ...props.points.map((p) => p.count || 0)))

const coords = computed(() => {
  const total = props.points.length
  if (total === 0) return []
  const span = WIDTH - PAD_X * 2
  const step = total > 1 ? span / (total - 1) : 0

  return props.points.map((point, index) => {
    const count = point.count || 0
    return {
      month: point.month,
      count,
      short: shortMonth(point.month),
      long: longMonth(point.month),
      x: total > 1 ? PAD_X + index * step : WIDTH / 2,
      y: baseline - (count / maxCount.value) * plotHeight,
    }
  })
})

const hitWidth = computed(() => {
  const total = coords.value.length
  return total > 1 ? (WIDTH - PAD_X * 2) / (total - 1) : WIDTH
})

const linePath = computed(() =>
  coords.value.map((point, index) => `${index === 0 ? 'M' : 'L'}${point.x} ${point.y}`).join(' '),
)

const areaPath = computed(() => {
  const points = coords.value
  if (points.length === 0) return ''
  const line = points.map((point) => `L${point.x} ${point.y}`).join(' ')
  return `M${points[0].x} ${baseline} ${line} L${points[points.length - 1].x} ${baseline} Z`
})

const activePoint = computed(() => coords.value[activeIndex.value] || null)

const tipStyle = computed(() => {
  const point = activePoint.value
  if (!point) return {}
  return {
    left: `${(point.x / WIDTH) * 100}%`,
    top: `${(point.y / HEIGHT) * 100}%`,
  }
})

const ariaLabel = computed(() => {
  const total = props.points.reduce((sum, point) => sum + (point.count || 0), 0)
  return `${total} ${props.unit} across the last ${props.points.length} months`
})

// With a year of buckets every label collides, so thin them out but always
// keep the most recent month visible.
function showTick(index) {
  const total = coords.value.length
  if (total <= 6) return true
  return index === total - 1 || (total - 1 - index) % 2 === 0
}

function monthDate(month) {
  const [year, index] = String(month || '').split('-')
  return new Date(Date.UTC(Number(year), Number(index) - 1, 1))
}

function shortMonth(month) {
  const date = monthDate(month)
  if (Number.isNaN(date.getTime())) return month
  return date.toLocaleDateString(undefined, { month: 'short', timeZone: 'UTC' })
}

function longMonth(month) {
  const date = monthDate(month)
  if (Number.isNaN(date.getTime())) return month
  return date.toLocaleDateString(undefined, { month: 'long', year: 'numeric', timeZone: 'UTC' })
}
</script>

<style scoped>
.trend {
  position: relative;
}

.trend__svg {
  display: block;
  width: 100%;
  height: auto;
  overflow: visible;
}

.trend__grid {
  stroke: var(--hairline);
  stroke-width: 1;
}

.trend__line {
  fill: none;
  stroke: var(--viper-400);
  stroke-width: 2;
  stroke-linejoin: round;
  stroke-linecap: round;
}

.trend__dot {
  fill: var(--viper-400);
  transition: r 0.12s ease;
}

.trend__dot.is-active {
  stroke: var(--bg);
  stroke-width: 2;
}

.trend__tick {
  fill: var(--text-faint);
  font-family: var(--mono);
  font-size: 11px;
}

.trend__tick.is-active {
  fill: var(--viper-400);
}

.trend__hit {
  fill: transparent;
}

.trend__tip {
  position: absolute;
  transform: translate(-50%, calc(-100% - 10px));
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1px;
  padding: 5px 9px;
  border: 0.5px solid var(--hairline-strong);
  border-radius: 8px;
  background: var(--panel-raised);
  pointer-events: none;
  white-space: nowrap;
  z-index: 2;
}

.trend__tip strong {
  font-family: var(--mono);
  font-size: 14px;
  color: var(--viper-400);
}

.trend__tip span {
  font-size: 11px;
  color: var(--text-dim);
}
</style>
