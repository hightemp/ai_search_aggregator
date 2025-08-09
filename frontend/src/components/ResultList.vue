<template>
  <div>
    <p v-if="items.length === 0" class="text-gray-500">No results yet.</p>
    <ul v-else class="space-y-4">
      <ResultItem v-for="r in items" :key="r.url" :item="r" />
    </ul>
  </div>
</template>

<script setup lang="ts">
import type { SearchResult } from '../types'
import ResultItem from './ResultItem.vue'

defineProps<{ items: SearchResult[] }>()
</script>
