-include .env
export

MIGRATIONS_DIR := migrations

.PHONY: up down migrate-up migrate-down run dev build

up:
	docker compose up -d

down:
	docker compose down

migrate-up:
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" down 1

build:
	go build ./...

run:
	go run ./cmd/server

dev:
	cd web && npm run dev
