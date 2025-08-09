<template>
  <article class="bg-white p-6 rounded-lg shadow-sm border hover:shadow-md transition-shadow duration-200">
    <!-- Header with index and relevance score -->
    <div class="flex items-start justify-between mb-3">
      <div class="flex items-center gap-3">
        <span class="flex-shrink-0 w-6 h-6 bg-blue-100 text-blue-600 text-xs font-medium rounded-full flex items-center justify-center">
          {{ index }}
        </span>
        <div class="min-w-0 flex-1">
          <!-- Title and external link indicator -->
          <h2 class="text-lg font-medium text-gray-900 leading-snug">
            <a 
              :href="item.url" 
              class="text-blue-600 hover:text-blue-800 hover:underline focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 rounded"
              target="_blank" 
              rel="noopener noreferrer"
              @click="trackClick"
            >
              {{ displayTitle }}
              <svg class="inline w-3 h-3 ml-1 text-gray-400" fill="currentColor" viewBox="0 0 20 20" aria-hidden="true">
                <path fill-rule="evenodd" d="M4.25 5.5a.75.75 0 00-.75.75v8.5c0 .414.336.75.75.75h8.5a.75.75 0 00.75-.75v-4a.75.75 0 011.5 0v4A2.25 2.25 0 0112.75 17h-8.5A2.25 2.25 0 012 14.75v-8.5A2.25 2.25 0 014.25 4h5a.75.75 0 010 1.5h-5z" clip-rule="evenodd" />
                <path fill-rule="evenodd" d="M6.194 12.753a.75.75 0 001.06.053L16.5 4.44v2.81a.75.75 0 001.5 0v-4.5a.75.75 0 00-.75-.75h-4.5a.75.75 0 000 1.5h2.553l-9.056 8.194a.75.75 0 00-.053 1.06z" clip-rule="evenodd" />
              </svg>
            </a>
          </h2>
        </div>
      </div>
      
      <!-- Relevance score -->
      <div class="flex-shrink-0 ml-4">
        <div class="flex items-center gap-2">
          <div 
            class="px-2 py-1 rounded text-xs font-medium"
            :class="scoreColorClass"
          >
            {{ Math.round(item.score * 100) }}%
          </div>
        </div>
      </div>
    </div>

    <!-- URL with domain highlighting -->
    <div class="mb-3">
      <p class="text-sm text-gray-500 flex items-center gap-2 group">
        <span class="font-medium text-gray-600">{{ domain }}</span>
        <span class="text-gray-400">•</span>
        <span class="truncate" :title="item.url">{{ cleanPath }}</span>
        <button 
          @click="copyUrl"
          class="opacity-0 group-hover:opacity-100 text-gray-400 hover:text-gray-600 transition-opacity p-1"
          :class="{ 'text-green-600': urlCopied }"
          :title="urlCopied ? 'Скопировано!' : 'Копировать ссылку'"
        >
          <svg v-if="!urlCopied" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
          </svg>
          <svg v-else class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
            <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
          </svg>
        </button>
      </p>
    </div>

    <!-- Snippet with highlighting -->
    <div class="text-gray-700 leading-relaxed">
      <p v-if="item.snippet" class="text-sm">{{ item.snippet }}</p>
      <p v-else class="text-sm text-gray-400 italic">Нет описания</p>
    </div>

    <!-- Actions -->
    <div class="mt-4 flex items-center gap-4 text-sm">
      <button 
        @click="openInNewTab"
        class="flex items-center gap-1 text-blue-600 hover:text-blue-800 transition-colors"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
        </svg>
        Открыть
      </button>
      
      <button 
        @click="shareResult"
        class="flex items-center gap-1 text-gray-600 hover:text-gray-800 transition-colors"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8.684 13.342C8.886 12.938 9 12.482 9 12c0-.482-.114-.938-.316-1.342m0 2.684a3 3 0 110-2.684m0 2.684l6.632 3.316m-6.632-6l6.632-3.316m0 0a3 3 0 105.367-2.684 3 3 0 00-5.367 2.684zm0 9.316a3 3 0 105.367 2.684 3 3 0 00-5.367-2.684z" />
        </svg>
        Поделиться
      </button>
    </div>
  </article>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { SearchResult } from '../types'

interface Props {
  item: SearchResult
  index?: number
}

const props = withDefaults(defineProps<Props>(), {
  index: 1
})

const urlCopied = ref(false)

// Computed properties
const displayTitle = computed(() => {
  return props.item.title || 'Без названия'
})

const domain = computed(() => {
  try {
    const url = new URL(props.item.url)
    return url.hostname.replace(/^www\./, '')
  } catch {
    return props.item.url
  }
})

const cleanPath = computed(() => {
  try {
    const url = new URL(props.item.url)
    return url.pathname + url.search
  } catch {
    return props.item.url
  }
})

const scoreColorClass = computed(() => {
  const score = props.item.score
  if (score >= 0.8) return 'bg-green-100 text-green-800'
  if (score >= 0.6) return 'bg-yellow-100 text-yellow-800'
  if (score >= 0.4) return 'bg-orange-100 text-orange-800'
  return 'bg-red-100 text-red-800'
})

// Actions
function trackClick() {
  // Here you could add analytics tracking
  console.log('Result clicked:', props.item.url)
}

function openInNewTab() {
  window.open(props.item.url, '_blank', 'noopener,noreferrer')
  trackClick()
}

async function copyUrl() {
  try {
    await navigator.clipboard.writeText(props.item.url)
    urlCopied.value = true
    setTimeout(() => {
      urlCopied.value = false
    }, 2000)
  } catch (err) {
    console.error('Failed to copy URL:', err)
  }
}

async function shareResult() {
  if (navigator.share) {
    try {
      await navigator.share({
        title: props.item.title,
        url: props.item.url,
        text: props.item.snippet
      })
    } catch (err) {
      // User cancelled or share failed
      copyUrl() // Fallback to copying URL
    }
  } else {
    // Fallback for browsers without Web Share API
    copyUrl()
  }
}
</script>
