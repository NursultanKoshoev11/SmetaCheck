.PHONY: test build up down logs health format

test:
	go test ./...

build:
	go build -o bin/crypto-analyzer ./cmd/analyzer

format:
	gofmt -w $$(find cmd internal migrations -name '*.go')

up:
	docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f analyzer

health:
	curl -fsS http://localhost:$${HTTP_PORT:-8085}/health
