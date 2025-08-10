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
        'INVALID_REQUEST': 'Неверный формат запроса. Пожалуйста, проверьте введенные данные.',
        'VALIDATION_FAILED': 'Ошибка валидации данных. Проверьте правильность заполнения полей.',
        'MISSING_API_KEY': 'Ошибка конфигурации сервера. Обратитесь к администратору.',
        'QUERY_GENERATION_FAILED': 'Не удалось сгенерировать поисковые запросы. Попробуйте еще раз.',
        'SEARCH_FAILED': 'Ошибка поиска. Проверьте подключение к интернету и повторите попытку.',
        'CONTENT_FETCH_FAILED': 'Не удалось загрузить содержимое страниц.',
        'RESPONSE_ENCODING_FAILED': 'Ошибка обработки ответа сервера.',
        'INTERNAL_ERROR': 'Внутренняя ошибка сервера. Попробуйте позже.',
        'CONNECTION_FAILED': 'Ошибка подключения к серверу.',
        'CONNECTION_LOST': 'Соединение потеряно во время поиска.',
        'TIMEOUT': 'Превышено время ожидания ответа.',
        'NETWORK_ERROR': 'Ошибка сети. Проверьте подключение к интернету.',
      }
      
      return errorMap[state.error.code] || `Неизвестная ошибка: ${state.error.message}`
    },
    // Новые getters для WebSocket и таймера
    formattedElapsed: (state): string => {
      const elapsed = state.searchElapsed
      if (elapsed < 1000) return `${elapsed}мс`
      if (elapsed < 60000) return `${(elapsed / 1000).toFixed(1)}с`
      return `${(elapsed / 60000).toFixed(1)}мин`
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
                message: 'Соединение потеряно во время поиска',
                details: 'Попробуйте повторить поиск'
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
          message: 'Неизвестная ошибка',
          details: e?.message || String(e),
        }
      }
    },
  },
})