# Tasks

This document breaks down the work needed to deliver the first minimal-viable product (MVP).
All items are phrased as GitHub issues; feel free to copy-paste.

## 0. Planning & Project Skeleton
- [ ] Decide on the public repository name
- [ ] Initialise git repository and commit this documentation
- [x] Create top-level directories:
  - `/backend`
  - `/frontend`
  - `/deploy` (Docker & CI)

## 1. Backend - Go
- [x] Bootstrap Go module (`go mod init ai-search-aggregator`)
- [x] Choose HTTP framework (standard `net/http` + chi OR Gin)
- [x] Implement configuration layer (env + defaults)
  - `OPENROUTER_API_KEY`
  - `SEARX_URL` (e.g. http://searx:8080)
  - `DEFAULT_QUERY_COUNT` (int)
  - `CONTENT_MODE_DEFAULT` (bool)
- [x] Implement OpenRouter client (ChatGPT-4o)
  - Function `GenerateQueries(prompt string, n int) ([]string, error)`
- [x] Implement Searx client
  - Function `Search(query string) ([]Result, error)`
- [x] Implement optional ContentFetcher (HTTP GET with timeout)
  - Respect in-progress search concurrency limit via `errgroup`
- [x] Aggregate & rank results
  - Merge duplicate URLs, keep highest score
  - Basic ranking: position weight + snippet/content similarity
- [x] HTTP Endpoints
  - `POST /api/search` – body { prompt, settings }
  - `GET  /healthz` – liveness probe
- [x] Write unit tests (OpenRouter mock, Searx stub)
- [x] Write integration test that spins up Searx container via Testcontainers

## 2. Frontend - Vue 3
- [x] Initialise project with Vite + TypeScript
- [x] Install TailwindCSS for styling
- [x] Components
- [x] State management with Pinia (or plain reactive state)
- [x] Display loader & error states
- [x] Mobile-first responsive layout
  - `SearchForm.vue` (prompt field + settings drawer)
  - `ResultList.vue`
  - `ResultItem.vue`
- [x] State management with Pinia (or plain reactive state)
- [x] Call backend `POST /api/search`
- [x] Display loader & error states
- [x] Mobile-first responsive layout

## 3. Deployment (Docker & Compose)
- [x] Dockerfile for backend (multi-stage)
- [x] Dockerfile for frontend (build then Nginx)
- [x] Add `searxng/searx` service in `docker-compose.yml`
- [x] Compose network & env-file
- [x] Optional: nginx reverse-proxy in front of all services

## 4. CI/CD (optional for MVP)
- [x] GitHub Actions workflow:
  - Lint & test Go
  - Lint Vue & run unit tests
  - Build Docker images

## 5. Documentation
- [ ] Update `README.md` with setup instructions
- [ ] Add OpenAPI spec (`deploy/openapi.yaml`)

## 6. Future Enhancements (Post-MVP)
- [ ] Persist search history in a database (SQLite/Postgres)
- [ ] User accounts & authentication
- [ ] Advanced ranking using embeddings
- [ ] Caching layer for Searx responses
- [ ] Internationalisation of the UI
