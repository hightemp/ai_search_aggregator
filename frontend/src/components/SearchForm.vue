<template>
  <form @submit.prevent="submit" class="w-full max-w-xl bg-white p-4 rounded shadow">
    <div class="flex flex-col gap-2">
      <input v-model="prompt" placeholder="Search..." class="border rounded p-2" required />
      <div class="flex items-center gap-2">
        <label class="text-sm">Queries:</label>
        <input type="number" v-model.number="settings.queries" min="1" max="10" class="w-20 border rounded p-1" />
        <label class="flex items-center gap-1 text-sm">
          <input type="checkbox" v-model="settings.content_mode" />
          Full content
        </label>
        <label class="flex items-center gap-1 text-sm">
          <input type="checkbox" v-model="settings.ai_filter" />
          AI relevance filter
        </label>
        <button :disabled="store.loading" class="ml-auto px-4 py-2 bg-blue-600 text-white rounded disabled:opacity-50">
          {{ store.loading ? 'Searching…' : 'Search' }}
        </button>
      </div>
    </div>
  </form>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import axios from 'axios'
import type { SearchResult, SearchRequestSettings } from '../types'

import { useSearchStore } from '../stores/search'

const prompt = ref('')
const settings = ref<SearchRequestSettings>({ queries: 5, content_mode: false, ai_filter: false })
const store = useSearchStore()

async function submit() {
  await store.search(prompt.value, settings.value)
}
</script>
