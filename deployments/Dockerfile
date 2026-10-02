FROM golang:1.24-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG SERVICE=api
ARG VERSION=dev

RUN CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -ldflags="-s -w" \
        -o /out/app \
        ./cmd/${SERVICE}

FROM alpine:3.21 AS runtime

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 -h /app sentinel

COPY --from=builder /out/app /usr/local/bin/app

USER sentinel
WORKDIR /app

ARG VERSION=dev
ENV APP_VERSION=${VERSION}

ENTRYPOINT ["/usr/local/bin/app"]
