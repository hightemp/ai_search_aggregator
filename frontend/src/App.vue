<template>
  <div class="min-h-screen flex flex-col items-center p-4 max-w-screen-md mx-auto">
    <h1 class="text-3xl font-bold mb-4 text-center">AI-powered Search</h1>
    <SearchForm />
    <div class="mt-6 w-full">
      <p v-if="store.loading" class="text-gray-500 text-center">Loading…</p>
      <p v-else-if="store.error" class="text-red-600 text-center">{{ store.error }}</p>
      <ResultList v-else :items="store.results" />
    </div>
  </div>
</template>

<script setup lang="ts">
import SearchForm from './components/SearchForm.vue'
import ResultList from './components/ResultList.vue'
import { useSearchStore } from './stores/search'

const store = useSearchStore()
</script>
