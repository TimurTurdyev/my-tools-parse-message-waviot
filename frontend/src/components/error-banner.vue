<script setup>
defineProps({
  message: { type: String, default: '' },
  tone: { type: String, default: 'error' }, // error | warn
})
</script>

<template>
  <div
    v-if="message"
    class="mb-4 px-4 py-3 rounded-lg text-sm border"
    :class="tone === 'warn'
      ? 'bg-amber-50 border-amber-300 text-amber-800 dark:bg-amber-900/30 dark:border-amber-700 dark:text-amber-200'
      : 'bg-red-50 border-red-300 text-red-800 dark:bg-red-900/30 dark:border-red-700 dark:text-red-200'"
  >
    {{ message }}
  </div>
</template>
