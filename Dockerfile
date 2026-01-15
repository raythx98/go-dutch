# Build Stage
FROM golang:1.24.11-bookworm AS builder

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build application
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main server.go

# Run Stage
# Use distroless for a minimal, secure, non-root production image
# https://github.com/GoogleContainerTools/distroless
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /app/main .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/ratelimit.yaml .

# Distroless nonroot user ID is 65532
USER 65532:65532

EXPOSE 8080
CMD ["./main"]