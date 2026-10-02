-include .env

DATABASE_URL ?= postgres://adicara:adicara-local@postgres:5432/adicara?sslmode=disable

.PHONY: dev down test lint build migrate-up migrate-down

dev:
	docker compose up --build

down:
	docker compose down

test:
	cd frontend && npm test
	cd backend && go test ./...

lint:
	cd frontend && npm run format:check && npm run check
	cd backend && gofmt -l . && go vet ./...

build:
	cd frontend && npm run build
	cd backend && go build ./cmd/api

migrate-up:
	docker compose run --rm migrate

migrate-down:
	docker compose run --rm migrate -path=/migrations -database="$(DATABASE_URL)" down 1
