.PHONY: dev down build test migrate seed logs-api logs-worker lint

dev: .env
	docker compose up --build

down:
	docker compose down

build:
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker

test:
	go test ./... -v -cover

migrate:
	cat db/migrations/*.sql | docker compose exec -T postgres psql -U scraper -d tokopedia_scraper -f /dev/stdin

seed:
	curl -s -X POST http://localhost:8080/v1/scrape/tokopedia/search \
		-H "Content-Type: application/json" \
		-d '{"keyword":"laptop gaming","max_items":30,"sort_by":"price_asc"}' | jq .

logs-api:
	docker compose logs -f api

logs-worker:
	docker compose logs -f worker

lint:
	golangci-lint run ./...

.env:
	cp .env.example .env
