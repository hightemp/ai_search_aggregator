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
