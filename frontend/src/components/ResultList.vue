<template>
  <div class="w-full max-w-4xl">
    <!-- Empty state -->
    <div v-if="items.length === 0 && !showPlaceholder" class="text-center py-12">
      <svg class="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
      </svg>
      <h3 class="mt-2 text-lg font-medium text-gray-900">Результатов пока нет</h3>
      <p class="mt-1 text-gray-500">Введите поисковый запрос выше чтобы начать поиск</p>
    </div>

    <!-- Results -->
    <div v-else>
      <!-- Results header with metadata -->
      <div v-if="items.length > 0 || queries.length > 0" class="mb-6 p-4 bg-gray-50 rounded-lg">
        <div class="flex items-center justify-between flex-wrap gap-2">
          <div class="flex items-center gap-4 text-sm text-gray-600">
            <span v-if="items.length > 0">
              <strong>{{ items.length }}</strong> результатов найдено
            </span>
            <span v-if="queries.length > 0">
              <strong>{{ queries.length }}</strong> запросов выполнено
            </span>
          </div>
          <div class="flex gap-2">
            <button 
              @click="showQueries = !showQueries"
              class="text-sm text-blue-600 hover:text-blue-800 transition-colors"
            >
              {{ showQueries ? 'Скрыть' : 'Показать' }} запросы
            </button>
          </div>
        </div>
        
        <!-- Generated queries -->
        <div v-if="showQueries && queries.length > 0" class="mt-3 pt-3 border-t border-gray-200">
          <p class="text-sm font-medium text-gray-700 mb-2">Сгенерированные поисковые запросы:</p>
          <div class="flex flex-wrap gap-2">
            <span 
              v-for="(query, idx) in queries" 
              :key="idx"
              class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800"
            >
              {{ query }}
            </span>
          </div>
        </div>
      </div>

      <!-- Results list -->
      <div class="space-y-4">
        <ResultItem 
          v-for="(result, idx) in items" 
          :key="result.url" 
          :item="result" 
          :index="idx + 1"
          class="result-item"
        />
      </div>

      <!-- Load more placeholder (for future pagination) -->
      <div v-if="items.length >= 20" class="mt-8 text-center">
        <p class="text-gray-500 text-sm">
          Показаны первые {{ items.length }} результатов
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { SearchResult } from '../types'
import ResultItem from './ResultItem.vue'

interface Props {
  items: SearchResult[]
  queries?: string[]
  showPlaceholder?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  queries: () => [],
  showPlaceholder: false
})

const showQueries = ref(false)
</script>

<style scoped>
.result-item {
  opacity: 0;
  transform: translateY(10px);
  animation: fadeInUp 0.3s ease-out forwards;
}

.result-item:nth-child(1) { animation-delay: 0.05s; }
.result-item:nth-child(2) { animation-delay: 0.1s; }
.result-item:nth-child(3) { animation-delay: 0.15s; }
.result-item:nth-child(4) { animation-delay: 0.2s; }
.result-item:nth-child(5) { animation-delay: 0.25s; }

@keyframes fadeInUp {
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
