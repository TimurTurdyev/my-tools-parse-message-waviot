import { ref, onMounted, onBeforeUnmount } from 'vue'
import {
  fetchAuthStatus,
  openLoginInBrowser,
  setJwtManually,
  clearJwt,
  startJwtCapture,
  stopJwtCapture,
  onJwtReceived,
} from '../services/api.js'

export function useAuth() {
  const status = ref('missing')
  const expiresAt = ref(0)
  const loginUrl = ref('')
  const storedPath = ref('')
  const error = ref('')
  const loading = ref(false)

  const capture = ref(null) // {url, nonce, snippet} | null
  const captureBusy = ref(false)
  let offReceived = null

  async function refresh() {
    loading.value = true
    error.value = ''
    try {
      const r = await fetchAuthStatus()
      status.value = r.status
      expiresAt.value = r.expiresAt
      loginUrl.value = r.loginUrl
      storedPath.value = r.storedPath || ''
    } catch (e) {
      error.value = formatError(e)
    } finally {
      loading.value = false
    }
  }

  async function openLogin() {
    error.value = ''
    try {
      await openLoginInBrowser()
    } catch (e) {
      error.value = formatError(e)
    }
  }

  async function saveJwt(raw) {
    error.value = ''
    loading.value = true
    try {
      const r = await setJwtManually(raw)
      status.value = r.status
      expiresAt.value = r.expiresAt
      loginUrl.value = r.loginUrl
      storedPath.value = r.storedPath || ''
      return true
    } catch (e) {
      error.value = formatError(e)
      return false
    } finally {
      loading.value = false
    }
  }

  async function clear() {
    error.value = ''
    try {
      await clearJwt()
      await refresh()
    } catch (e) {
      error.value = formatError(e)
    }
  }

  async function startCapture() {
    error.value = ''
    captureBusy.value = true
    try {
      const s = await startJwtCapture()
      capture.value = s
    } catch (e) {
      error.value = formatError(e)
    } finally {
      captureBusy.value = false
    }
  }

  async function stopCapture() {
    try {
      await stopJwtCapture()
    } catch (_) {
      /* ignore */
    }
    capture.value = null
  }

  onMounted(() => {
    offReceived = onJwtReceived(() => {
      capture.value = null
      refresh()
    })
  })

  onBeforeUnmount(() => {
    if (offReceived) offReceived()
    stopCapture()
  })

  return {
    status, expiresAt, loginUrl, storedPath, error, loading,
    refresh, openLogin, saveJwt, clear,
    capture, captureBusy, startCapture, stopCapture,
  }
}

function formatError(e) {
  if (!e) return 'Неизвестная ошибка'
  if (typeof e === 'string') return e
  if (e.message) return e.message
  return String(e)
}
