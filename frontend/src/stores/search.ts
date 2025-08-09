import { defineStore } from 'pinia'
import axios from 'axios'
import type { SearchResult, SearchRequestSettings } from '../types'

interface State {
  results: SearchResult[]
  loading: boolean
  error: string | null
}

export const useSearchStore = defineStore('search', {
  state: (): State => ({
    results: [],
    loading: false,
    error: null,
  }),
  actions: {
    async search(prompt: string, settings: SearchRequestSettings) {
      this.loading = true
      this.error = null
      try {
        const res = await axios.post('/api/search', { prompt, settings })
        this.results = res.data.results as SearchResult[]
      } catch (e: any) {
        this.error = e?.message ?? 'unknown error'
      } finally {
        this.loading = false
      }
    },
  },
})