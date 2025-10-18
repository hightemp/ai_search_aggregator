import { defineStore } from 'pinia'
import type { SearchResult, SearchRequestSettings, AppError, LoadingState, WSSearchStatus } from '../types'
import { WebSocketSearchClient } from '../services/websocket'

interface State {
  results: SearchResult[]
  queries: string[]
  loadingState: LoadingState
  error: AppError | null
  retryCount: number
  lastSearchParams: { prompt: string; settings: SearchRequestSettings } | null
  // WebSocket и таймер состояние
  searchStartTime: number | null
  searchElapsed: number
  searchStatus: WSSearchStatus | null
  wsClient: WebSocketSearchClient | null
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
    // WebSocket и таймер состояние
    searchStartTime: null,
    searchElapsed: 0,
    searchStatus: null,
    wsClient: null,
  }),
  getters: {
    isLoading: (state) => state.loadingState === 'loading',
    hasError: (state) => state.loadingState === 'error' && state.error !== null,
    canRetry: (state) => state.retryCount < MAX_RETRY_ATTEMPTS && state.lastSearchParams !== null,
    userFriendlyError: (state): string => {
      if (!state.error) return ''
      
      const errorMap: Record<string, string> = {
        'INVALID_REQUEST': 'Invalid request format. Please check your input.',
        'VALIDATION_FAILED': 'Validation error. Please verify the fields.',
        'MISSING_API_KEY': 'Server configuration error. Contact the administrator.',
        'QUERY_GENERATION_FAILED': 'Failed to generate search queries. Please try again.',
        'SEARCH_FAILED': 'Search error. Check your internet connection and try again.',
        'CONTENT_FETCH_FAILED': 'Failed to load page content.',
        'RESPONSE_ENCODING_FAILED': 'Server response processing error.',
        'INTERNAL_ERROR': 'Internal server error. Please try again later.',
        'CONNECTION_FAILED': 'Connection error to the server.',
        'CONNECTION_LOST': 'Connection lost during search.',
        'TIMEOUT': 'Response timeout exceeded.',
        'NETWORK_ERROR': 'Network error. Check your internet connection.',
      }
      
      return errorMap[state.error.code] || `Unknown error: ${state.error.message}`
    },
    // Новые getters для WebSocket и таймера
    formattedElapsed: (state): string => {
      const elapsed = state.searchElapsed
      if (elapsed < 1000) return `${elapsed}ms`
      if (elapsed < 60000) return `${(elapsed / 1000).toFixed(1)}s`
      return `${(elapsed / 60000).toFixed(1)}min`
    },
    currentElapsed(state): number {
      if (!state.searchStartTime) return state.searchElapsed
      return Date.now() - state.searchStartTime
    },
    isWebSocketConnected: (state) => state.wsClient?.isConnected ?? false,
  },
  actions: {
    async search(prompt: string, settings: SearchRequestSettings) {
      this.loadingState = 'loading'
      this.error = null
      this.lastSearchParams = { prompt, settings }
      this.searchStartTime = Date.now()
      this.searchElapsed = 0
      this.searchStatus = null
      
      try {
        await this.connectAndSearch(prompt, settings)
      } catch (e) {
        this.handleSearchError(e)
      }
    },

    async connectAndSearch(prompt: string, settings: SearchRequestSettings) {
      // Инициализируем WebSocket клиент если его нет
      if (!this.wsClient) {
        this.wsClient = new WebSocketSearchClient()
      }

      // Подключаемся если не подключены
      if (!this.wsClient.isConnected) {
        await this.wsClient.connect({
          onStatus: (status) => {
            this.searchStatus = status
          },
          onResult: (result) => {
            this.results = result.results
            this.queries = result.queries
            this.searchElapsed = result.elapsed_ms
            this.loadingState = 'success'
            this.retryCount = 0
            this.searchStatus = null // Очищаем статус после завершения
          },
          onError: (error) => {
            this.error = error
            this.loadingState = 'error'
            this.searchStatus = null // Очищаем статус при ошибке
          },
          onDisconnect: () => {
            // При отключении во время поиска считаем это ошибкой
            if (this.loadingState === 'loading') {
              this.error = {
                code: 'CONNECTION_LOST',
                message: 'Connection lost during search',
                details: 'Please try the search again'
              }
              this.loadingState = 'error'
              this.searchStatus = null // Очищаем статус при отключении
            }
          }
        })
      }

      // Запускаем поиск
      this.wsClient.search(prompt, settings)
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
      this.searchStartTime = null
      this.searchElapsed = 0
      this.searchStatus = null
    },

    // WebSocket management
    disconnectWebSocket() {
      if (this.wsClient) {
        this.wsClient.disconnect()
        this.wsClient = null
      }
    },

    updateSearchElapsed() {
      if (this.searchStartTime) {
        this.searchElapsed = Date.now() - this.searchStartTime
      }
    },

    handleSearchError(e: any) {
      this.loadingState = 'error'
      
      // Если ошибка уже в правильном формате (от WebSocket)
      if (e && typeof e === 'object' && e.code && e.message) {
        this.error = e as AppError
      } else {
        // Неизвестная ошибка
        this.error = {
          code: 'UNKNOWN_ERROR',
          message: 'Unknown error',
          details: e?.message || String(e),
        }
      }
    },
  },
})