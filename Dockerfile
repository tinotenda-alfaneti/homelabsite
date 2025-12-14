# syntax=docker/dockerfile:1.6

FROM golang:1.22-bookworm AS builder
WORKDIR /app

COPY go.mod ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod CGO_ENABLED=0 GOOS=linux go build -o /out/homelabsite .

FROM gcr.io/distroless/base-debian12
WORKDIR /srv

COPY --from=builder /out/homelabsite /usr/local/bin/homelabsite
COPY config /srv/config

EXPOSE 8080
ENV PORT=8080

ENTRYPOINT ["/usr/local/bin/homelabsite"]
