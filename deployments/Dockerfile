FROM golang:1.24-alpine AS builder

WORKDIR /src

# Зависимости объявлены явно в go.mod/go.sum и скачиваются отдельным слоем,
# чтобы изменение кода не инвалидировало кэш модулей (фактор II).
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG SERVICE=api
ARG VERSION=dev

# CGO выключен: бинарник статический, его можно положить в минимальный образ.
RUN CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -ldflags="-s -w" \
        -o /out/app \
        ./cmd/${SERVICE}

FROM alpine:3.21 AS runtime

# ca-certificates нужны для проверки HTTPS-целей, tzdata — для корректных
# временных зон в логах и агрегатах.
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 -h /app sentinel

COPY --from=builder /out/app /usr/local/bin/app

# Процесс работает не от root: у него нет причин иметь лишние права.
USER sentinel
WORKDIR /app

ARG VERSION=dev
ENV APP_VERSION=${VERSION}

ENTRYPOINT ["/usr/local/bin/app"]
