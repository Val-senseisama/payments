# Build Stage
FROM golang:alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy dependency locks
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build production binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/payments ./cmd/main.go

# Runtime Stage
FROM alpine:3.19 AS runner

WORKDIR /app

# Install ca-certificates and curl for health checks
RUN apk add --no-cache ca-certificates curl

# Copy binary from builder
COPY --from=builder /app/bin/payments /app/payments

# Copy migration SQL files and web frontend dashboard
COPY --from=builder /app/cmd/migrations/sql ./cmd/migrations/sql
COPY --from=builder /app/web ./web
COPY --from=builder /app/.env* ./

EXPOSE 8080

CMD ["/app/payments"]
