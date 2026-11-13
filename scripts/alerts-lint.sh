#!/usr/bin/env bash
# Проверка правил алертов и конфига Alertmanager без кластера.
#
# В манифестах правила лежат внутри ConfigMap, поэтому сначала извлекаем
# вложенные YAML во временную папку, затем проверяем их настоящими
# инструментами: promtool — правила Prometheus/vmalert, amtool — alertmanager.yml.
#
# Файлы подаются в контейнер через stdin, а не bind-mount: монтирование
# свежих каталогов Docker Desktop на Windows отдаёт недетерминированно
# (ls внутри контейнера файлы видит, stat — уже нет), а канал в /tmp самого
# контейнера работает всегда и на любой ОС.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MONITORING_DIR="$SCRIPT_DIR/../k8s/monitoring"

PROM_IMAGE="${PROM_IMAGE:-prom/prometheus:v3.5.0}"
AM_IMAGE="${AM_IMAGE:-prom/alertmanager:v0.28.1}"

workdir="$(mktemp -d)"
trap 'rm -rf "$workdir"' EXIT

# Извлечение блоков data.<key>: | из ConfigMap: ключ становится именем файла,
# каждая строка содержимого теряет отступ в 4 пробела. Разделитель документов
# (--- и ...) обязателен к обработке: в манифесте с несколькими документами
# после data: идут Service и Deployment, и их 4-пробельные строки затекли бы
# в извлечённый конфиг.
extract() {
    awk -v dir="$1" '
        /^---/ || /^\.\.\./ {
            if (file != "") close(file)
            file = ""; inmap = 0
            next
        }
        /^data:/ { inmap = 1; next }
        inmap && /^  [^ ].*: \|$/ {
            if (file != "") close(file)
            line = $0
            sub(/^  /, "", line)
            sub(/: \|$/, "", line)
            file = dir "/" line
            next
        }
        inmap && file != "" && /^    / { print substr($0, 5) >> file }
    ' "$2"
}

for manifest in sli-rules.yaml alerts.yaml alertmanager.yaml; do
    extract "$workdir" "$MONITORING_DIR/$manifest"
done

# Проверяемый файл уходит в stdin, внутри контейнера сохраняется в /tmp
# и проверяется оттуда. Никаких путей хоста в аргументах — Git Bash такое
# не искажает, а docker различий между ОС не замечает.
check_rules() {
    docker run -i --rm --entrypoint sh "$PROM_IMAGE" \
        -c 'cat > /tmp/rules.yaml && promtool check rules /tmp/rules.yaml' < "$1"
}

check_amconfig() {
    docker run -i --rm --entrypoint sh "$AM_IMAGE" \
        -c 'cat > /tmp/amconfig.yml && /bin/amtool check-config /tmp/amconfig.yml' < "$1"
}

echo "== promtool: recording rules"
check_rules "$workdir/sentinel-sli.yaml"

echo "== promtool: правила алертов"
check_rules "$workdir/sentinel-alerts.yaml"

echo "== amtool: конфиг Alertmanager"
check_amconfig "$workdir/alertmanager.yml"

echo "OK: правила и конфиг Alertmanager валидны"
