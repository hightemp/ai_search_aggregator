import { defineStore } from 'pinia'
import axios, { AxiosError } from 'axios'
import type { SearchResult, SearchRequestSettings, AppError, ErrorResponse, LoadingState } from '../types'

interface State {
  results: SearchResult[]
  queries: string[]
  loadingState: LoadingState
  error: AppError | null
  retryCount: number
  lastSearchParams: { prompt: string; settings: SearchRequestSettings } | null
}

const MAX_RETRY_ATTEMPTS = 3
const RETRY_DELAY_MS = 1000

export const useSearchStore = defineStore('search', {
  state: (): State => ({
    results: [],
    queries: [],
    loadingState: 'idle',
    error: null,
    retryCount: 0,
    lastSearchParams: null,
  }),
  getters: {
    isLoading: (state) => state.loadingState === 'loading',
    hasError: (state) => state.loadingState === 'error' && state.error !== null,
    canRetry: (state) => state.retryCount < MAX_RETRY_ATTEMPTS && state.lastSearchParams !== null,
    userFriendlyError: (state): string => {
      if (!state.error) return ''
      
      const errorMap: Record<string, string> = {
        'INVALID_REQUEST': 'Неверный формат запроса. Пожалуйста, проверьте введенные данные.',
        'VALIDATION_FAILED': 'Ошибка валидации данных. Проверьте правильность заполнения полей.',
        'MISSING_API_KEY': 'Ошибка конфигурации сервера. Обратитесь к администратору.',
        'QUERY_GENERATION_FAILED': 'Не удалось сгенерировать поисковые запросы. Попробуйте еще раз.',
        'SEARCH_FAILED': 'Ошибка поиска. Проверьте подключение к интернету и повторите попытку.',
        'CONTENT_FETCH_FAILED': 'Не удалось загрузить содержимое страниц.',
        'RESPONSE_ENCODING_FAILED': 'Ошибка обработки ответа сервера.',
        'INTERNAL_ERROR': 'Внутренняя ошибка сервера. Попробуйте позже.',
      }
      
      return errorMap[state.error.code] || `Неизвестная ошибка: ${state.error.message}`
    },
  },
  actions: {
    async search(prompt: string, settings: SearchRequestSettings) {
      this.loadingState = 'loading'
      this.error = null
      this.lastSearchParams = { prompt, settings }
      
      try {
        const response = await axios.post('/api/search', { prompt, settings }, {
          timeout: 120000, // 2 минуты
          headers: {
            'Content-Type': 'application/json',
          },
        })
        
        this.results = response.data.results as SearchResult[]
        this.queries = response.data.queries as string[]
        this.loadingState = 'success'
        this.retryCount = 0
        
      } catch (e) {
        this.handleSearchError(e)
      }
    },

    async retrySearch() {
      if (!this.canRetry || !this.lastSearchParams) return
      
      this.retryCount++
      
      // Добавляем задержку перед повторной попыткой
      await new Promise(resolve => setTimeout(resolve, RETRY_DELAY_MS * this.retryCount))
      
      return this.search(this.lastSearchParams.prompt, this.lastSearchParams.settings)
    },

    clearError() {
      this.error = null
      if (this.loadingState === 'error') {
        this.loadingState = 'idle'
      }
    },

    clearResults() {
      this.results = []
      this.queries = []
      this.error = null
      this.loadingState = 'idle'
      this.retryCount = 0
      this.lastSearchParams = null
    },

    handleSearchError(e: any) {
      this.loadingState = 'error'
      
      if (axios.isAxiosError(e)) {
        const axiosError = e as AxiosError<ErrorResponse>
        
        if (axiosError.response?.data?.error) {
          // Структурированная ошибка от API
          this.error = axiosError.response.data.error
        } else if (axiosError.code === 'ECONNABORTED') {
          // Тайм-аут
          this.error = {
            code: 'TIMEOUT',
            message: 'Превышено время ожидания ответа',
            details: 'Попробуйте уменьшить количество запросов или отключить режим анализа контента',
          }
        } else if (axiosError.code === 'ERR_NETWORK') {
          // Проблемы с сетью
          this.error = {
            code: 'NETWORK_ERROR',
            message: 'Ошибка сети',
            details: 'Проверьте подключение к интернету',
          }
        } else {
          // Другие HTTP ошибки
          this.error = {
            code: 'HTTP_ERROR',
            message: `Ошибка сервера (${axiosError.response?.status || 'неизвестный код'})`,
            details: axiosError.message,
          }
        }
      } else {
        // Неизвестная ошибка
        this.error = {
          code: 'UNKNOWN_ERROR',
          message: 'Неизвестная ошибка',
          details: e?.message || String(e),
        }
      }
    },
  },
})