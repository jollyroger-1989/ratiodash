.PHONY: dev-backend dev-frontend install tidy build-backend build-frontend test test-backend

## Install all dependencies
install:
	cd frontend && npm install

## Download and tidy Go modules
tidy:
	cd backend && go mod tidy

## Run the Go API server (requires Go + gcc for CGo/SQLite)
dev-backend:
	cd backend && go run ./cmd/api

## Run the Vite dev server
dev-frontend:
	cd frontend && npm run dev

## Build the Go binary
build-backend:
	cd backend && go build -o bin/api ./cmd/api

## Build the frontend, copy assets into the embed path, then build the binary
build: build-frontend
	cp -r frontend/dist/. backend/internal/server/dist/
	$(MAKE) build-backend

## Build the frontend for production
build-frontend:
	cd frontend && npm run build

## Run all backend tests
test-backend:
	cd backend && go test ./...

## Run all tests (alias)
test: test-backend
