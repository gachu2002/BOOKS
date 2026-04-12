.PHONY: run up down test build lint batch-reports projector-once projector-rebuild

run:
	go run ./cmd/api

build:
	go build ./...

test:
	go test ./...

lint:
	golangci-lint run

batch-reports:
	go run ./cmd/batch-reports

projector-once:
	PROJECTOR_MODE=once go run ./cmd/projector

projector-rebuild:
	PROJECTOR_MODE=rebuild go run ./cmd/projector

up:
	docker compose up -d postgres

down:
	docker compose down
