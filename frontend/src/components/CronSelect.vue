<template>
  <div class="cb">

    <!-- Level: frequency unit + interval -->
    <div class="cb-row">
      <span class="cb-lbl">{{ $t('cron.every') }}</span>
      <div class="cb-controls">
        <select v-if="unit !== 'weeks'" v-model="interval" class="cb-sel cb-sel--xs">
          <option v-for="n in intervalOptions" :key="n" :value="n">{{ n }}</option>
        </select>
        <select v-model="unit" class="cb-sel">
          <option value="minutes">{{ $t('cron.unit.minutes') }}</option>
          <option value="hours">{{ $t('cron.unit.hours') }}</option>
          <option value="days">{{ $t('cron.unit.days') }}</option>
          <option value="weeks">{{ $t('cron.unit.weeks') }}</option>
          <option value="months">{{ $t('cron.unit.months') }}</option>
        </select>
      </div>
    </div>

    <!-- Level: day of week (weeks only) -->
    <Transition name="cb-slide">
      <div v-if="unit === 'weeks'" class="cb-row">
        <span class="cb-lbl">{{ $t('cron.onWeekday') }}</span>
        <div class="cb-controls">
          <select v-model="atDayOfWeek" class="cb-sel">
            <option v-for="i in 7" :key="i - 1" :value="i - 1">{{ $t(`cron.weekdays.${i - 1}`) }}</option>
          </select>
        </div>
      </div>
    </Transition>

    <!-- Level: day of month (months only) -->
    <Transition name="cb-slide">
      <div v-if="unit === 'months'" class="cb-row">
        <span class="cb-lbl">{{ $t('cron.onDay') }}</span>
        <div class="cb-controls">
          <select v-model="atDayOfMonth" class="cb-sel cb-sel--xs">
            <option v-for="d in 28" :key="d" :value="d">{{ d }}</option>
          </select>
        </div>
      </div>
    </Transition>

    <!-- Level: hour (days / weeks / months) -->
    <Transition name="cb-slide">
      <div v-if="unit === 'days' || unit === 'weeks' || unit === 'months'" class="cb-row">
        <span class="cb-lbl">{{ $t('cron.atTime') }}</span>
        <div class="cb-controls">
          <select v-model="atHour" class="cb-sel cb-sel--xs">
            <option v-for="h in 24" :key="h - 1" :value="h - 1">{{ String(h - 1).padStart(2, '0') }}</option>
          </select>
        </div>
      </div>
    </Transition>

    <!-- Level: minute (hours / days / weeks / months) -->
    <Transition name="cb-slide">
      <div v-if="unit !== 'minutes'" class="cb-row">
        <span class="cb-lbl">{{ $t('cron.atMinute') }}</span>
        <div class="cb-controls">
          <select v-model="atMinute" class="cb-sel cb-sel--xs">
            <option v-for="m in MINUTE_OPTIONS" :key="m" :value="m">:{{ String(m).padStart(2, '0') }}</option>
          </select>
        </div>
      </div>
    </Transition>

    <!-- Timezone hint (days / weeks / months — where local time matters) -->
    <Transition name="cb-slide">
      <div v-if="unit === 'days' || unit === 'weeks' || unit === 'months'" class="cb-row">
        <span class="cb-lbl"></span>
        <span class="cb-tz-hint">{{ $t('cron.tz', { tz: userTzLabel }) }}</span>
      </div>
    </Transition>

    <!-- Preview chip -->
    <div class="cb-preview">
      <span class="cb-preview__label">{{ $t('cron.preview') }}</span>
      <code class="cb-preview__code">{{ cronExpr }}</code>
    </div>

  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'

type FrequencyUnit = 'minutes' | 'hours' | 'days' | 'weeks' | 'months'

const MINUTE_OPTIONS = [0, 5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55]

const INTERVAL_OPTIONS: Record<FrequencyUnit, number[]> = {
  minutes: [5, 10, 15, 20, 30, 45],
  hours:   [1, 2, 3, 4, 6, 8, 12],
  days:    [1, 2, 3, 7, 14],
  weeks:   [1],
  months:  [1, 2, 3, 6],
}

const props = defineProps<{ modelValue: string }>()
const emit  = defineEmits<{ 'update:modelValue': [value: string] }>()

const unit        = ref<FrequencyUnit>('hours')
const interval    = ref(1)
const atMinute    = ref(0)
const atHour      = ref(0)
const atDayOfWeek = ref(1)   // 1 = Monday
const atDayOfMonth = ref(1)

const intervalOptions = computed(() => INTERVAL_OPTIONS[unit.value])

// ── Timezone helpers ───────────────────────────────────────────────────────
// Day 15 is used as the reference date to avoid month-boundary edge cases.

/** Local H:M → UTC. Returns h, m and dayOffset (-1 / 0 / +1). */
function localToUtc(localH: number, localM: number) {
  const now = new Date()
  const d = new Date(now.getFullYear(), now.getMonth(), 15, localH, localM)
  return { h: d.getUTCHours(), m: d.getUTCMinutes(), dayOffset: d.getUTCDate() - 15 }
}

/** UTC H:M → local. Returns h, m and dayOffset (-1 / 0 / +1). */
function utcToLocal(utcH: number, utcM: number) {
  const now = new Date()
  const d = new Date(Date.UTC(now.getFullYear(), now.getMonth(), 15, utcH, utcM))
  return { h: d.getHours(), m: d.getMinutes(), dayOffset: d.getDate() - 15 }
}

/** Human-readable label shown in the TZ hint, e.g. "Europe/Paris (UTC+02)". */
const userTzLabel = (() => {
  const tz = Intl.DateTimeFormat().resolvedOptions().timeZone
  const off = -new Date().getTimezoneOffset()          // minutes, positive = east
  const h   = Math.floor(Math.abs(off) / 60)
  const m   = Math.abs(off) % 60
  const s   = off >= 0 ? '+' : '-'
  const label = `UTC${s}${String(h).padStart(2, '0')}${m ? ':' + String(m).padStart(2, '0') : ''}`
  return `${tz} (${label})`
})()

// ── Cron expression builder ────────────────────────────────────────────────
const cronExpr = computed<string>(() => {
  const M = atMinute.value
  const H = atHour.value
  const N = interval.value
  switch (unit.value) {
    // Minutes / hours: relative intervals — no wall-clock anchor, no TZ conversion needed
    case 'minutes': return `*/${N} * * * *`
    case 'hours':   return `${M} */${N} * * *`
    // Days / weeks / months: anchored to a specific local time — convert to UTC
    case 'days': {
      const u = localToUtc(H, M)
      return `${u.m} ${u.h} */${N} * *`
    }
    case 'weeks': {
      const u = localToUtc(H, M)
      const utcDow = ((atDayOfWeek.value + u.dayOffset) % 7 + 7) % 7
      return `${u.m} ${u.h} * * ${utcDow}`
    }
    case 'months': {
      const u = localToUtc(H, M)
      const utcDay = Math.min(28, Math.max(1, atDayOfMonth.value + u.dayOffset))
      return `${u.m} ${u.h} ${utcDay} */${N} *`
    }
  }
})

// Emit whenever internal state changes
watch(cronExpr, (v) => emit('update:modelValue', v), { immediate: false })

// ── Parse incoming expression ───────────────────────────────────────────────
function clampToOptions(value: number, options: number[]): number {
  return options.reduce((prev, curr) =>
    Math.abs(curr - value) < Math.abs(prev - value) ? curr : prev
  )
}

function parseExpr(expr: string) {
  // Special aliases
  if (expr === '@hourly') { unit.value = 'hours'; interval.value = 1; atMinute.value = 0; return }
  if (expr === '@daily') {
    unit.value = 'days'; interval.value = 1
    const loc = utcToLocal(0, 0)
    atHour.value = loc.h; atMinute.value = clampToOptions(loc.m, MINUTE_OPTIONS)
    return
  }
  if (expr === '@weekly') {
    unit.value = 'weeks'
    const loc = utcToLocal(0, 0)
    atDayOfWeek.value = ((0 + loc.dayOffset) % 7 + 7) % 7   // 0 = Sunday UTC
    atHour.value = loc.h; atMinute.value = clampToOptions(loc.m, MINUTE_OPTIONS)
    return
  }
  if (expr === '@monthly') {
    unit.value = 'months'; interval.value = 1
    const loc = utcToLocal(0, 0)
    atDayOfMonth.value = Math.min(28, Math.max(1, 1 + loc.dayOffset))
    atHour.value = loc.h; atMinute.value = clampToOptions(loc.m, MINUTE_OPTIONS)
    return
  }

  const parts = expr.trim().split(/\s+/)
  if (parts.length !== 5) return // leave unchanged

  const [minF, hourF, domF, , dowF] = parts

  // Minutes: */N * * * *
  if (hourF === '*' && domF === '*' && minF.startsWith('*/')) {
    const n = parseInt(minF.slice(2), 10)
    unit.value = 'minutes'
    interval.value = clampToOptions(n, INTERVAL_OPTIONS.minutes)
    return
  }
  // Hours: M */N * * *
  if (domF === '*' && hourF.startsWith('*/') && !minF.startsWith('*/')) {
    const n = parseInt(hourF.slice(2), 10)
    unit.value = 'hours'
    interval.value = clampToOptions(n, INTERVAL_OPTIONS.hours)
    atMinute.value = clampToOptions(parseInt(minF, 10) || 0, MINUTE_OPTIONS)
    return
  }
  // Days: M H */N * *  — stored in UTC, display in local
  if (domF.startsWith('*/') && dowF === '*') {
    const n = parseInt(domF.slice(2), 10)
    unit.value = 'days'
    interval.value = clampToOptions(n, INTERVAL_OPTIONS.days)
    const loc = utcToLocal(parseInt(hourF, 10) || 0, parseInt(minF, 10) || 0)
    atHour.value = loc.h
    atMinute.value = clampToOptions(loc.m, MINUTE_OPTIONS)
    return
  }
  // Months: M H D */N *  — stored in UTC, display in local
  if (parts[3].startsWith('*/') && dowF === '*') {
    const n = parseInt(parts[3].slice(2), 10)
    unit.value = 'months'
    interval.value = clampToOptions(n, INTERVAL_OPTIONS.months)
    const loc = utcToLocal(parseInt(hourF, 10) || 0, parseInt(minF, 10) || 0)
    atDayOfMonth.value = Math.min(28, Math.max(1, (parseInt(domF, 10) || 1) + loc.dayOffset))
    atHour.value = loc.h
    atMinute.value = clampToOptions(loc.m, MINUTE_OPTIONS)
    return
  }
  // Weeks: M H * * W  — stored in UTC, display in local
  if (domF === '*' && parts[3] === '*' && !hourF.startsWith('*/')) {
    unit.value = 'weeks'
    const utcDow = parseInt(dowF, 10) % 7
    const loc = utcToLocal(parseInt(hourF, 10) || 0, parseInt(minF, 10) || 0)
    atDayOfWeek.value = ((utcDow + loc.dayOffset) % 7 + 7) % 7
    atHour.value = loc.h
    atMinute.value = clampToOptions(loc.m, MINUTE_OPTIONS)
    return
  }
  // Fallback
  unit.value = 'hours'
  interval.value = 1
  atMinute.value = 0
}

// Watch incoming model value (handles initial load + edits)
watch(
  () => props.modelValue,
  (val) => {
    if (val && val !== cronExpr.value) parseExpr(val)
  },
  { immediate: true }
)
</script>

<style scoped>
/* ── Container ─────────────────────────────────────────────────────────── */
.cb {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

/* ── Row: 2-column grid — label always 4.5rem, controls fill the rest ───── */
.cb-row {
  display: grid;
  grid-template-columns: 4.5rem 1fr;
  align-items: center;
  column-gap: 0.6rem;
  min-height: 2rem;
  overflow: hidden;
}

.cb-lbl {
  text-align: right;
  font-size: 0.78rem;
  font-weight: 500;
  color: rgba(180, 185, 255, 0.5);
  text-transform: uppercase;
  letter-spacing: 0.06em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  user-select: none;
}

.cb-controls {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  flex-wrap: nowrap;
}

/* ── Select elements ────────────────────────────────────────────────────── */
.cb-sel {
  appearance: none;
  -webkit-appearance: none;
  background: rgba(255, 255, 255, 0.06)
    url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='9' height='5'%3E%3Cpath d='M0 0l4.5 5L9 0z' fill='rgba(170,175,255,.5)'/%3E%3C/svg%3E")
    no-repeat right 0.45rem center / 8px 4px;
  border: 1px solid rgba(120, 140, 255, 0.18);
  border-radius: 7px;
  color: var(--color-text, #e8e8f0);
  font-size: 0.85rem;
  padding: 0.3rem 1.5rem 0.3rem 0.6rem;
  cursor: pointer;
  transition: border-color 0.15s, background-color 0.15s, box-shadow 0.15s;
  white-space: nowrap;
}

.cb-sel:hover {
  border-color: rgba(120, 140, 255, 0.4);
  background-color: rgba(255, 255, 255, 0.09);
}

.cb-sel:focus {
  outline: none;
  border-color: rgba(120, 140, 255, 0.65);
  background-color: rgba(255, 255, 255, 0.1);
  box-shadow: 0 0 0 2px rgba(120, 140, 255, 0.14);
}

/* Extra-small variant for numeric selects (interval, hour, minute, day) */
.cb-sel--xs {
  padding-right: 1.5rem;
  min-width: 4.2rem;
  text-align: center;
}

/* ── Preview chip ───────────────────────────────────────────────────────── */
.cb-preview {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  margin-top: 0.25rem;
  margin-left: calc(4.5rem + 0.6rem); /* align with controls column */
  padding: 0.2rem 0.55rem;
  background: rgba(120, 140, 255, 0.06);
  border: 1px solid rgba(120, 140, 255, 0.14);
  border-radius: 20px;
  width: fit-content;
}

.cb-preview__label {
  font-size: 0.68rem;
  color: rgba(180, 185, 255, 0.4);
  text-transform: uppercase;
  letter-spacing: 0.06em;
  user-select: none;
}

.cb-preview__code {
  font-family: 'JetBrains Mono', 'Fira Mono', ui-monospace, monospace;
  font-size: 0.76rem;
  color: rgba(180, 192, 255, 0.8);
  letter-spacing: 0.04em;
}

/* ── Timezone hint ──────────────────────────────────────────────────────── */
.cb-tz-hint {
  font-size: 0.7rem;
  font-style: italic;
  color: rgba(180, 185, 255, 0.35);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* ── Slide transition ───────────────────────────────────────────────────── */
.cb-slide-enter-active,
.cb-slide-leave-active {
  transition: opacity 0.16s ease, max-height 0.2s ease, margin 0.2s ease;
  max-height: 2.5rem;
  overflow: hidden;
}

.cb-slide-enter-from,
.cb-slide-leave-to {
  opacity: 0;
  max-height: 0;
  margin-top: 0;
  margin-bottom: 0;
}
</style>
