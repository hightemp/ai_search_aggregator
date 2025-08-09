# AI Search Aggregator

A lightweight web application that enhances traditional web search by using AI to generate multiple focused queries and execute them in parallel via SearxNG and filtering with AI.

See `deploy/env.example`

![](screenshots/2025-08-09_14-17.png)

## Tech Stack

- **Backend**: Go 1.23, Chi router, structured logging
- **Frontend**: Vue 3, TypeScript, Tailwind CSS, Pinia
- **Search**: SearxNG (self-hosted)
- **AI**: OpenRouter API (ChatGPT-4o)
- **Deploy**: Docker Compose

## License

MIT