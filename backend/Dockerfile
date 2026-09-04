# Stage 1: Build
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install git and ca-certificates
RUN apk add --no-cache git ca-certificates tzdata

# Cache go modules
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy all backend source
COPY . .
RUN go mod tidy

# Build API binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/api ./cmd/api

# Build Worker binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/worker ./cmd/worker

# Stage 2: Production API Image
FROM alpine:3.19 AS api
RUN apk add --no-cache ca-certificates tzdata curl
WORKDIR /app
COPY --from=builder /bin/api /app/api
EXPOSE 8080
ENTRYPOINT ["/app/api"]

# Stage 3: Production Worker Image
FROM alpine:3.19 AS worker
RUN apk add --no-cache ca-certificates tzdata curl
WORKDIR /app
COPY --from=builder /bin/worker /app/worker
EXPOSE 8081
ENTRYPOINT ["/app/worker"]
