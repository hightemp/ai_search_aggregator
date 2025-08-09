<template>
  <li class="bg-white p-4 rounded shadow">
    <a :href="item.url" class="text-blue-700 font-semibold hover:underline" target="_blank" rel="noopener">{{ item.title }}</a>
    <p class="text-sm text-gray-500">{{ item.url }}</p>
    <p class="mt-1 text-gray-700">{{ item.snippet }}</p>
  </li>
</template>

<script setup lang="ts">
import type { SearchResult } from '../types'

defineProps<{ item: SearchResult }>()
</script>
