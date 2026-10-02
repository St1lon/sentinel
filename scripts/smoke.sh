#!/usr/bin/env bash
# Сквозная проверка поднятого окружения через публичный HTTP API:
# регистрация → создание монитора → чтение → обновление → статистика →
# публичная статус-страница → удаление.
#
# Намеренно работает только с внешним контрактом, без обращения к БД:
# это проверка системы как чёрного ящика.
#
# Использование: scripts/smoke.sh [BASE_URL]

set -euo pipefail

BASE_URL="${1:-http://localhost:8081}"
EMAIL="smoke-$(date +%s)-$RANDOM@example.com"
PASSWORD="smoke-password-123"

pass() { printf '  ok    %s\n' "$1"; }
fail() { printf '  FAIL  %s\n' "$1" >&2; exit 1; }

json_field() { sed -n "s/.*\"$1\":\"\([^\"]*\)\".*/\1/p" <<<"$2"; }

echo "== Сквозная проверка $BASE_URL =="

# 1. Технические эндпоинты.
curl -fsS "$BASE_URL/healthz" >/dev/null || fail "/healthz"
pass "/healthz"

curl -fsS "$BASE_URL/readyz" >/dev/null || fail "/readyz"
pass "/readyz (БД доступна)"

# 2. Регистрация.
AUTH=$(curl -fsS -X POST "$BASE_URL/api/v1/auth/register" \
    -H 'Content-Type: application/json' \
    -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")

TOKEN=$(json_field token "$AUTH")
SLUG=$(json_field status_page_slug "$AUTH")

[[ -n "$TOKEN" ]] || fail "регистрация не вернула токен"
[[ -n "$SLUG" ]] || fail "регистрация не вернула слаг статус-страницы"
pass "регистрация и выдача токена"

AUTH_HEADER="Authorization: Bearer $TOKEN"

# 3. Запрос без токена должен отклоняться.
CODE=$(curl -s -o /dev/null -w '%{http_code}' "$BASE_URL/api/v1/monitors")
[[ "$CODE" == "401" ]] || fail "список мониторов без токена вернул $CODE вместо 401"
pass "защищённый эндпоинт требует токен"

# 4. Повторная регистрация того же email — конфликт.
CODE=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$BASE_URL/api/v1/auth/register" \
    -H 'Content-Type: application/json' \
    -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")
[[ "$CODE" == "409" ]] || fail "повторная регистрация вернула $CODE вместо 409"
pass "повторная регистрация отклоняется"

# 5. Вход.
LOGIN=$(curl -fsS -X POST "$BASE_URL/api/v1/auth/login" \
    -H 'Content-Type: application/json' \
    -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")
[[ -n "$(json_field token "$LOGIN")" ]] || fail "вход не вернул токен"
pass "вход по паролю"

# 6. Создание монитора (C из CRUD).
MONITOR=$(curl -fsS -X POST "$BASE_URL/api/v1/monitors" \
    -H 'Content-Type: application/json' -H "$AUTH_HEADER" \
    -d '{"name":"Smoke monitor","target":"https://example.com","interval_seconds":60,"timeout_seconds":5,"is_public":true}')

MONITOR_ID=$(json_field id "$MONITOR")
[[ -n "$MONITOR_ID" ]] || fail "создание монитора"
pass "создание монитора"

# 7. Чтение (R).
curl -fsS -H "$AUTH_HEADER" "$BASE_URL/api/v1/monitors/$MONITOR_ID" >/dev/null || fail "чтение монитора"
grep -q '"total":1' <<<"$(curl -fsS -H "$AUTH_HEADER" "$BASE_URL/api/v1/monitors")" || fail "список мониторов"
pass "чтение монитора и списка"

# 8. Обновление (U).
UPDATED=$(curl -fsS -X PATCH "$BASE_URL/api/v1/monitors/$MONITOR_ID" \
    -H 'Content-Type: application/json' -H "$AUTH_HEADER" \
    -d '{"name":"Smoke monitor renamed","interval_seconds":120}')
grep -q 'Smoke monitor renamed' <<<"$UPDATED" || fail "обновление монитора"
pass "частичное обновление монитора"

# 9. Валидация: таймаут не может быть больше интервала.
CODE=$(curl -s -o /dev/null -w '%{http_code}' -X PATCH "$BASE_URL/api/v1/monitors/$MONITOR_ID" \
    -H 'Content-Type: application/json' -H "$AUTH_HEADER" \
    -d '{"timeout_seconds":120}')
[[ "$CODE" == "400" ]] || fail "некорректное расписание вернуло $CODE вместо 400"
pass "валидация расписания"

# 10. Защита от SSRF: приватный адрес нельзя взять в мониторинг.
RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/monitors" \
    -H 'Content-Type: application/json' -H "$AUTH_HEADER" \
    -d '{"name":"SSRF probe","target":"http://169.254.169.254/latest/meta-data/"}')
grep -q 'TARGET_NOT_ALLOWED' <<<"$RESPONSE" || fail "адрес облачных метаданных не отклонён: $RESPONSE"
pass "приватные и служебные адреса отклоняются"

# 11. Статистика и инциденты.
curl -fsS -H "$AUTH_HEADER" "$BASE_URL/api/v1/monitors/$MONITOR_ID/stats" >/dev/null || fail "статистика"
curl -fsS -H "$AUTH_HEADER" "$BASE_URL/api/v1/monitors/$MONITOR_ID/incidents" >/dev/null || fail "инциденты"
curl -fsS -H "$AUTH_HEADER" "$BASE_URL/api/v1/monitors/$MONITOR_ID/checks" >/dev/null || fail "проверки"
pass "статистика, проверки, инциденты"

# 12. Публичная статус-страница без токена.
PAGE=$(curl -fsS "$BASE_URL/api/v1/public/status/$SLUG")
grep -q '"services"' <<<"$PAGE" || fail "статус-страница"
grep -q '"target"' <<<"$PAGE" && fail "статус-страница не должна раскрывать URL целей"
pass "публичная статус-страница без токена и без URL целей"

# 13. Удаление (D).
CODE=$(curl -s -o /dev/null -w '%{http_code}' -X DELETE -H "$AUTH_HEADER" \
    "$BASE_URL/api/v1/monitors/$MONITOR_ID")
[[ "$CODE" == "204" ]] || fail "удаление вернуло $CODE вместо 204"

CODE=$(curl -s -o /dev/null -w '%{http_code}' -H "$AUTH_HEADER" \
    "$BASE_URL/api/v1/monitors/$MONITOR_ID")
[[ "$CODE" == "404" ]] || fail "удалённый монитор доступен (код $CODE)"
pass "удаление монитора"

echo "== Все проверки пройдены =="
