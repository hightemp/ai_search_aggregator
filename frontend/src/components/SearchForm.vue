<template>
  <div class="w-full max-w-2xl">
    <!-- Error notification -->
    <div v-if="store.hasError" class="mb-4 p-4 bg-red-50 border border-red-200 rounded-lg">
      <div class="flex items-start gap-3">
        <svg class="w-5 h-5 text-red-500 mt-0.5 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
          <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
        </svg>
        <div class="flex-1">
          <h3 class="text-sm font-medium text-red-800">{{ store.userFriendlyError }}</h3>
          <p v-if="store.error?.details" class="mt-1 text-sm text-red-600">{{ store.error.details }}</p>
          <div class="mt-3 flex gap-2">
            <button 
              v-if="store.canRetry" 
              @click="store.retrySearch()"
              :disabled="store.isLoading"
              class="text-sm bg-red-100 hover:bg-red-200 text-red-800 px-3 py-1 rounded disabled:opacity-50"
            >
              Повторить попытку ({{ store.retryCount }}/3)
            </button>
            <button 
              @click="store.clearError()"
              class="text-sm text-red-600 hover:text-red-800"
            >
              Закрыть
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Search form -->
    <form @submit.prevent="submit" class="bg-white p-6 rounded-lg shadow-sm border" :class="{ 'opacity-75': store.isLoading }">
      <div class="flex flex-col gap-4">
        <!-- Main search input -->
        <div class="relative">
          <label for="search-input" class="sr-only">Поисковый запрос</label>
          <input 
            id="search-input"
            v-model="prompt" 
            placeholder="Введите ваш поисковый запрос..." 
            class="w-full border rounded-lg p-3 pr-12 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-colors"
            :class="{ 'border-red-300': promptError }"
            required 
            maxlength="1000"
            :disabled="store.isLoading"
            @input="validatePrompt"
          />
          <div class="absolute right-3 top-3 text-gray-400">
            <span class="text-sm">{{ prompt.length }}/1000</span>
          </div>
          <p v-if="promptError" class="mt-1 text-sm text-red-600">{{ promptError }}</p>
        </div>

        <!-- Settings row -->
        <div class="flex items-center gap-4 flex-wrap">
          <!-- Queries count -->
          <div class="flex items-center gap-2">
            <label for="queries-input" class="text-sm font-medium text-gray-700">Запросов:</label>
            <input 
              id="queries-input"
              type="number" 
              v-model.number="settings.queries" 
              min="1" 
              max="20" 
              class="w-16 border rounded p-1 text-center focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
              :disabled="store.isLoading"
            />
          </div>

          <!-- Content mode toggle -->
          <label class="flex items-center gap-2 text-sm font-medium text-gray-700 cursor-pointer">
            <input 
              type="checkbox" 
              v-model="settings.content_mode"
              :disabled="store.isLoading"
              class="rounded text-blue-600 focus:ring-blue-500"
            />
            <span>Анализ содержимого страниц</span>
            <button 
              type="button"
              @click="showContentModeHelp = !showContentModeHelp"
              class="text-gray-400 hover:text-gray-600"
              aria-label="Справка по анализу содержимого"
            >
              <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-8-3a1 1 0 00-.867.5 1 1 0 11-1.731-1A3 3 0 0113 8a3.001 3.001 0 01-2 2.83V11a1 1 0 11-2 0v-1a1 1 0 011-1 1 1 0 100-2zm0 8a1 1 0 100-2 1 1 0 000 2z" clip-rule="evenodd" />
              </svg>
            </button>
          </label>

          <!-- Engines selector -->
          <div class="relative" ref="enginesMenuRef">
            <button 
              type="button" 
              class="flex items-center gap-2 border rounded-lg px-3 py-2 text-sm hover:bg-gray-50 focus:ring-1 focus:ring-blue-500 focus:border-blue-500 transition-colors"
              :disabled="store.isLoading"
              @click="toggleEngines"
            >
              <span>Поисковики</span>
              <span v-if="selectedEngines.length > 0" class="bg-blue-100 text-blue-800 text-xs px-2 py-0.5 rounded-full">
                {{ selectedEngines.length }}
              </span>
              <svg class="w-4 h-4 transition-transform" :class="{ 'rotate-180': openEngines }" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clip-rule="evenodd" />
              </svg>
            </button>
            
            <div v-if="openEngines" class="absolute z-10 mt-1 w-64 bg-white border rounded-lg shadow-lg p-3 max-h-64 overflow-auto">
              <div class="flex justify-between items-center mb-2">
                <span class="text-sm font-medium text-gray-700">Выберите поисковики:</span>
                <button 
                  type="button" 
                  @click="selectedEngines = []"
                  class="text-xs text-blue-600 hover:text-blue-800"
                >
                  Очистить
                </button>
              </div>
              <label v-for="eng in availableEngines" :key="eng" class="flex items-center gap-2 text-sm py-1 hover:bg-gray-50 rounded px-1 cursor-pointer">
                <input type="checkbox" :value="eng" v-model="selectedEngines" class="rounded text-blue-600" />
                <span class="capitalize">{{ eng }}</span>
              </label>
            </div>
          </div>

          <!-- Submit button -->
          <button 
            type="submit"
            :disabled="store.isLoading || !isFormValid"
            class="ml-auto px-6 py-2 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 transition-all"
          >
            <span v-if="store.isLoading" class="flex items-center gap-2">
              <svg class="animate-spin w-4 h-4" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
              Поиск...
            </span>
            <span v-else>Найти</span>
          </button>
        </div>

        <!-- Help text for content mode -->
        <div v-if="showContentModeHelp" class="p-3 bg-blue-50 border border-blue-200 rounded text-sm text-blue-800">
          <p><strong>Анализ содержимого страниц:</strong> Загружает содержимое найденных страниц и фильтрует результаты по релевантности с помощью ИИ. Повышает качество результатов, но увеличивает время поиска.</p>
        </div>
      </div>
    </form>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import type { SearchRequestSettings } from '../types'
import { useSearchStore } from '../stores/search'

const store = useSearchStore()

// Form state
const prompt = ref('')
const settings = ref<SearchRequestSettings>({ queries: 5, content_mode: false, engines: [] })
const promptError = ref('')
const showContentModeHelp = ref(false)

// Engines management
const openEngines = ref(false)
const enginesMenuRef = ref<HTMLElement | null>(null)
const availableEngines = ref<string[]>([
  'google', 'bing', 'duckduckgo', 'brave', 'qwant', 'yandex', 
  'wikipedia', 'github', 'stackoverflow', 'reddit', 'youtube'
])
const selectedEngines = ref<string[]>([])

// Computed properties
const isFormValid = computed(() => {
  return prompt.value.trim().length > 0 && 
         prompt.value.trim().length <= 1000 && 
         settings.value.queries >= 1 && 
         settings.value.queries <= 20 &&
         !promptError.value
})

// Form validation
function validatePrompt() {
  const trimmed = prompt.value.trim()
  if (trimmed.length === 0) {
    promptError.value = 'Поисковый запрос не может быть пустым'
  } else if (trimmed.length > 1000) {
    promptError.value = 'Поисковый запрос не может превышать 1000 символов'
  } else {
    promptError.value = ''
  }
}

// Engines management
function toggleEngines() {
  if (!openEngines.value) {
    selectedEngines.value = [...(settings.value.engines ?? [])]
  }
  openEngines.value = !openEngines.value
}

watch(selectedEngines, (val) => {
  settings.value.engines = [...val]
})

// Click outside handler for engines dropdown
function handleDocumentClick(e: MouseEvent) {
  if (!openEngines.value) return
  const el = enginesMenuRef.value
  if (!el) return
  const target = e.target as Node
  if (target && !el.contains(target)) {
    openEngines.value = false
  }
}

// Form submission
async function submit() {
  if (!isFormValid.value) return
  
  validatePrompt()
  if (promptError.value) return
  
  store.clearError()
  await store.search(prompt.value.trim(), settings.value)
}

// Keyboard shortcuts
function handleKeyDown(e: KeyboardEvent) {
  // Ctrl/Cmd + Enter to submit
  if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
    e.preventDefault()
    submit()
  }
  // Escape to close engines dropdown
  if (e.key === 'Escape' && openEngines.value) {
    openEngines.value = false
  }
}

// Lifecycle
onMounted(() => {
  document.addEventListener('click', handleDocumentClick)
  document.addEventListener('keydown', handleKeyDown)
  
  // Load saved settings from localStorage
  const saved = localStorage.getItem('search-settings')
  if (saved) {
    try {
      const parsed = JSON.parse(saved)
      settings.value = { ...settings.value, ...parsed }
      selectedEngines.value = [...(parsed.engines ?? [])]
    } catch (e) {
      // Ignore invalid saved settings
    }
  }
})

onBeforeUnmount(() => {
  document.removeEventListener('click', handleDocumentClick)
  document.removeEventListener('keydown', handleKeyDown)
})

// Save settings to localStorage when they change
watch(settings, (newSettings) => {
  localStorage.setItem('search-settings', JSON.stringify(newSettings))
}, { deep: true })

// Clear error when prompt changes
watch(prompt, () => {
  if (promptError.value) {
    validatePrompt()
  }
})
</script>
