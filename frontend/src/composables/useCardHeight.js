import { computed, ref, watch } from 'vue'

const STORAGE_KEY = 'parse-api-messages:card-height'

// Пресеты высоты карточки (overall max-height контейнера metric-card в px).
// `normal` совпадает с историческим хардкодом max-h-[32rem] = 512px.
const PRESETS = Object.freeze({
  compact: 384, // 24rem
  normal: 512, // 32rem (default)
  tall: 768, // 48rem
  xl: 1024, // 64rem
})

const ORDER = ['compact', 'normal', 'tall', 'xl']

const LABELS = Object.freeze({
  compact: 'S',
  normal: 'M',
  tall: 'L',
  xl: 'XL',
})

const TITLES = Object.freeze({
  compact: 'Компактная (384px)',
  normal: 'Обычная (512px)',
  tall: 'Высокая (768px)',
  xl: 'Очень высокая (1024px)',
})

function loadInitial() {
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved && PRESETS[saved] !== undefined) return saved
  } catch (_) {}
  return 'normal'
}

const height = ref(loadInitial())

watch(height, (next, prev) => {
  const px = PRESETS[next] ?? PRESETS.normal
  console.debug('[useCardHeight] preset changed: %s -> %s (%dpx)', prev, next, px)
  try {
    localStorage.setItem(STORAGE_KEY, next)
  } catch (_) {}
})

const heightPx = computed(() => PRESETS[height.value] ?? PRESETS.normal)

export function useCardHeight() {
  return {
    height,
    heightPx,
    presets: PRESETS,
    order: ORDER,
    labels: LABELS,
    titles: TITLES,
    setHeight(name) {
      if (PRESETS[name] === undefined) return
      height.value = name
    },
    cycle() {
      const idx = ORDER.indexOf(height.value)
      const next = ORDER[(idx + 1) % ORDER.length]
      height.value = next
    },
  }
}
