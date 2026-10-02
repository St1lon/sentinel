#!/usr/bin/env bash
# Наполняет окружение демо-данными через HTTP API.
# Использование: scripts/demo.sh [BASE_URL]

set -euo pipefail

BASE_URL="${1:-http://localhost:8081}"
EMAIL="${DEMO_EMAIL:-demo@sentinel.local}"
PASSWORD="${DEMO_PASSWORD:-demo-password-123}"

say() { printf '\n==> %s\n' "$1"; }

say "Регистрация $EMAIL на $BASE_URL"
REGISTER_BODY=$(printf '{"email":"%s","password":"%s"}' "$EMAIL" "$PASSWORD")

RESPONSE=$(curl -sS -X POST "$BASE_URL/api/v1/auth/register" \
    -H 'Content-Type: application/json' \
    -d "$REGISTER_BODY" || true)

# Пользователь мог быть создан предыдущим запуском — тогда просто входим.
if ! grep -q '"token"' <<<"$RESPONSE"; then
    echo "Регистрация не прошла (возможно, пользователь уже есть), пробуем вход."
    RESPONSE=$(curl -sS -X POST "$BASE_URL/api/v1/auth/login" \
        -H 'Content-Type: application/json' \
        -d "$REGISTER_BODY")
fi

TOKEN=$(sed -n 's/.*"token":"\([^"]*\)".*/\1/p' <<<"$RESPONSE")
SLUG=$(sed -n 's/.*"status_page_slug":"\([^"]*\)".*/\1/p' <<<"$RESPONSE")

if [[ -z "$TOKEN" ]]; then
    echo "Не удалось получить токен. Ответ API:" >&2
    echo "$RESPONSE" >&2
    exit 1
fi

create_monitor() {
    local name="$1" target="$2" interval="$3" expected="$4" public="$5"

    curl -sS -o /dev/null -w "  %{http_code}  $name\n" \
        -X POST "$BASE_URL/api/v1/monitors" \
        -H 'Content-Type: application/json' \
        -H "Authorization: Bearer $TOKEN" \
        -d "$(printf '{"name":"%s","target":"%s","interval_seconds":%s,"timeout_seconds":5,"expected_status":%s,"failure_threshold":2,"is_public":%s}' \
            "$name" "$target" "$interval" "$expected" "$public")"
}

say "Создание мониторов"
create_monitor "Example.com"            "https://example.com"                      30  200 true
create_monitor "GitHub API"             "https://api.github.com"                  60  200 true
create_monitor "Httpbin 200"            "https://httpbin.org/status/200"           60  200 true
create_monitor "Httpbin 503 (упадёт)"   "https://httpbin.org/status/503"           30  200 true
create_monitor "Несуществующий домен"   "https://this-domain-does-not-exist.invalid" 60 200 false

say "Готово"
echo "  email:            $EMAIL"
echo "  пароль:           $PASSWORD"
echo "  статус-страница:  http://localhost:8080/status.html?slug=$SLUG"
echo
echo "Через 30–60 секунд воркер успеет проверить цели и на странице появятся данные."
