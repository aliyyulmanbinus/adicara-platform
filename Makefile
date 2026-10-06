-include .env

.PHONY: dev down test lint build migrate-up migrate-fresh

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

# The API migrates on start (MIGRATE_ON_START); these run the same code on demand.
migrate-up:
	docker compose run --rm --no-deps --entrypoint /adicara-migrate backend up

# DESTRUCTIVE: wipes the compose database, then re-applies every migration.
migrate-fresh:
	docker compose run --rm --no-deps --entrypoint /adicara-migrate backend fresh -yes
