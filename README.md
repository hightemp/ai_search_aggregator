# AI Search Aggregator

A lightweight web application that enhances traditional web search by using AI to generate multiple focused queries and execute them in parallel via SearxNG and filtering with AI.

See `deploy/env.example`

![](screenshots/2025-08-09_14-17.png)

## Tech Stack

- **Backend**: Go 1.23, Chi router, Gorilla WebSocket, structured logging
- **Frontend**: Vue 3, TypeScript, Tailwind CSS, Pinia, WebSocket API
- **Search**: SearxNG (self-hosted)
- **AI**: OpenRouter API
- **Deploy**: Docker Compose with WebSocket-enabled Nginx

## License

MIT