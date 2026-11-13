#!/usr/bin/env bash
# Переключение управляемой деградации API для демонстрации SLI, SLO и алертов.
#
# Использование:
#   scripts/chaos.sh status                    # текущие значения
#   scripts/chaos.sh errors 0.3                # доля ответов 503 на /api/
#   scripts/chaos.sh latency 800ms 1           # задержка и доля запросов с ней
#   scripts/chaos.sh readyz on|off             # отказ проверки готовности
#   scripts/chaos.sh goroutines 10             # утечка: 10 горутин на запрос
#   scripts/chaos.sh memhog 8                  # рост: 8 MiB на запрос
#   scripts/chaos.sh off                       # выключить всё
#
# По умолчанию работает с minikube. MODE=compose — то же в docker compose.

set -euo pipefail

MODE="${MODE:-k8s}"
NAMESPACE="${NAMESPACE:-sentinel}"

VARS=(CHAOS_ERROR_RATE CHAOS_LATENCY CHAOS_LATENCY_RATE CHAOS_READYZ_FAIL CHAOS_GOROUTINE_LEAK CHAOS_MEM_HOG_MB)

usage() {
    sed -n '2,14p' "$0" >&2
    exit 1
}

apply() {
    case "$MODE" in
    k8s)
        kubectl set env "deploy/api" -n "$NAMESPACE" "$@"
        kubectl rollout status "deploy/api" -n "$NAMESPACE" --timeout=90s
        ;;
    compose)
        # compose не умеет менять env на лету: пересоздаём контейнер api
        # с новыми значениями, остальные сервисы не трогаем.
        env "$@" docker compose up -d --no-deps api
        ;;
    *)
        echo "Неизвестный MODE=$MODE, ожидается k8s или compose" >&2
        exit 1
        ;;
    esac
}

status() {
    case "$MODE" in
    k8s)
        kubectl set env "deploy/api" -n "$NAMESPACE" --list |
            grep -E "^(CHAOS_)" || echo "деградация выключена: переменных CHAOS_* нет"
        ;;
    compose)
        docker compose exec api env | grep -E "^CHAOS_" || echo "деградация выключена"
        ;;
    esac
}

case "${1:-}" in
status)
    status
    ;;
errors)
    [[ $# -eq 2 ]] || usage
    apply "CHAOS_ERROR_RATE=$2"
    ;;
latency)
    [[ $# -eq 3 ]] || usage
    apply "CHAOS_LATENCY=$2" "CHAOS_LATENCY_RATE=$3"
    ;;
readyz)
    case "${2:-}" in
    on) apply "CHAOS_READYZ_FAIL=true" ;;
    off) apply "CHAOS_READYZ_FAIL=false" ;;
    *) usage ;;
    esac
    ;;
goroutines)
    [[ $# -eq 2 ]] || usage
    apply "CHAOS_GOROUTINE_LEAK=$2"
    ;;
memhog)
    [[ $# -eq 2 ]] || usage
    apply "CHAOS_MEM_HOG_MB=$2"
    ;;
off)
    if [[ "$MODE" == "k8s" ]]; then
        # Минус в конце имени убирает переменную из Deployment.
        apply "${VARS[@]/%/-}"
    else
        apply "CHAOS_ERROR_RATE=0" "CHAOS_LATENCY=0s" \
            "CHAOS_LATENCY_RATE=0" "CHAOS_READYZ_FAIL=false" \
            "CHAOS_GOROUTINE_LEAK=0" "CHAOS_MEM_HOG_MB=0"
    fi
    ;;
*)
    usage
    ;;
esac
