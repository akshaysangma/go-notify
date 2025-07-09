# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod tidy
COPY . .
# Build the Go app
# CGO_ENABLED=0 is used to build a statically linked binary.
# -ldflags="-w -s" strips debugging information, which reduces the binary size.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/go-notify ./cmd/server

# Runner stage
FROM alpine:latest

WORKDIR /app
# Copy required files to run from build stage
COPY --from=builder /app/go-notify /app/go-notify
COPY --from=builder /app/config.yaml /app/config.yaml

# Expose port 8080 to the outside world
EXPOSE 8080

CMD ["/app/go-notify"]