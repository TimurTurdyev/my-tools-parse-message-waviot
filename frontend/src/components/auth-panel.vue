<script setup>
import { ref } from 'vue'

const props = defineProps({
  status: { type: String, default: 'missing' },
  expiresAt: { type: Number, default: 0 },
  loginUrl: { type: String, default: '' },
  storedPath: { type: String, default: '' },
  loading: { type: Boolean, default: false },
  capture: { type: Object, default: null },
  captureBusy: { type: Boolean, default: false },
})

const emit = defineEmits([
  'open-login',
  'save-jwt',
  'clear-jwt',
  'start-capture',
  'stop-capture',
])

const showManual = ref(false)
const jwtInput = ref('')
const copied = ref('')

const manualSnippet = `(async () => {
  const m = document.cookie.match(/WAVIOT_JWT=([^;]+)/);
  if (!m) {
    alert("WAVIOT_JWT не найден в document.cookie (скорее всего HttpOnly).\\nОткройте DevTools > Application > Cookies > WAVIOT_JWT и скопируйте значение вручную.");
    return;
  }
  const jwt = decodeURIComponent(m[1]);
  try {
    await navigator.clipboard.writeText(jwt);
    alert("WAVIOT_JWT скопирован в буфер обмена (" + jwt.length + " символов). Вставьте его в Parse API Messages.");
  } catch (e) {
    console.log("WAVIOT_JWT:", jwt);
    prompt("WAVIOT_JWT (выделите и скопируйте):", jwt);
  }
})();`

function onSave() {
  const raw = jwtInput.value.trim()
  if (!raw) return
  emit('save-jwt', raw)
  jwtInput.value = ''
  showManual.value = false
}

async function copySnippet() {
  if (!props.capture?.snippet) return
  try {
    await navigator.clipboard.writeText(props.capture.snippet)
    copied.value = 'snippet'
    setTimeout(() => (copied.value = ''), 2000)
  } catch (_) {
    copied.value = ''
  }
}

async function copyManualSnippet() {
  try {
    await navigator.clipboard.writeText(manualSnippet)
    copied.value = 'manual'
    setTimeout(() => (copied.value = ''), 2000)
  } catch (_) {
    copied.value = ''
  }
}

function formatExp(unix) {
  if (!unix) return ''
  const d = new Date(unix * 1000)
  return d.toLocaleString('ru-RU')
}
</script>

<template>
  <section class="ui-card p-4 mb-4">
    <div class="flex items-center justify-between mb-3 flex-wrap gap-2">
      <div>
        <h2 class="text-base font-semibold">Авторизация WAVIOT</h2>
        <div class="text-sm mt-1">
          <span v-if="status === 'valid'" class="text-emerald-600 dark:text-emerald-400">
            ● JWT валиден<span v-if="expiresAt"> · истекает {{ formatExp(expiresAt) }}</span>
          </span>
          <span v-else-if="status === 'expired'" class="text-amber-600 dark:text-amber-400">
            ● JWT просрочен<span v-if="expiresAt"> ({{ formatExp(expiresAt) }})</span>
          </span>
          <span v-else class="text-red-600 dark:text-red-400">● JWT отсутствует</span>
        </div>
      </div>
      <div class="flex gap-2 flex-wrap">
        <button
          v-if="!capture"
          type="button"
          class="ui-btn-primary !py-1.5 !text-xs"
          :disabled="captureBusy || loading"
          @click="emit('start-capture')"
        >
          Автопередача JWT
        </button>
        <button
          v-else
          type="button"
          class="ui-btn-danger !py-1.5 !text-xs"
          @click="emit('stop-capture')"
        >
          Остановить приём
        </button>
        <button
          type="button"
          class="ui-btn-ghost !py-1.5 !text-xs"
          :disabled="loading"
          @click="showManual = !showManual"
        >
          {{ showManual ? 'Скрыть ручную' : 'Ручная вставка' }}
        </button>
        <button
          v-if="status !== 'missing'"
          type="button"
          class="ui-btn-ghost !py-1.5 !text-xs"
          @click="emit('clear-jwt')"
        >
          Очистить
        </button>
      </div>
    </div>

    <!-- Автопередача: активная сессия -->
    <div
      v-if="capture"
      class="mt-3 rounded-lg border p-3
             bg-blue-50 border-blue-200 text-blue-900
             dark:bg-blue-900/30 dark:border-blue-700 dark:text-blue-100"
    >
      <div class="text-xs mb-2">
        <p class="mb-1 font-semibold">Слушаю {{ capture.url.replace(/\?.*/, '') }} — токен придёт автоматически.</p>
        <ol class="list-decimal list-inside space-y-0.5">
          <li>
            Откройте в браузере нужный хост WAVIOT (например,
            <button type="button" class="underline text-blue-700 dark:text-blue-300" @click="emit('open-login')">lk.waviot.ru</button>)
            и залогиньтесь, если ещё не сделали этого.
          </li>
          <li>Откройте DevTools (F12), перейдите на вкладку <b>Console</b>.</li>
          <li>Нажмите «Скопировать снипет» ниже и вставьте его в Console, затем Enter.</li>
          <li>Приложение поймает токен и само обновит статус.</li>
        </ol>
      </div>
      <div class="flex items-center gap-2 mb-2">
        <button type="button" class="ui-btn-primary !py-1 !text-xs" @click="copySnippet">
          {{ copied === 'snippet' ? '✓ Скопировано' : 'Скопировать снипет' }}
        </button>
        <span class="text-[10px] font-mono break-all">{{ capture.url }}</span>
      </div>
      <pre class="ui-code-block max-h-48">{{ capture.snippet }}</pre>
    </div>

    <!-- Ручная вставка -->
    <div v-if="showManual" class="mt-3">
      <div class="text-xs ui-text-muted mb-2">
        <p class="mb-1">Как получить токен вручную:</p>
        <ol class="list-decimal list-inside space-y-0.5">
          <li>
            Нажмите
            <button
              type="button"
              class="underline text-blue-600 dark:text-blue-400"
              @click="emit('open-login')"
            >«Открыть lk.waviot.ru»</button> — откроется страница в браузере.
          </li>
          <li>Залогиньтесь под нужным аккаунтом.</li>
          <li>Откройте DevTools (F12) → вкладка <b>Console</b>.</li>
          <li>Вставьте снипет ниже и нажмите Enter — он скопирует <code>WAVIOT_JWT</code> в буфер обмена (или покажет prompt, если не получилось).</li>
          <li>Вернитесь в приложение и вставьте токен в поле ниже.</li>
        </ol>
        <p class="mt-2 italic">
          Если снипет скажет «не найден в document.cookie» — значит cookie <code>HttpOnly</code>.
          Тогда откройте DevTools → <b>Application</b> → <b>Cookies</b> → <code>WAVIOT_JWT</code> и скопируйте вручную.
        </p>
      </div>

      <div class="mb-3">
        <div class="flex items-center justify-between mb-1">
          <span class="text-[11px] font-semibold ui-text-muted">JS-снипет для консоли браузера:</span>
          <button type="button" class="ui-btn-primary !py-1 !text-xs" @click="copyManualSnippet">
            {{ copied === 'manual' ? '✓ Скопировано' : 'Скопировать снипет' }}
          </button>
        </div>
        <pre class="ui-code-block max-h-40">{{ manualSnippet }}</pre>
      </div>

      <textarea
        v-model="jwtInput"
        rows="3"
        placeholder="Вставьте сюда значение cookie WAVIOT_JWT..."
        class="ui-input font-mono p-2 text-xs"
      ></textarea>
      <div class="flex gap-2 mt-2">
        <button
          type="button"
          class="ui-btn-success !py-1.5 !text-xs"
          :disabled="!jwtInput.trim() || loading"
          @click="onSave"
        >
          Сохранить
        </button>
        <button type="button" class="ui-btn-ghost !py-1.5 !text-xs" @click="emit('open-login')">
          Открыть lk.waviot.ru
        </button>
      </div>
    </div>

    <p v-if="storedPath" class="mt-3 text-[10px] ui-text-dim font-mono break-all">
      Файл токена: {{ storedPath }}
    </p>
  </section>
</template>
