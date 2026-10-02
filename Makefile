.PHONY: help env tidy fmt lint test test-race test-cover test-integration build up up-scaled down logs ps migrate-up migrate-down cleanup demo

GO      ?= go
COMPOSE ?= docker compose
API_URL ?= http://localhost:8081

help:
	@echo "env              - создать .env из .env.example"
	@echo "up               - поднять окружение (postgres + миграции + api + worker + web)"
	@echo "up-scaled        - то же, но с тремя воркерами (проверка горизонтального масштабирования)"
	@echo "down             - остановить окружение и удалить данные"
	@echo "logs             - логи всех сервисов"
	@echo "migrate-up       - применить миграции"
	@echo "migrate-down     - откатить последнюю миграцию"
	@echo "cleanup          - одноразовый процесс очистки истории проверок"
	@echo "demo             - создать демо-пользователя и несколько мониторов через API"
	@echo "build            - собрать бинарники локально"
	@echo "test             - юнит-тесты"
	@echo "test-race        - юнит-тесты с детектором гонок"
	@echo "test-cover       - юнит-тесты с отчётом о покрытии"
	@echo "test-integration - интеграционные тесты (нужен Docker: testcontainers)"
	@echo "lint             - golangci-lint"
	@echo "fmt              - gofmt -w по проекту"

env:
	@test -f .env || cp .env.example .env
	@echo ".env готов"

tidy:
	$(GO) mod tidy

fmt:
	gofmt -w ./cmd ./internal

lint:
	golangci-lint run --build-tags=integration ./...

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

test-cover:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out | tail -1

test-integration:
	$(GO) test -tags=integration -count=1 -timeout=10m ./internal/integrationtests/...

build:
	$(GO) build -o bin/api ./cmd/api
	$(GO) build -o bin/worker ./cmd/worker
	$(GO) build -o bin/cleanup ./cmd/cleanup

up: env
	$(COMPOSE) up --build -d
	@echo "Фронтенд:  http://localhost:8080"
	@echo "API:       $(API_URL)/healthz"

up-scaled: env
	$(COMPOSE) up --build -d --scale worker=3

down:
	$(COMPOSE) down -v

logs:
	$(COMPOSE) logs -f --tail=100

ps:
	$(COMPOSE) ps

migrate-up: env
	$(COMPOSE) run --rm migrate

# DSN собирается из .env, а не прошит в цели: иначе он разъедется
# с настройками окружения при первой же смене пароля.
migrate-down: env
	set -a; . ./.env; set +a; \
	$(COMPOSE) run --rm migrate -path=/migrations \
		-database=postgres://$$POSTGRES_USER:$$POSTGRES_PASSWORD@postgres:5432/$$POSTGRES_DB?sslmode=disable \
		down 1

cleanup: env
	$(COMPOSE) run --rm cleanup

demo:
	@bash scripts/demo.sh $(API_URL)
