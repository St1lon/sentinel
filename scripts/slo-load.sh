#!/usr/bin/env bash
# Фоновая нагрузка на API: на ней считаются request-based SLI.
# Использование: scripts/slo-load.sh [BASE_URL] [RPS]
#
# Нужен демо-пользователь: сначала scripts/demo.sh.

set -euo pipefail

BASE_URL="${1:-http://localhost:8081}"
RPS="${2:-5}"
EMAIL="${DEMO_EMAIL:-demo@sentinel.local}"
PASSWORD="${DEMO_PASSWORD:-demo-password-123}"

PAUSE=$(awk -v rps="$RPS" 'BEGIN { printf "%.3f", 1 / rps }')

login() {
    curl -sS -X POST "$BASE_URL/api/v1/auth/login" \
        -H 'Content-Type: application/json' \
        -d "$(printf '{"email":"%s","password":"%s"}' "$EMAIL" "$PASSWORD")" |
        sed -n 's/.*"token":"\([^"]*\)".*/\1/p'
}

TOKEN=$(login || true)

if [[ -z "$TOKEN" ]]; then
    echo "Не удалось войти как $EMAIL на $BASE_URL. Сначала scripts/demo.sh" >&2
    exit 1
fi

ok=0
failed=0

report() {
    printf '\nЗапросов: успешных %d, с ошибкой %d\n' "$ok" "$failed"
    exit 0
}

trap report INT TERM

echo "Нагрузка на $BASE_URL, около $RPS запросов в секунду. Остановить — Ctrl+C."

while true; do
    for path in /api/v1/monitors /api/v1/me; do
        code=$(curl -sS -o /dev/null -w '%{http_code}' \
            -H "Authorization: Bearer $TOKEN" "$BASE_URL$path" || echo 000)

        if [[ "$code" == 2* ]]; then
            ok=$((ok + 1))
        else
            failed=$((failed + 1))
        fi

        # Токен мог истечь, а api — перезапуститься с новым секретом.
        if [[ "$code" == "401" ]]; then
            TOKEN=$(login || true)
        fi
    done

    sleep "$PAUSE"
done
