# Multi-stage build
# PHP equivalent: Dockerfile typically uses php:fpm image directly

# Stage 1: Build
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files first (better caching)
COPY go.mod ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server main.go

# Stage 2: Runtime
# PHP equivalent: No equivalent - PHP always needs runtime
FROM alpine:latest

# Add ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy only the binary - no runtime needed!
# PHP: Would need to copy entire vendor/ directory and php runtime
COPY --from=builder /app/server .

# Non-root user for security
RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8080

CMD ["./server"]
