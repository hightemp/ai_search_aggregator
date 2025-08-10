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



export type LoadingState = 'idle' | 'loading' | 'success' | 'error'

// WebSocket типы
export interface WSMessage {
  type: string
  data: any
}

export interface WSSearchStatus {
  stage: string
  progress: number
  total: number
  message: string
  timestamp: number
}

export interface WSSearchResult {
  queries: string[]
  results: SearchResult[]
  elapsed_ms: number
}
