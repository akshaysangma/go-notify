export DATABASE_URL="postgresql://user:password@127.0.0.1:5432/go_notify_db?sslmode=disable"


.PHONY: infra-up infra-down migrate-up migrate-down generate build mock-data docker-build docker-run

docker-run: infra-up docker-build
	@echo "Starting app in docker..."
	docker run --rm -p 8080:8080 \
  --network=go-notify_default \
  -e DATABASE_CONNECTION_STRING="postgresql://user:password@postgres:5432/go_notify_db?sslmode=disable" \
  -e REDIS_ADDRESS="redis:6379" \
  go-notify-app
	
infra-up: ## Start all infrastructure services (PostgreSQL, Redis)
	@echo "Starting infrastructure services..."
	docker-compose up -d

infra-down: ## Stop and remove all infrastructure services
	@echo "Stopping and removing infrastructure services..."
	docker-compose down -v

migrate-up: ## Apply all pending database migrations
	@echo "Applying database migrations..."
	goose -dir sql/schema postgres "$(DATABASE_URL)" up

migrate-down: ## Rollback the last applied database migration
	@echo "Rolling back last database migration..."
	goose -dir sql/schema postgres "$(DATABASE_URL)" down

generate: ## Run sqlc to generate Go code from SQL queries
	@echo "Running sqlc generate..."
	sqlc generate
	@echo "sqlc generate complete."

build: ## Build all Go service binaries (e.g., api-gateway)
	@echo "Building Go binaries..."
	go build -o bin/go-notify cmd/server/main.go # Builds server binary

docker-build:
	@echo "Building Server Image in Docker.."
	docker build -t go-notify-app .

mock-data: # Insert some data to postgres DB
	@echo "Mocking pending data..."
	docker-compose exec postgres psql -U user -d go_notify_db -c "INSERT INTO notifications.messages (content, recipient_phone_number, created_at) VALUES ('Hello from GoDistro Mentor! This is message 1.', '+15550000001', NOW() - INTERVAL '5 minutes'), ('Test message 2 for automatic sending.', '+15550000002', NOW() - INTERVAL '4 minutes'), ('Another unsent message, ready to go!', '+15550000003', NOW() - INTERVAL '3 minutes'), ('Message 4 - will be picked up by next cycle.', '+15550000004', NOW() - INTERVAL '2 minutes'), ('Final unsent message for this batch.', '+15550000005', NOW() - INTERVAL '1 minute');"