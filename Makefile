test:
	go test ./...

build:
	docker compose build

up:
	docker compose up -d

logs:
	docker compose logs -f --tail=200

down:
	docker compose down
