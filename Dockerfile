# syntax=docker/dockerfile:1.6

FROM golang:1.22-bookworm AS builder
WORKDIR /app

# Install build dependencies for SQLite
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    libc6-dev \
    && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum* ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
# Enable CGO for SQLite support
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-linkmode external -extldflags "-static"' -o /out/homelabsite .

FROM gcr.io/distroless/base-debian12
WORKDIR /srv

COPY --from=builder /out/homelabsite /usr/local/bin/homelabsite
COPY config /srv/config

EXPOSE 8080
ENV PORT=8080

ENTRYPOINT ["/usr/local/bin/homelabsite"]
