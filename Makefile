.PHONY: help env tidy fmt lint test test-race test-cover test-integration build up up-scaled down logs ps migrate-up migrate-down cleanup demo load chaos metrics sli image-prober prober-up monitoring-up monitoring-down rules-up rules-reload alerts-lint image-sink sink-up sink-logs alerts-up alertmanager-ui vmalert-ui grafana-ui dashboards-up vm-ui

GO           ?= go
COMPOSE      ?= docker compose
API_URL      ?= http://localhost:8081
METRICS_URL  ?= http://localhost:9091/metrics
SLI_METRICS_URL ?= http://localhost:9092/metrics
LOAD_RPS     ?= 5
VM_RELEASE   ?= vm-single
VM_NAMESPACE ?= monitoring

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
	@echo "load             - фоновая нагрузка на API (на ней считаются SLI)"
	@echo "chaos            - управляемая деградация: make chaos ARGS=\"errors 0.3\""
	@echo "metrics          - бизнес-метрики api с поднятого окружения"
	@echo "sli              - метрики blackbox-пробера (SLI) с поднятого окружения"
	@echo "image-prober     - собрать образ пробера внутри minikube"
	@echo "prober-up        - собрать образ и развернуть пробер в minikube"
	@echo "monitoring-up    - поставить Victoria Metrics в minikube (helm)"
	@echo "monitoring-down  - снести Victoria Metrics"
	@echo "rules-up         - поставить vmalert с recording rules и алертами"
	@echo "rules-reload     - применить изменённые правила и перезапустить vmalert"
	@echo "alerts-lint      - проверить правила и alertmanager.yml (promtool/amtool)"
	@echo "sink-up          - собрать образ и развернуть notifier-sink (приёмник уведомлений)"
	@echo "sink-logs        - логи notifier-sink: тела уведомлений Alertmanager"
	@echo "alerts-up        - Alertmanager + правила + связь vmalert с Alertmanager"
	@echo "alertmanager-ui  - проброс порта к Alertmanager (список алертов)"
	@echo "grafana-ui       - проброс порта к Grafana (дашборд Sentinel SLO)"
	@echo "vmalert-ui       - проброс порта к vmalert (статус правил)"
	@echo "dashboards-up    - датасорс Victoria Metrics и дашборд SLO в Grafana"
	@echo "vm-ui            - проброс порта к Victoria Metrics (vmui)"
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
	$(GO) build -o bin/prober ./cmd/prober
	$(GO) build -o bin/notifier-sink ./cmd/notifier-sink

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

load:
	@bash scripts/slo-load.sh $(API_URL) $(LOAD_RPS)

chaos:
	@bash scripts/chaos.sh $(ARGS)

metrics:
	@curl -fsS $(METRICS_URL) | grep '^sentinel_' || \
		echo "метрик нет: окружение не поднято или порт не проброшен (API_METRICS_PORT)"

sli:
	@curl -fsS $(SLI_METRICS_URL) | grep '^sentinel_sli' || \
		echo "метрик пробера нет: окружение не поднято или порт не проброшен (PROBER_METRICS_PORT)"

# Чарт ставится upgrade --install, чтобы цель была идемпотентной.
monitoring-up:
	helm repo add vm https://victoriametrics.github.io/helm-charts/
	helm repo update vm
	helm upgrade --install $(VM_RELEASE) vm/victoria-metrics-single \
		--namespace $(VM_NAMESPACE) --create-namespace \
		-f k8s/monitoring/vm-single-values.yaml --wait

monitoring-down:
	helm uninstall $(VM_RELEASE) --namespace $(VM_NAMESPACE)

# Образ собирается внутри minikube: рантайм containerd, docker-env не подойдёт.
image-prober:
	docker build -t sentinel/prober:dev \
		--build-arg SERVICE=prober --build-arg VERSION=k8s \
		-f deployments/Dockerfile .
	minikube image load sentinel/prober:dev

prober-up: image-prober
	kubectl apply -f k8s/01-configmap.yaml -f k8s/02-secret.yaml -f k8s/70-prober.yaml
	kubectl rollout status deploy/prober -n sentinel

# Датасорс читается Grafana только при старте, поэтому рестарт, а не apply.
dashboards-up:
	kubectl apply -f k8s/logging/30-grafana.yaml -f k8s/logging/31-grafana-dashboards.yaml
	kubectl rollout restart deploy/grafana -n logging
	kubectl rollout status deploy/grafana -n logging
	@echo "Grafana: make grafana-ui — дашборд Sentinel SLO в папке Sentinel"

rules-up:
	kubectl apply -f k8s/monitoring/sli-rules.yaml -f k8s/monitoring/alerts.yaml -f k8s/monitoring/vmalert.yaml
	kubectl rollout status deploy/vmalert -n $(VM_NAMESPACE)

# Перечитать правила без пересоздания пода.
rules-reload:
	kubectl apply -f k8s/monitoring/sli-rules.yaml -f k8s/monitoring/alerts.yaml
	kubectl rollout restart deploy/vmalert -n $(VM_NAMESPACE)

# Правила и alertmanager.yml лежат внутри ConfigMap, поэтому сначала они
# извлекаются во временную папку, потом проверяются инструментами.
alerts-lint:
	@bash scripts/alerts-lint.sh

# Образ собирается внутри minikube: рантайм containerd, docker-env не подойдёт.
image-sink:
	docker build -t sentinel/notifier-sink:dev \
		--build-arg SERVICE=notifier-sink --build-arg VERSION=k8s \
		-f deployments/Dockerfile .
	minikube image load sentinel/notifier-sink:dev

sink-up: image-sink
	kubectl apply -f k8s/monitoring/notifier-sink.yaml
	kubectl rollout status deploy/notifier-sink -n $(VM_NAMESPACE)

# Здесь и появляются уведомления: JSON-тело каждого письма от Alertmanager.
sink-logs:
	kubectl logs -f deploy/notifier-sink -n $(VM_NAMESPACE)

# Полная связка: приёмник -> Alertmanager -> vmalert перечитывает алерты.
alerts-up: sink-up
	kubectl apply -f k8s/monitoring/alertmanager.yaml -f k8s/monitoring/alerts.yaml
	kubectl rollout status deploy/alertmanager -n $(VM_NAMESPACE)
	kubectl rollout restart deploy/vmalert -n $(VM_NAMESPACE)
	kubectl rollout status deploy/vmalert -n $(VM_NAMESPACE)

alertmanager-ui:
	@echo "Alertmanager: http://localhost:9093"
	kubectl port-forward svc/alertmanager -n $(VM_NAMESPACE) 9093:9093

grafana-ui:
	@echo "Grafana: http://localhost:30300"
	kubectl port-forward svc/grafana -n logging 30300:3000

vmalert-ui:
	@echo "vmalert: http://localhost:8880"
	kubectl port-forward svc/vmalert -n $(VM_NAMESPACE) 8880:8880

vm-ui:
	@echo "vmui: http://localhost:8428/vmui"
	kubectl port-forward svc/$(VM_RELEASE)-victoria-metrics-single-server -n $(VM_NAMESPACE) 8428:8428
