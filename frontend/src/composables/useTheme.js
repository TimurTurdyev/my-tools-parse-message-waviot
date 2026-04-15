import { ref, watch } from 'vue'

const STORAGE_KEY = 'parse-api-messages:theme'

function loadInitial() {
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved === 'light' || saved === 'dark') return saved
  } catch (_) {}
  try {
    if (window.matchMedia?.('(prefers-color-scheme: dark)').matches) return 'dark'
  } catch (_) {}
  return 'dark' // для desktop-инструмента дефолт — тёмный
}

function apply(t) {
  const root = document.documentElement
  if (t === 'dark') root.classList.add('dark')
  else root.classList.remove('dark')
}

// Синглтон — одно общее состояние на всё приложение.
const theme = ref(loadInitial())
apply(theme.value)
watch(theme, (t) => {
  apply(t)
  try {
    localStorage.setItem(STORAGE_KEY, t)
  } catch (_) {}
})

export function useTheme() {
  return {
    theme,
    toggle() {
      theme.value = theme.value === 'dark' ? 'light' : 'dark'
    },
    setTheme(t) {
      theme.value = t === 'dark' ? 'dark' : 'light'
    },
  }
}
