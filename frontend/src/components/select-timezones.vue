<script setup>
import { computed } from 'vue'

defineProps({
  modelValue: String,
})

defineEmits(['update:modelValue'])

// Генерируем список UTC-12..UTC+14.
// ВАЖНО: в IANA `Etc/GMT` знак инвертирован (POSIX-стиль):
//   UTC+5  ↔ Etc/GMT-5
//   UTC-3  ↔ Etc/GMT+3
//   UTC+0  ↔ Etc/GMT  (он же UTC)
// Мы показываем пользователю привычный формат UTC±N, а в value
// отдаём валидный IANA, который понимает `toLocaleString({timeZone})`.
const options = computed(() => {
  const list = []
  for (let offset = 14; offset >= -12; offset--) {
    const label = offset === 0 ? 'UTC±0' : `UTC${offset > 0 ? '+' : '−'}${Math.abs(offset)}`
    const value = offset === 0 ? 'UTC' : `Etc/GMT${offset > 0 ? '-' : '+'}${Math.abs(offset)}`
    list.push({ label, value })
  }
  return list
})
</script>

<template>
  <select
    @change="$emit('update:modelValue', $event.target.value)"
    :value="modelValue"
    class="bg-gray-50 border border-gray-300 text-gray-600 text-sm rounded-lg focus:ring-blue-500 focus:border-blue-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-gray-300 dark:focus:ring-blue-500 dark:focus:border-blue-500"
  >
    <option v-for="opt in options" :key="opt.value" :value="opt.value">
      {{ opt.label }}
    </option>
  </select>
</template>
