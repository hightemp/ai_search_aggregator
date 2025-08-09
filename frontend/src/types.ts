export interface SearchResult {
  title: string
  url: string
  snippet: string
  score: number
}

export interface SearchRequestSettings {
  queries: number
  content_mode: boolean
  engines?: string[]
}

export interface AppError {
  code: string
  message: string
  details?: string
}

export interface ErrorResponse {
  error: AppError
}

export type LoadingState = 'idle' | 'loading' | 'success' | 'error'
