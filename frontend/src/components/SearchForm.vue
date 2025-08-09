<template>
  <form @submit.prevent="submit" class="w-full max-w-xl bg-white p-4 rounded shadow">
    <div class="flex flex-col gap-2">
      <input v-model="prompt" placeholder="Search..." class="border rounded p-2" required />
      <div class="flex items-center gap-2 flex-wrap">
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
        <div class="relative" ref="enginesMenuRef">
          <button type="button" class="border rounded px-2 py-1 text-sm" @click="toggleEngines">
            Engines
          </button>
          <div v-if="openEngines" class="absolute z-10 mt-1 w-56 bg-white border rounded shadow p-2 max-h-64 overflow-auto">
            <label v-for="eng in availableEngines" :key="eng" class="flex items-center gap-2 text-sm py-0.5">
              <input type="checkbox" :value="eng" v-model="selectedEngines" />
              <span>{{ eng }}</span>
            </label>
            <div class="flex justify-end mt-2">
              <button type="button" class="text-xs px-2 py-1 border rounded" @click="applyEngines">Apply</button>
            </div>
          </div>
        </div>
        <button :disabled="store.loading" class="ml-auto px-4 py-2 bg-blue-600 text-white rounded disabled:opacity-50">
          {{ store.loading ? 'Searching…' : 'Search' }}
        </button>
      </div>
    </div>
  </form>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import type { SearchResult, SearchRequestSettings } from '../types'

import { useSearchStore } from '../stores/search'

const prompt = ref('')
const settings = ref<SearchRequestSettings>({ queries: 5, content_mode: false, ai_filter: false, engines: [] })
const store = useSearchStore()

const openEngines = ref(false)
const enginesMenuRef = ref<HTMLElement | null>(null)
const availableEngines = ref<string[]>([
  'google', 'bing', 'duckduckgo', 'brave', 'qwant', 'yandex', 'wikipedia', 'github', 'stackoverflow',
])
const selectedEngines = ref<string[]>([])

function toggleEngines() {
  if (!openEngines.value) {
    selectedEngines.value = [...(settings.value.engines ?? [])]
  }
  openEngines.value = !openEngines.value
}

function applyEngines() {
  settings.value.engines = [...selectedEngines.value]
  openEngines.value = false
}

function handleDocumentClick(e: MouseEvent) {
  if (!openEngines.value) return
  const el = enginesMenuRef.value
  if (!el) return
  const target = e.target as Node
  if (target && !el.contains(target)) {
    openEngines.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleDocumentClick)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', handleDocumentClick)
})

async function submit() {
  await store.search(prompt.value, settings.value)
}
</script>
