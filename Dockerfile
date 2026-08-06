FROM golang:1.26.4-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/main .

FROM debian:bookworm-slim AS migrate

ARG TARGETARCH
ARG MIGRATE_VERSION=v4.19.1

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates curl tar && \
    curl -fsSL "https://github.com/golang-migrate/migrate/releases/download/${MIGRATE_VERSION}/migrate.linux-${TARGETARCH}.tar.gz" | tar -xz && \
    mv migrate /usr/local/bin/migrate && \
    rm -rf /var/lib/apt/lists/*

FROM debian:bookworm-slim AS runtime

WORKDIR /app

ENV GIN_MODE=release

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates curl && \
    rm -rf /var/lib/apt/lists/* && \
    groupadd --system monierave && \
    useradd --system --gid monierave --home-dir /app --no-create-home monierave

COPY --from=builder --chown=monierave:monierave /out/main /app/main
COPY --from=migrate /usr/local/bin/migrate /usr/local/bin/migrate
COPY --chown=monierave:monierave db/migration /app/db/migration

EXPOSE 8080

USER monierave

CMD ["/app/main", "api"]
