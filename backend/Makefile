.PHONY: run tidy test migrate-up migrate-fresh

run:
	go run ./cmd/api

tidy:
	go mod tidy

test:
	go test ./...

migrate-up:
	go run ./cmd/migrate up

# DESTRUCTIVE: drops the public schema of the database in .env, then re-migrates.
migrate-fresh:
	go run ./cmd/migrate fresh -yes
