# ── Stage 1: build the Vue frontend ──────────────────────────────────────────
FROM node:22-alpine AS frontend-builder

WORKDIR /app/frontend

RUN npm install -g npm@11

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# ── Stage 2: build the Go backend ────────────────────────────────────────────
# CGO is required by go-sqlite3; Alpine provides musl-gcc via the build-base package.
FROM golang:1.25-alpine AS backend-builder

RUN apk add --no-cache build-base

WORKDIR /app/backend

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./

# Embed the pre-built frontend assets into the binary.
COPY --from=frontend-builder /app/frontend/dist ./internal/server/dist

RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /app/ratiodash ./cmd/api

# ── Stage 3: minimal runtime image ───────────────────────────────────────────
FROM alpine:3.21

RUN apk add --no-cache ca-certificates

WORKDIR /app

# Only the binary and YAML scraper definitions are needed — frontend assets are embedded inside it.
COPY --from=backend-builder /app/ratiodash ./ratiodash
COPY backend/scrapers ./scrapers

# Persist the SQLite database across restarts via a named volume
VOLUME ["/data"]

ENV SERVER_ADDR="0.0.0.0:8080" \
    DATABASE_URL="/data/ratiodash.db" \
    SCRAPERS_DIR="/app/scrapers"

EXPOSE 8080

ENTRYPOINT ["./ratiodash"]
