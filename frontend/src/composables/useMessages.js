import { reactive, ref } from 'vue'
import { fetchMessages } from '../services/api.js'
import { getObis } from '../func.js'

// Хранит «сырые» данные: { [modem]: { [obisCode]: [{time, value}, ...] } }.
// Форматирование humanTime делается на лету в table-modem-messages.vue
// на основе текущей таймзоны — смена TZ не требует повторного запроса.
export function useMessages() {
  const messages = reactive({})
  const error = ref('')
  const loading = ref(false)
  const unauthorized = ref(false)

  async function load(rawUrls) {
    error.value = ''
    unauthorized.value = false
    loading.value = true

    // Сброс старых данных
    for (const key of Object.keys(messages)) {
      delete messages[key]
    }

    const urls = rawUrls
      .split('\n')
      .map((u) => u.trim())
      .filter((u, i, self) => u !== '' && i === self.indexOf(u))

    try {
      for (const url of urls) {
        try {
          const response = await fetchMessages(url)
          const data = JSON.parse(response)
          if (!(data instanceof Object)) continue

          for (const dataKey in data) {
            const values = data[dataKey]
            const resultValues = {}

            if (values instanceof Array) {
              for (const value of values) {
                const code = getObis(value.obis_code)
                const time = value.timestamp

                if (!(code in resultValues)) resultValues[code] = []

                resultValues[code].push({
                  time,
                  value: value.value,
                })
              }
            }

            for (const code in resultValues) {
              resultValues[code].sort((a, b) => (a.time > b.time ? -1 : 1))
            }

            messages[dataKey] = resultValues
          }
        } catch (e) {
          const msg = formatError(e)
          if (msg.includes('unauthorized') || msg.includes('no token') || msg.includes('expired')) {
            unauthorized.value = true
            error.value = 'Нужен свежий JWT. Откройте страницу логина и вставьте новый токен.'
            return
          }
          error.value = msg
        }
      }
    } finally {
      loading.value = false
    }
  }

  return { messages, error, loading, unauthorized, load }
}

function formatError(e) {
  if (!e) return 'Неизвестная ошибка'
  if (typeof e === 'string') return e
  if (e.message) return e.message
  return String(e)
}
