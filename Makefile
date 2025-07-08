export DATABASE_URL="postgresql://user:password@127.0.0.1:5432/go_notify_db?sslmode=disable"


.PHONY: infra-up infra-down migrate-up migrate-down generate build

infra-up: ## Start all infrastructure services (PostgreSQL, Redis)
	@echo "Starting infrastructure services..."
	docker-compose up -d

down: ## Stop and remove all infrastructure services
	@echo "Stopping and removing infrastructure services..."
	docker-compose down -v

migrate-up: ## Apply all pending database migrations
	@echo "Applying database migrations..."
	goose -dir schema postgres "$(DATABASE_URL)" up

migrate-down: ## Rollback the last applied database migration
	@echo "Rolling back last database migration..."
	goose -dir schema postgres "$(DATABASE_URL)" down

generate: ## Run sqlc to generate Go code from SQL queries
	@echo "Running sqlc generate..."
	sqlc generate
	@echo "sqlc generate complete."

build: ## Build all Go service binaries (e.g., api-gateway)
	@echo "Building Go binaries..."
	go build -o bin/go-notify $(API_GATEWAY_CMD)/main.go # Builds api-gateway binary