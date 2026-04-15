<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import SelectTimezones from './components/select-timezones.vue'
import MetricCard from './components/metric-card.vue'
import AuthPanel from './components/auth-panel.vue'
import ErrorBanner from './components/error-banner.vue'
import { useAuth } from './composables/useAuth.js'
import { useMessages } from './composables/useMessages.js'
import { useTheme } from './composables/useTheme.js'

const form = reactive({
  urls: '',
  timezone: 'Etc/GMT-3', // UTC+3
})

const auth = useAuth()
const msgs = useMessages()
const { theme, toggle: toggleTheme } = useTheme()

// Auth-панель раскрыта автоматически если JWT не валиден.
const authPanelForcedOpen = ref(false)
const authPanelOpen = computed(() => {
  if (authPanelForcedOpen.value) return true
  return auth.status.value !== 'valid'
})

function toggleAuthPanel() {
  authPanelForcedOpen.value = !authPanelForcedOpen.value
}

onMounted(() => {
  auth.refresh()
})

watch(
  () => auth.status.value,
  (s) => {
    if (s === 'valid') authPanelForcedOpen.value = false
  },
)

async function apply() {
  if (auth.status.value !== 'valid') {
    await auth.refresh()
  }
  await msgs.load(form.urls)
}

const statusLabel = computed(() => {
  switch (auth.status.value) {
    case 'valid': return 'JWT валиден'
    case 'expired': return 'JWT просрочен'
    default: return 'JWT отсутствует'
  }
})

const statusDot = computed(() => {
  switch (auth.status.value) {
    case 'valid': return 'bg-emerald-500'
    case 'expired': return 'bg-amber-500'
    default: return 'bg-red-500'
  }
})

// Группируем карточки по модему.
const modemGroups = computed(() => {
  const groups = []
  for (const modem of Object.keys(msgs.messages)) {
    const codes = msgs.messages[modem] || {}
    const cards = []
    let samplesTotal = 0
    for (const code of Object.keys(codes)) {
      const samples = codes[code] || []
      if (samples.length === 0) continue
      cards.push({ key: `${modem}:${code}`, modem, obisCode: code, samples })
      samplesTotal += samples.length
    }
    if (cards.length === 0) continue
    groups.push({
      modem,
      hex: Number(modem).toString(16),
      cards,
      samplesTotal,
    })
  }
  return groups
})

const modemCount = computed(() => modemGroups.value.length)
const totalCards = computed(() => modemGroups.value.reduce((n, g) => n + g.cards.length, 0))
const totalSamples = computed(() => modemGroups.value.reduce((n, g) => n + g.samplesTotal, 0))

function prettyTz(tz) {
  if (tz === 'UTC') return 'UTC±0'
  return tz
    .replace('Etc/GMT-', 'UTC+')
    .replace('Etc/GMT+', 'UTC−')
}

const exportedAll = ref(false)

function humanTimeIn(unix, tz) {
  try {
    return new Date(unix * 1000).toLocaleString('ru-RU', { timeZone: tz })
  } catch (_) {
    return new Date(unix * 1000).toLocaleString('ru-RU')
  }
}

function csvEscape(v) {
  const s = String(v)
  if (s.includes(',') || s.includes('"') || s.includes('\n')) {
    return `"${s.replace(/"/g, '""')}"`
  }
  return s
}

async function exportAllCsv() {
  const rows = ['modem,obis_code,time_unix,time_human,value']
  for (const g of modemGroups.value) {
    for (const c of g.cards) {
      for (const s of c.samples) {
        rows.push(
          [c.modem, c.obisCode, s.time, humanTimeIn(s.time, form.timezone), s.value]
            .map(csvEscape)
            .join(','),
        )
      }
    }
  }
  try {
    await navigator.clipboard.writeText(rows.join('\n'))
    exportedAll.value = true
    setTimeout(() => (exportedAll.value = false), 2000)
  } catch (_) {}
}
</script>

<template>
  <div class="min-h-screen">
    <!-- Top bar -->
    <header class="ui-topbar">
      <div class="px-6 py-3 flex items-center justify-between gap-4">
        <div class="flex items-baseline gap-2">
          <h1 class="text-lg font-semibold">Parse API Messages</h1>
          <span class="text-xs ui-text-dim">WAVIOT</span>
        </div>
        <div class="flex items-center gap-2">
          <!-- Переключатель темы -->
          <button
            type="button"
            class="ui-badge !rounded-full !p-1.5 !gap-0"
            :title="theme === 'dark' ? 'Переключить на светлую тему' : 'Переключить на тёмную тему'"
            @click="toggleTheme"
          >
            <svg v-if="theme === 'dark'" class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="4"/>
              <path stroke-linecap="round" d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41"/>
            </svg>
            <svg v-else class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>
            </svg>
          </button>

          <!-- JWT бейдж -->
          <button type="button" class="ui-badge" @click="toggleAuthPanel">
            <span class="w-2 h-2 rounded-full" :class="statusDot"></span>
            <span>{{ statusLabel }}</span>
            <svg
              class="w-3 h-3 ui-text-dim transition-transform"
              :class="{ 'rotate-180': authPanelOpen }"
              viewBox="0 0 20 20" fill="currentColor"
            >
              <path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.06l3.71-3.83a.75.75 0 011.08 1.04l-4.25 4.39a.75.75 0 01-1.08 0L5.21 8.27a.75.75 0 01.02-1.06z" clip-rule="evenodd" />
            </svg>
          </button>
        </div>
      </div>
    </header>

    <!-- Main content -->
    <main class="px-6 py-6 space-y-6">
      <transition
        enter-active-class="transition duration-150 ease-out"
        leave-active-class="transition duration-100 ease-in"
        enter-from-class="opacity-0 -translate-y-1"
        leave-to-class="opacity-0 -translate-y-1"
      >
        <auth-panel
          v-if="authPanelOpen"
          :status="auth.status.value"
          :expires-at="auth.expiresAt.value"
          :login-url="auth.loginUrl.value"
          :stored-path="auth.storedPath.value"
          :loading="auth.loading.value"
          :capture="auth.capture.value"
          :capture-busy="auth.captureBusy.value"
          @open-login="auth.openLogin()"
          @save-jwt="auth.saveJwt($event)"
          @clear-jwt="auth.clear()"
          @start-capture="auth.startCapture()"
          @stop-capture="auth.stopCapture()"
        />
      </transition>

      <error-banner :message="auth.error.value" />
      <error-banner
        :message="msgs.error.value"
        :tone="msgs.unauthorized.value ? 'warn' : 'error'"
      />

      <!-- Форма запроса (компактная) -->
      <section class="ui-card-soft px-4 py-3">
        <div class="flex items-start gap-3">
          <div class="w-[140px] shrink-0">
            <label for="input-timezone" class="ui-label block mb-1">Часовая зона</label>
            <select-timezones id="input-timezone" v-model="form.timezone"></select-timezones>
          </div>
          <div class="flex-1 min-w-0">
            <label for="input-urls" class="ui-label block mb-1">Список URL (по одному на строку)</label>
            <textarea
              v-model="form.urls"
              id="input-urls"
              rows="2"
              class="ui-input font-mono p-2 text-xs resize-y"
              placeholder="https://api.waviot.ru/api/eav?modem_id=...&obis_code=value"
            ></textarea>
          </div>
          <div class="shrink-0 self-stretch flex flex-col justify-end">
            <button
              type="button"
              @click="apply"
              :disabled="msgs.loading.value || !form.urls.trim()"
              class="ui-btn-primary"
            >
              {{ msgs.loading.value ? 'Получаем...' : 'Применить' }}
            </button>
          </div>
        </div>
        <div v-if="modemCount > 0 || auth.status.value !== 'valid'" class="mt-2 flex items-center gap-3 text-[11px]">
          <span v-if="modemCount > 0" class="ui-text-muted">
            Получено модемов: <b>{{ modemCount }}</b>
            · карточек: <b>{{ totalCards }}</b>
            · значений: <b>{{ totalSamples }}</b>
          </span>
          <span v-if="auth.status.value !== 'valid'" class="text-amber-600 dark:text-amber-400">
            Сначала получите JWT
          </span>
        </div>
      </section>

      <!-- Результаты, сгруппированные по модему -->
      <section v-if="modemGroups.length > 0">
        <div class="flex items-baseline justify-between mb-3 flex-wrap gap-2">
          <h2 class="text-sm font-semibold flex items-baseline gap-2">
            <span>Результаты</span>
            <span class="text-xs ui-text-dim font-normal">
              · {{ modemCount }} модем. · {{ totalCards }} карточ. · {{ totalSamples }} знач. · {{ prettyTz(form.timezone) }}
            </span>
          </h2>
          <button type="button" class="ui-btn-ghost !py-1 !text-xs" @click="exportAllCsv">
            {{ exportedAll ? '✓ Скопировано' : 'Экспорт всех в CSV' }}
          </button>
        </div>

        <div class="space-y-6">
          <div
            v-for="group in modemGroups"
            :key="group.modem"
          >
            <div class="ui-topbar sticky top-14 -mx-1 px-1 py-2 mb-3 !border-b flex items-baseline gap-3 flex-wrap">
              <div class="flex items-baseline gap-2">
                <span class="ui-label">modem</span>
                <span class="text-base font-semibold font-mono tabular-nums">{{ group.modem }}</span>
                <span class="text-[10px] ui-text-dim font-mono">· hex {{ group.hex }}</span>
              </div>
              <span class="text-[11px] ui-text-dim">
                {{ group.cards.length }} карточ. · {{ group.samplesTotal }} знач.
              </span>
            </div>

            <div class="grid gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5">
              <metric-card
                v-for="card in group.cards"
                :key="card.key"
                :modem="card.modem"
                :obis-code="card.obisCode"
                :samples="card.samples"
                :timezone="form.timezone"
              />
            </div>
          </div>
        </div>
      </section>
    </main>
  </div>
</template>
