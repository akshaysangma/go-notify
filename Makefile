# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=go-notify
BINARY_UNIX=$(BINARY_NAME)_unix

# Docker parameters
DOCKER_COMPOSE=docker-compose
DOCKER=docker

# SQLC
SQLC=sqlc

# Goose
GOOSE=goose

all: help

help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  up               - Start the local infrastructure (Postgres & Redis)"
	@echo "  down             - Stop the local infrastructure"
	@echo "  build            - Build the application"
	@echo "  run              - Run the application"
	@echo "  test             - Run tests"
	@echo "  clean            - Clean up build artifacts"
	@echo "  docker-build-run - Build and run the application in a Docker container"
	@echo "  sqlc-generate    - Generate Go code from SQL queries"
	@echo "  migrate-up       - Apply all available database migrations"
	@echo "  migrate-down     - Roll back the last database migration"

## Infrastructure
up:
	@echo "Starting local infrastructure..."
	$(DOCKER_COMPOSE) up -d

down:
	@echo "Stopping local infrastructure..."
	$(DOCKER_COMPOSE) down


## Build, Run, Test
build:
	@echo "Building the application..."
	CGO_ENABLED=0 GOOS=linux $(GOBUILD) -ldflags="-w -s" -o bin/$(BINARY_NAME) ./cmd/server

run:
	@echo "Running the application..."
	./bin/$(BINARY_NAME)

test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

clean:
	@echo "Cleaning up..."
	$(GOCLEAN)
	rm -f bin/$(BINARY_NAME) bin/$(BINARY_UNIX)


## Database
sqlc-generate:
	@echo "Generating Go code from SQL..."
	$(SQLC) generate

migrate-up:
	@echo "Applying database migrations..."
	$(GOOSE) -dir ./sql/schema postgres "postgresql://user:password@localhost:5432/go_notify_db?sslmode=disable" up

migrate-down:
	@echo "Rolling back the last migration..."
	$(GOOSE) -dir ./sql/schema postgres "postgresql://user:password@localhost:5432/go_notify_db?sslmode=disable" down


## Docker
docker-build-run:
	@echo "Building and running the application in a Docker container..."
	$(DOCKER) build -t go-notify-app .
	$(DOCKER) run --rm -p 8080:8080 \
	--network=go-notify_default \
	-e DATABASE_CONNECTION_STRING="postgresql://user:password@postgres:5432/go_notify_db?sslmode=disable" \
	-e REDIS_ADDRESS="redis:6379" \
	go-notify-app