<script setup>
import { computed, ref, onMounted, onBeforeUnmount, watch } from 'vue'

const props = defineProps({
  modem: { type: [String, Number], required: true },
  obisCode: { type: String, required: true },
  samples: { type: Array, required: true }, // [{time, value}]
  timezone: { type: String, default: 'UTC' },
})

// ------------------------------------------------------------
// Локальное состояние карточки
// ------------------------------------------------------------
const search = ref('')
const debouncedSearch = ref('')
const copied = ref(false)
const sortKey = ref('time') // 'time' | 'value'
const sortDir = ref('desc') // 'asc' | 'desc'

let searchTimer = null
watch(search, (v) => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    debouncedSearch.value = v
  }, 150)
})

// ------------------------------------------------------------
// Утилиты
// ------------------------------------------------------------
function humanTime(unixSeconds) {
  try {
    return new Date(unixSeconds * 1000).toLocaleString('ru-RU', { timeZone: props.timezone })
  } catch (_) {
    return new Date(unixSeconds * 1000).toLocaleString('ru-RU')
  }
}

function decToHex(modem) {
  return Number(modem).toString(16)
}

function asNumber(v) {
  if (v === null || v === undefined || v === '') return NaN
  const n = Number(v)
  return Number.isFinite(n) ? n : NaN
}

function csvEscape(v) {
  const s = String(v)
  if (s.includes(',') || s.includes('"') || s.includes('\n')) {
    return `"${s.replace(/"/g, '""')}"`
  }
  return s
}

function formatNumber(n) {
  if (!Number.isFinite(n)) return '—'
  const abs = Math.abs(n)
  const digits = abs >= 100 ? 2 : abs >= 1 ? 3 : 4
  return n.toLocaleString('ru-RU', { maximumFractionDigits: digits })
}

// ------------------------------------------------------------
// Статистика (min/max/avg) — только для численных карточек
// ------------------------------------------------------------
const allNumeric = computed(() => props.samples.every((s) => !Number.isNaN(asNumber(s.value))))

const stats = computed(() => {
  if (!allNumeric.value || props.samples.length === 0) return null
  let min = Infinity
  let max = -Infinity
  let sum = 0
  for (const s of props.samples) {
    const n = asNumber(s.value)
    if (n < min) min = n
    if (n > max) max = n
    sum += n
  }
  return { min, max, avg: sum / props.samples.length }
})

// ------------------------------------------------------------
// Сортировка + предвычисленный humanTime (кешируется один раз
// на изменение timezone/samples/sortKey/sortDir, а не пересчитывается
// на каждый keystroke в поиске).
// ------------------------------------------------------------
const sortedWithHuman = computed(() => {
  const arr = new Array(props.samples.length)
  for (let i = 0; i < props.samples.length; i++) {
    const s = props.samples[i]
    arr[i] = { time: s.time, value: s.value, _human: humanTime(s.time) }
  }
  const mul = sortDir.value === 'asc' ? 1 : -1
  if (sortKey.value === 'value' && allNumeric.value) {
    arr.sort((a, b) => (asNumber(a.value) - asNumber(b.value)) * mul)
  } else if (sortKey.value === 'value') {
    arr.sort((a, b) => String(a.value).localeCompare(String(b.value)) * mul)
  } else {
    arr.sort((a, b) => (a.time - b.time) * mul)
  }
  return arr
})

// ------------------------------------------------------------
// Фильтрация (использует debounced search и предвычисленный humanTime)
// ------------------------------------------------------------
const filtered = computed(() => {
  const q = debouncedSearch.value.trim().toLowerCase()
  if (!q) return sortedWithHuman.value
  return sortedWithHuman.value.filter((s) => {
    return (
      String(s.value).toLowerCase().includes(q) ||
      String(s.time).includes(q) ||
      s._human.toLowerCase().includes(q)
    )
  })
})

function highlight(text) {
  if (!debouncedSearch.value.trim()) return String(text)
  const q = debouncedSearch.value.toLowerCase()
  const re = new RegExp(`(${q.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'ig')
  return String(text).replace(re, '<mark class="ui-mark">$1</mark>')
}

// ------------------------------------------------------------
// «Последнее» значение — самое свежее по времени независимо от сортировки
// ------------------------------------------------------------
const latest = computed(() => {
  let best = null
  for (const s of props.samples) {
    if (!best || s.time > best.time) best = s
  }
  return best
})

// ------------------------------------------------------------
// Экспорт карточки в CSV (сохраняет текущий порядок сортировки)
// ------------------------------------------------------------
async function copyCsv() {
  const rows = ['time_unix,time_human,value']
  for (const s of sortedWithHuman.value) {
    rows.push([s.time, s._human, s.value].map(csvEscape).join(','))
  }
  try {
    await navigator.clipboard.writeText(rows.join('\n'))
    copied.value = true
    setTimeout(() => (copied.value = false), 2000)
  } catch (_) {}
}

function cycleSort(key) {
  if (sortKey.value !== key) {
    sortKey.value = key
    sortDir.value = key === 'time' ? 'desc' : 'asc'
    return
  }
  sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc'
}

function sortBadge(key) {
  if (sortKey.value !== key) return ''
  return sortDir.value === 'asc' ? '↑' : '↓'
}

// ------------------------------------------------------------
// Виртуализация списка:
//   - фиксированная высота строки ROW_HEIGHT,
//   - плавающее окно рендера [start; end] пересчитывается из scrollTop,
//   - padding-спейсеры сверху и снизу резервируют полную высоту,
//   - scroll-обработчик throttled через requestAnimationFrame.
// ------------------------------------------------------------
const ROW_HEIGHT = 44
const OVERSCAN = 6

const scrollEl = ref(null)
const scrollTop = ref(0)
const viewportHeight = ref(0)
let rafId = null
let resizeObs = null

function onScroll(e) {
  if (rafId) return
  rafId = requestAnimationFrame(() => {
    scrollTop.value = e.target.scrollTop
    rafId = null
  })
}

function measureViewport() {
  if (scrollEl.value) {
    viewportHeight.value = scrollEl.value.clientHeight
  }
}

onMounted(() => {
  measureViewport()
  if (typeof ResizeObserver !== 'undefined' && scrollEl.value) {
    resizeObs = new ResizeObserver(measureViewport)
    resizeObs.observe(scrollEl.value)
  }
})

onBeforeUnmount(() => {
  if (resizeObs) resizeObs.disconnect()
  if (rafId) cancelAnimationFrame(rafId)
  if (searchTimer) clearTimeout(searchTimer)
})

// Сбрасываем скролл при изменении фильтра/сортировки/таймзоны.
watch(
  [() => debouncedSearch.value, sortKey, sortDir, () => props.timezone, () => props.samples],
  () => {
    if (scrollEl.value) {
      scrollEl.value.scrollTop = 0
      scrollTop.value = 0
    }
  },
)

const windowRange = computed(() => {
  const total = filtered.value.length
  if (total === 0) return { start: 0, end: 0 }
  const vh = viewportHeight.value || 400
  const start = Math.max(0, Math.floor(scrollTop.value / ROW_HEIGHT) - OVERSCAN)
  const visible = Math.ceil(vh / ROW_HEIGHT) + OVERSCAN * 2
  const end = Math.min(total, start + visible)
  return { start, end }
})

const windowedRows = computed(() =>
  filtered.value.slice(windowRange.value.start, windowRange.value.end),
)

const padTop = computed(() => windowRange.value.start * ROW_HEIGHT)
const padBottom = computed(() =>
  Math.max(0, (filtered.value.length - windowRange.value.end) * ROW_HEIGHT),
)
</script>

<template>
  <article class="ui-card flex flex-col min-h-0 max-h-[32rem] overflow-hidden">
    <!-- Header -->
    <header class="px-4 py-3 border-b border-slate-200 dark:border-slate-700 bg-slate-50/80 dark:bg-slate-800/80">
      <div class="flex items-center justify-between gap-2 mb-1">
        <h3 class="text-sm font-semibold font-mono truncate" :title="obisCode">
          {{ obisCode }}
        </h3>
        <span class="text-[10px] ui-text-dim whitespace-nowrap tabular-nums">
          {{ samples.length }} знач.
        </span>
      </div>
      <div class="flex items-center justify-between gap-2 text-[10px] ui-text-dim font-mono">
        <span class="truncate">modem {{ modem }} · hex {{ decToHex(modem) }}</span>
        <button type="button" class="ui-btn-ghost-sm" @click="copyCsv">
          {{ copied ? '✓' : 'CSV' }}
        </button>
      </div>

      <!-- Stats: min / max / avg -->
      <div
        v-if="stats"
        class="mt-2 grid grid-cols-3 gap-1 text-[10px] ui-text-muted border-t border-slate-200 dark:border-slate-700/60 pt-2"
      >
        <div>
          <div class="uppercase tracking-wide text-[9px] ui-text-dim">min</div>
          <div class="tabular-nums truncate" :title="stats.min">{{ formatNumber(stats.min) }}</div>
        </div>
        <div>
          <div class="uppercase tracking-wide text-[9px] ui-text-dim">max</div>
          <div class="tabular-nums truncate" :title="stats.max">{{ formatNumber(stats.max) }}</div>
        </div>
        <div>
          <div class="uppercase tracking-wide text-[9px] ui-text-dim">avg</div>
          <div class="tabular-nums truncate" :title="stats.avg">{{ formatNumber(stats.avg) }}</div>
        </div>
      </div>
    </header>

    <!-- Controls: search + sort -->
    <div class="px-3 pt-2 pb-2 border-b border-slate-200 dark:border-slate-700/50 space-y-2">
      <div class="relative">
        <svg
          class="absolute left-2 top-1/2 -translate-y-1/2 w-3 h-3 ui-text-dim"
          viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2"
        >
          <path stroke-linecap="round" stroke-linejoin="round" d="m19 19-4-4m0-7A7 7 0 1 1 1 8a7 7 0 0 1 14 0Z"/>
        </svg>
        <input
          v-model="search"
          type="search"
          placeholder="Поиск..."
          class="ui-input pl-7 pr-2 py-1 text-xs"
        />
      </div>
      <div class="flex items-center gap-1 text-[10px]">
        <span class="ui-text-dim mr-1">Сортировка:</span>
        <button
          type="button"
          class="px-2 py-0.5 rounded border transition"
          :class="sortKey === 'time'
            ? 'bg-blue-100 dark:bg-blue-600/30 border-blue-400 dark:border-blue-600/60 text-blue-700 dark:text-blue-200'
            : 'ui-btn-ghost-sm'"
          @click="cycleSort('time')"
        >
          Время {{ sortBadge('time') }}
        </button>
        <button
          type="button"
          class="px-2 py-0.5 rounded border transition"
          :class="sortKey === 'value'
            ? 'bg-blue-100 dark:bg-blue-600/30 border-blue-400 dark:border-blue-600/60 text-blue-700 dark:text-blue-200'
            : 'ui-btn-ghost-sm'"
          @click="cycleSort('value')"
        >
          Значение {{ sortBadge('value') }}
        </button>
      </div>
      <div v-if="latest" class="text-[10px] ui-text-dim">
        последнее:
        <span class="ui-text-muted tabular-nums">{{ latest.value }}</span>
        <span> · {{ humanTime(latest.time) }}</span>
      </div>
      <div v-if="filtered.length !== samples.length" class="text-[10px] ui-text-dim">
        отфильтровано: <b class="ui-text-muted">{{ filtered.length }}</b> из {{ samples.length }}
      </div>
    </div>

    <!-- Виртуализованный список -->
    <div
      ref="scrollEl"
      class="flex-1 overflow-y-auto overflow-x-hidden"
      @scroll.passive="onScroll"
    >
      <div v-if="filtered.length === 0" class="p-4 text-center text-xs ui-text-dim">
        Нет значений
      </div>
      <template v-else>
        <div :style="{ height: padTop + 'px' }" aria-hidden="true"></div>
        <ul class="divide-y divide-slate-200 dark:divide-slate-700/50">
          <li
            v-for="(s, i) in windowedRows"
            :key="windowRange.start + i"
            class="vr-row px-3 flex items-center justify-between gap-3 hover:bg-slate-100 dark:hover:bg-slate-700/30"
          >
            <div class="flex flex-col min-w-0 leading-tight">
              <span class="whitespace-nowrap text-xs" v-html="highlight(s._human)"></span>
              <span class="text-[10px] ui-text-dim font-mono" v-html="highlight(s.time)"></span>
            </div>
            <span
              class="text-xs font-mono tabular-nums whitespace-nowrap text-right"
              v-html="highlight(s.value)"
            ></span>
          </li>
        </ul>
        <div :style="{ height: padBottom + 'px' }" aria-hidden="true"></div>
      </template>
    </div>
  </article>
</template>

<style scoped>
/* Фиксированная высота строки — обязательна для корректной виртуализации.
   Должна совпадать с ROW_HEIGHT в JS (44px). */
.vr-row {
  height: 44px;
  box-sizing: border-box;
}
</style>
