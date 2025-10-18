<template>
  <div class="min-h-screen bg-gray-50">
    <!-- Header -->
    <header class="bg-white border-b border-gray-200 sticky top-0 z-50">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex justify-between items-center py-4">
          <div class="flex items-center gap-3">
            <div class="w-8 h-8 bg-blue-600 rounded-lg flex items-center justify-center">
              <svg class="w-5 h-5 text-white" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M8 4a4 4 0 100 8 4 4 0 000-8zM2 8a6 6 0 1110.89 3.476l4.817 4.817a1 1 0 01-1.414 1.414l-4.816-4.816A6 6 0 012 8z" clip-rule="evenodd" />
              </svg>
            </div>
            <div>
              <h1 class="text-xl font-bold text-gray-900">AI Search Aggregator</h1>
              <p class="text-sm text-gray-500 hidden sm:block">Intelligent search with multiple queries</p>
            </div>
          </div>
          
          <!-- Status indicator -->
          <div class="flex items-center gap-2">
            <div class="flex items-center gap-2 text-sm">
              <div 
                class="w-2 h-2 rounded-full"
                :class="{
                  'bg-green-500': store.loadingState === 'success',
                  'bg-yellow-500 animate-pulse': store.loadingState === 'loading',
                  'bg-red-500': store.loadingState === 'error',
                  'bg-gray-400': store.loadingState === 'idle'
                }"
              ></div>
              <span class="text-gray-600 hidden sm:inline">
                {{ statusText }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </header>

    <!-- Main content -->
    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div class="flex flex-col lg:flex-row gap-8">
        <!-- Search sidebar -->
        <aside class="lg:w-96 flex-shrink-0">
          <div class="sticky top-24">
            <SearchForm />
            
            <!-- Quick tips -->
            <div v-if="store.loadingState === 'idle'" class="mt-6 p-4 bg-blue-50 border border-blue-200 rounded-lg">
              <h3 class="font-medium text-blue-900 mb-2">💡 Search tips:</h3>
              <ul class="text-sm text-blue-800 space-y-1">
                <li>• Be specific with your questions</li>
                <li>• Use keywords</li>
                <li>• Enable content analysis for better results</li>
                <li>• Use Ctrl+Enter to search quickly</li>
              </ul>
            </div>
          </div>
        </aside>

        <!-- Results area -->
        <div class="flex-1 min-w-0">
          <!-- Search Status -->
          <SearchStatus />

          <!-- Results -->
          <ResultList 
            :items="store.results" 
            :queries="store.queries"
            :show-placeholder="store.loadingState === 'idle' && !store.isLoading"
          />
        </div>
      </div>
    </main>

    <!-- Footer -->
    <footer class="bg-white border-t border-gray-200 mt-16">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
        <div class="flex justify-between items-center text-sm text-gray-600">
          <p>&copy; 2025 AI Search Aggregator. Built with Vue 3 and Go.</p>
          <div class="flex gap-4">
            <a href="#" class="hover:text-gray-900 transition-colors">About</a>
            <a href="#" class="hover:text-gray-900 transition-colors">Help</a>
          </div>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import SearchForm from './components/SearchForm.vue'
import SearchStatus from './components/SearchStatus.vue'
import ResultList from './components/ResultList.vue'
import { useSearchStore } from './stores/search'

const store = useSearchStore()

const statusText = computed(() => {
  switch (store.loadingState) {
    case 'loading': return 'Searching...'
    case 'success': {
      const time = store.formattedElapsed
      return `${store.results.length} results (${time})`
    }
    case 'error': return 'Error'
    default: return 'Ready to search'
  }
})
</script>

<style>
/* Убираем стандартные стили для лучшего контроля */
* {
  box-sizing: border-box;
}

html {
  scroll-behavior: smooth;
}

/* Улучшенные стили для focus */
button:focus,
input:focus,
select:focus,
textarea:focus {
  outline: none;
}

/* Анимации */
@media (prefers-reduced-motion: no-preference) {
  .transition-all {
    transition: all 0.2s ease-in-out;
  }
  
  .transition-colors {
    transition: color 0.2s ease-in-out, background-color 0.2s ease-in-out, border-color 0.2s ease-in-out;
  }
  
  .transition-opacity {
    transition: opacity 0.2s ease-in-out;
  }
  
  .transition-shadow {
    transition: box-shadow 0.2s ease-in-out;
  }
}

/* Скрытие скроллбара но с сохранением прокрутки */
.scrollbar-hide {
  -ms-overflow-style: none;
  scrollbar-width: none;
}

.scrollbar-hide::-webkit-scrollbar {
  display: none;
}
</style>
