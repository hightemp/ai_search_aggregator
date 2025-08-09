# ---- Build stage ----
FROM golang:1.23-alpine AS builder

WORKDIR /app/backend

# Copy go.mod (and optionally go.sum) first for caching
COPY backend/go.mod ./
RUN go mod download

# Copy the rest of the source code
COPY backend/ ./

RUN go build -o server .

# ---- Runtime stage ----
FROM alpine:latest
WORKDIR /app

COPY --from=builder /app/backend/server ./server

EXPOSE 8080
ENV PORT=8080
CMD ["./server"]
