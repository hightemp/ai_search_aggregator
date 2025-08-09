import type { WSMessage, WSSearchStatus, WSSearchResult, AppError, SearchRequestSettings } from '../types'

export interface WebSocketSearchCallbacks {
  onStatus?: (status: WSSearchStatus) => void
  onResult?: (result: WSSearchResult) => void
  onError?: (error: AppError) => void
  onConnect?: () => void
  onDisconnect?: () => void
}

export class WebSocketSearchClient {
  private ws: WebSocket | null = null
  private reconnectTimer: number | null = null
  private reconnectAttempts = 0
  private maxReconnectAttempts = 5
  private callbacks: WebSocketSearchCallbacks = {}

  constructor(private baseUrl: string = '') {
    // Определяем WebSocket URL на основе текущего location
    if (!baseUrl) {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const host = window.location.host
      this.baseUrl = `${protocol}//${host}`
    }
  }

  connect(callbacks: WebSocketSearchCallbacks = {}): Promise<void> {
    this.callbacks = callbacks
    
    return new Promise((resolve, reject) => {
      try {
        const wsUrl = `${this.baseUrl}/api/ws/search`
        this.ws = new WebSocket(wsUrl)

        this.ws.onopen = () => {
          console.log('WebSocket connected')
          this.reconnectAttempts = 0
          this.callbacks.onConnect?.()
          resolve()
        }

        this.ws.onmessage = (event) => {
          try {
            const message: WSMessage = JSON.parse(event.data)
            this.handleMessage(message)
          } catch (error) {
            console.error('Failed to parse WebSocket message:', error)
          }
        }

        this.ws.onclose = (event) => {
          console.log('WebSocket disconnected:', event.code, event.reason)
          this.callbacks.onDisconnect?.()
          
          // Автоматическое переподключение для неожиданных отключений
          if (event.code !== 1000 && this.reconnectAttempts < this.maxReconnectAttempts) {
            this.scheduleReconnect()
          }
        }

        this.ws.onerror = (error) => {
          console.error('WebSocket error:', error)
          reject(new Error('WebSocket connection failed'))
        }

      } catch (error) {
        reject(error)
      }
    })
  }

  disconnect(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }

    if (this.ws) {
      this.ws.close(1000, 'Client disconnect')
      this.ws = null
    }
  }

  search(prompt: string, settings: SearchRequestSettings): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      throw new Error('WebSocket not connected')
    }

    const message: WSMessage = {
      type: 'search',
      data: {
        prompt,
        settings
      }
    }

    this.ws.send(JSON.stringify(message))
  }

  private handleMessage(message: WSMessage): void {
    switch (message.type) {
      case 'status':
        this.callbacks.onStatus?.(message.data as WSSearchStatus)
        break
      
      case 'search_complete':
        this.callbacks.onResult?.(message.data as WSSearchResult)
        break
      
      case 'error':
        this.callbacks.onError?.(message.data as AppError)
        break
      
      default:
        console.warn('Unknown message type:', message.type)
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
    }

    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000) // Exponential backoff, max 30s
    
    this.reconnectTimer = window.setTimeout(() => {
      this.reconnectAttempts++
      console.log(`Attempting to reconnect (${this.reconnectAttempts}/${this.maxReconnectAttempts})...`)
      
      this.connect(this.callbacks).catch(() => {
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
          this.scheduleReconnect()
        } else {
          console.error('Max reconnection attempts reached')
          this.callbacks.onError?.({
            code: 'CONNECTION_FAILED',
            message: 'Не удалось восстановить соединение',
            details: 'Превышено максимальное количество попыток переподключения'
          })
        }
      })
    }, delay)
  }

  get isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN
  }

  get connectionState(): string {
    if (!this.ws) return 'disconnected'
    
    switch (this.ws.readyState) {
      case WebSocket.CONNECTING: return 'connecting'
      case WebSocket.OPEN: return 'connected'
      case WebSocket.CLOSING: return 'closing'
      case WebSocket.CLOSED: return 'disconnected'
      default: return 'unknown'
    }
  }
}