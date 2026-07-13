FROM golang:1.23-alpine AS builder
WORKDIR /src

RUN apk add --no-cache ca-certificates git
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/crypto-analyzer ./cmd/analyzer

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget \
    && addgroup -S app \
    && adduser -S -G app app

WORKDIR /app
COPY --from=builder /out/crypto-analyzer /usr/local/bin/crypto-analyzer

USER app
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/health >/dev/null || exit 1

ENTRYPOINT ["/usr/local/bin/crypto-analyzer"]
