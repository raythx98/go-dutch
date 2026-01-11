# Build Stage
FROM golang:1.24.0-bookworm AS builder

RUN apt-get update && apt-get install -y ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . /app
RUN go build -o main server.go

# Run Stage
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/main .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/ratelimit.yaml .

EXPOSE 8080
CMD ["./main"]