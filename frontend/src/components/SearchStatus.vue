<template>
  <div v-if="store.isLoading || store.searchStatus" class="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-4">
    <div class="flex items-center justify-between">
      <!-- Статус поиска -->
      <div class="flex items-center gap-3">
        <div class="relative">
          <svg class="animate-spin w-5 h-5 text-blue-600" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
        </div>
        
        <div class="flex-1">
          <div class="text-sm font-medium text-blue-900">
            {{ statusMessage }}
          </div>
          
          <!-- Прогресс-бар для WebSocket поиска -->
          <div v-if="store.searchStatus && store.searchStatus.total > 0" class="mt-2">
            <div class="flex justify-between text-xs text-blue-600 mb-1">
              <span>{{ store.searchStatus.progress }} / {{ store.searchStatus.total }}</span>
              <span>{{ Math.round((store.searchStatus.progress / store.searchStatus.total) * 100) }}%</span>
            </div>
            <div class="w-full bg-blue-200 rounded-full h-2">
              <div 
                class="bg-blue-600 h-2 rounded-full transition-all duration-300"
                :style="{ width: `${(store.searchStatus.progress / store.searchStatus.total) * 100}%` }"
              ></div>
            </div>
          </div>
        </div>
      </div>

      <!-- Таймер -->
      <div class="text-right">
        <div class="text-lg font-mono font-bold text-blue-900">
          {{ formattedElapsed }}
        </div>
        <div class="text-xs text-blue-600">
          время поиска
        </div>
      </div>
    </div>

    <!-- WebSocket статус -->
    <div class="mt-3 flex items-center gap-2 text-xs">
      <div class="flex items-center gap-1">
        <div 
          class="w-2 h-2 rounded-full"
          :class="store.isWebSocketConnected ? 'bg-green-500' : 'bg-red-500'"
        ></div>
        <span class="text-gray-600">
          WebSocket {{ store.isWebSocketConnected ? 'подключен' : 'отключен' }}
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { useSearchStore } from '../stores/search'

const store = useSearchStore()

// Таймер для обновления времени
let timer: number | null = null

const statusMessage = computed(() => {
  if (store.searchStatus) {
    return store.searchStatus.message
  }
  return 'Выполняется поиск...'
})

const formattedElapsed = computed(() => {
  const elapsed = store.currentElapsed
  if (elapsed < 1000) return `${elapsed}мс`
  if (elapsed < 60000) return `${(elapsed / 1000).toFixed(1)}с`
  const minutes = Math.floor(elapsed / 60000)
  const seconds = Math.floor((elapsed % 60000) / 1000)
  return `${minutes}:${seconds.toString().padStart(2, '0')}`
})

onMounted(() => {
  // Обновляем таймер каждые 100мс для плавного отображения
  timer = window.setInterval(() => {
    if (store.isLoading) {
      store.updateSearchElapsed()
    }
  }, 100)
})

onUnmounted(() => {
  if (timer) {
    clearInterval(timer)
  }
})
</script>