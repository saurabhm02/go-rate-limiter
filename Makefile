.PHONY: help run build test test-integration test-concurrency test-all lint docker-up docker-down docker-prod-up docker-prod-down migrate examples-up loadtest loadtest-smoke

COMPOSE_FILE := deploy/docker/docker-compose.yml
COMPOSE_EXAMPLES := deploy/docker/docker-compose.examples.yml
COMPOSE_PROD := deploy/docker/docker-compose.prod.yml
K6_IMAGE ?= grafana/k6:0.57.0
K6_DIR := $(CURDIR)/load/k6
BASE_URL ?= http://host.docker.internal:8080
API_KEY ?= rl_demo_abc123xyz

help:
	@echo "Targets:"
	@echo "  run          Run the API server locally"
	@echo "  build        Build server binary"
	@echo "  test         Run unit tests"
	@echo "  test-concurrency  Run concurrency tests with race detector"
	@echo "  test-integration  Run integration tests (requires Docker)"
	@echo "  test-all     Run unit + concurrency + integration"
	@echo "  lint         Run go vet"
	@echo "  docker-up    Start postgres, redis, prometheus, and API"
	@echo "  docker-down  Stop docker services"
	@echo "  docker-prod-up   Production compose overlay + migrate"
	@echo "  examples-up  Start full stack including example services (M6+)"
	@echo "  loadtest-smoke   Quick k6 smoke test (requires stack up)"
	@echo "  loadtest     k6 sustained load (1000 req/s, 30s)"

run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

test:
	go test ./test/unit/... ./test/concurrency/...

test-concurrency:
	go test -race -count=1 ./test/concurrency/...

test-integration:
	go test -tags=integration -count=1 -timeout=15m ./test/integration/...

test-all: test test-concurrency test-integration

lint:
	go vet ./...

docker-up:
	docker compose -f $(COMPOSE_FILE) up -d --build

docker-down:
	docker compose -f $(COMPOSE_FILE) down

docker-prod-up:
	docker compose -f $(COMPOSE_FILE) -f $(COMPOSE_PROD) up -d --build postgres redis
	docker compose -f $(COMPOSE_FILE) -f $(COMPOSE_PROD) run --rm migrate
	docker compose -f $(COMPOSE_FILE) -f $(COMPOSE_PROD) up -d ratelimit

docker-prod-down:
	docker compose -f $(COMPOSE_FILE) -f $(COMPOSE_PROD) down

examples-up:
	docker compose -f $(COMPOSE_FILE) -f $(COMPOSE_EXAMPLES) up -d --build

migrate:
	go run ./cmd/migrate

loadtest-smoke:
	docker run --rm -i \
		-e BASE_URL=$(BASE_URL) \
		-e API_KEY=$(API_KEY) \
		-v $(K6_DIR):/scripts \
		$(K6_IMAGE) run /scripts/check_smoke.js

loadtest:
	docker run --rm -i \
		-e BASE_URL=$(BASE_URL) \
		-e API_KEY=$(API_KEY) \
		-v $(K6_DIR):/scripts \
		$(K6_IMAGE) run /scripts/check_load.js
